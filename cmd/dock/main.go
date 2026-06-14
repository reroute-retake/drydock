// Command dock is the drydock host-side orchestrator.
//
// M1: setup/build/start/shell/stop. M2 adds addrepo (clone + stack detection ->
// per-stack image layers), sync, space switch, and update, and build now builds
// the generated Dockerfile. Pass --dry-run to print docker/git commands instead
// of running them. self-update/run remain stubs for later milestones.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/reroute-retake/drydock/internal/config"
	"github.com/reroute-retake/drydock/internal/gen"
	"github.com/reroute-retake/drydock/internal/paths"
	"github.com/reroute-retake/drydock/internal/proposal"
	"github.com/reroute-retake/drydock/internal/selfupdate"
	"github.com/reroute-retake/drydock/internal/stack"
	"github.com/reroute-retake/drydock/internal/telemetry"
	"github.com/reroute-retake/drydock/internal/vault"
	"github.com/reroute-retake/drydock/internal/version"
	"github.com/reroute-retake/drydock/internal/works"
)

var dryRun bool

// GitHub repo that hosts dock releases (used by self-update).
const (
	ghOwner = "reroute-retake"
	ghRepo  = "drydock"
)

const usage = `dock — drydock host-side orchestrator

Usage: dock [--dry-run] <command> [args]

Environment:
  setup <space>      Scaffold a space (repos/vault/works, manifest, .env) and make it active
  addrepo <url>      Clone a repo into the active space, detect its stack, update the manifest
  build              Generate gateway+compose+Dockerfile and build the dev image
  start              Start the space container + LiteLLM gateway; publish ports
  shell              Attach a shell to the running space container
  forward <h:c>      Ad-hoc port forward host:container                         [M3]
  stop               Stop the space containers
  sync               git pull --ff-only all repos in the active space
  space switch <s>   Switch the active space

Tickets (lifecycle state — the works/ artifact contract):
  work new <ticket>  Scaffold a ticket's works/ folder (ticket.md, state.yaml) and make it active
  work status [t]    Show a ticket's phase state
  work set <phase> <status> [--model m --note n --artifact f]   Record phase progress
  archive [ticket]   Copy a ticket's artifacts to the vault inbox and run vault:ingest [--clean]

Self-improvement (telemetry + human-gated skill proposals):
  telemetry [--ticket t] [--json]   Summarize captured session telemetry (input to retrospect)
  skill propose <skill> --content <file> [--source s --rationale r]   Propose a skill change (pending)
  skill list | show <id> | apply <id> [--to dir] | reject <id>        Review/apply proposals (human gate)

Maintenance:
  update             Refresh the active space's config/scaffolding (NOT the binary)
  self-update        Replace the dock binary from the latest GitHub release (checksum-verified)
  doctor             Diagnose the local environment
  version            Print version

Flags:
  --dry-run          Print docker/git commands instead of executing them
`

func main() {
	var rest []string
	for _, a := range os.Args[1:] {
		if a == "--dry-run" {
			dryRun = true
			continue
		}
		rest = append(rest, a)
	}
	if len(rest) == 0 {
		fmt.Print(usage)
		os.Exit(2)
	}
	cmd, args := rest[0], rest[1:]

	var err error
	switch cmd {
	case "version", "-v", "--version":
		fmt.Println(version.String())
	case "help", "-h", "--help":
		fmt.Print(usage)
	case "doctor":
		os.Exit(doctor())
	case "setup":
		err = cmdSetup(args)
	case "addrepo":
		err = cmdAddrepo(args)
	case "build":
		err = cmdBuild(args)
	case "start":
		err = cmdStart(args)
	case "shell":
		err = cmdShell(args)
	case "stop":
		err = cmdStop(args)
	case "sync":
		err = cmdSync(args)
	case "space":
		err = cmdSpace(args)
	case "update":
		err = cmdUpdate(args)
	case "work":
		err = cmdWork(args)
	case "archive":
		err = cmdArchive(args)
	case "telemetry":
		err = cmdTelemetry(args)
	case "skill":
		err = cmdSkill(args)
	case "forward":
		err = cmdForward(args)
	case "self-update":
		err = cmdSelfUpdate(args)
	case "run":
		stub(cmd, args)
	default:
		fmt.Fprintf(os.Stderr, "dock: unknown command %q\n\n", cmd)
		fmt.Print(usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "dock: "+err.Error())
		os.Exit(1)
	}
}

func stub(cmd string, args []string) {
	fmt.Printf("[stub] dock %s %s — not implemented yet (see design doc section 9)\n", cmd, strings.Join(args, " "))
}

// --- active space ---------------------------------------------------------

func setActive(space string) error {
	if err := os.MkdirAll(paths.StateHome(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(paths.StateHome(), "active"), []byte(space+"\n"), 0o644)
}

func activeSpace() (paths.Space, error) {
	b, err := os.ReadFile(filepath.Join(paths.StateHome(), "active"))
	if err != nil {
		return paths.Space{}, fmt.Errorf("no active space; run 'dock setup <space>' first")
	}
	name := strings.TrimSpace(string(b))
	if name == "" {
		return paths.Space{}, fmt.Errorf("active space is empty; run 'dock setup <space>'")
	}
	return paths.For(name), nil
}

func loadActive() (paths.Space, *config.Manifest, error) {
	sp, err := activeSpace()
	if err != nil {
		return paths.Space{}, nil, err
	}
	m, err := config.Load(sp.Manifest())
	if err != nil {
		return paths.Space{}, nil, err
	}
	return sp, m, nil
}

// --- commands --------------------------------------------------------------

func cmdSetup(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: dock setup <space>")
	}
	space := args[0]
	sp := paths.For(space)
	for _, d := range []string{sp.Repos, sp.Vault, sp.Works, sp.Drydock} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	if _, err := os.Stat(sp.Manifest()); os.IsNotExist(err) {
		if err := os.WriteFile(sp.Manifest(), config.Template(space), 0o644); err != nil {
			return err
		}
		fmt.Println("created", sp.Manifest())
	} else {
		fmt.Println("kept existing", sp.Manifest())
	}
	if _, err := os.Stat(sp.Env()); os.IsNotExist(err) {
		if err := os.WriteFile(sp.Env(), []byte(defaultEnv), 0o600); err != nil {
			return err
		}
		fmt.Println("created", sp.Env(), "(fill in keys — gitignored)")
	} else {
		fmt.Println("kept existing", sp.Env())
	}
	if err := setActive(space); err != nil {
		return err
	}
	fmt.Printf("space %q ready at %s (now active)\n", space, sp.Root)
	fmt.Println("next: 'dock addrepo <url>', then 'dock build' and 'dock start'")
	return nil
}

const defaultEnv = `# drydock space secrets — NEVER commit (P12/C8).
ANTHROPIC_API_KEY=
OPENAI_API_KEY=
LITELLM_MASTER_KEY=sk-drydock-local
`

func cmdAddrepo(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: dock addrepo <git-url>")
	}
	url := args[0]
	sp, m, err := loadActive()
	if err != nil {
		return err
	}
	name := repoName(url)
	dest := filepath.Join(sp.Repos, name)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		if err := run("git", "clone", url, dest); err != nil {
			return err
		}
	} else {
		fmt.Println("repo already present:", dest)
	}
	res := stack.Detect(dest) // empty under --dry-run (clone skipped)
	if !repoKnown(m, url) {
		m.Repos = append(m.Repos, config.Repo{URL: url, Stack: res.Stack})
	}
	if res.Layer != "" {
		m.Image.Stacks = addUnique(m.Image.Stacks, res.Layer)
	}
	if err := m.Save(sp.Manifest()); err != nil {
		return err
	}
	if res.Layer != "" {
		fmt.Printf("added %s (stack %s/%s -> image layer %q)\n", name, res.Stack.Lang, res.Stack.Build, res.Layer)
	} else {
		fmt.Printf("added %s (stack not detected%s)\n", name, dryNote())
	}
	fmt.Println("manifest updated; run 'dock build' to (re)build the image")
	return nil
}

func cmdBuild(args []string) error {
	sp, m, err := loadActive()
	if err != nil {
		return err
	}
	if err := writeGenerated(sp, m); err != nil {
		return err
	}
	fmt.Println("generated:", sp.Dockerfile(), "+", sp.LiteLLM(), "+", sp.Compose())
	return run("docker", "compose", "-f", sp.Compose(), "build")
}

func cmdStart(args []string) error {
	sp, m, err := loadActive()
	if err != nil {
		return err
	}
	if err := writeGenerated(sp, m); err != nil {
		return err
	}
	session := time.Now().UTC().Format("20060102T150405Z")
	sdir := paths.SessionDir(sp.Name, "_session", session)
	if _, err := telemetry.StartSession(sdir, sp.Name, "_session", session); err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not start telemetry session:", err)
	}
	if err := run("docker", "compose", "-f", sp.Compose(), "up", "-d"); err != nil {
		return err
	}
	fmt.Println("gateway:  http://localhost:4000/v1")
	for _, p := range m.Ports {
		fmt.Printf("preview:  http://localhost:%d\n", p)
	}
	fmt.Println("session:  ", sdir)
	fmt.Println("attach:   dock shell")
	return nil
}

func cmdShell(args []string) error {
	sp, err := activeSpace()
	if err != nil {
		return err
	}
	return run("docker", "compose", "-f", sp.Compose(), "exec", "dev", "bash")
}

func cmdStop(args []string) error {
	sp, err := activeSpace()
	if err != nil {
		return err
	}
	return run("docker", "compose", "-f", sp.Compose(), "down")
}

func cmdSync(args []string) error {
	sp, m, err := loadActive()
	if err != nil {
		return err
	}
	if len(m.Repos) == 0 {
		fmt.Println("no repos in this space; add one with 'dock addrepo <url>'")
		return nil
	}
	failed := 0
	for _, r := range m.Repos {
		n := repoName(r.URL)
		fmt.Println("syncing", n)
		if err := run("git", "-C", filepath.Join(sp.Repos, n), "pull", "--ff-only"); err != nil {
			fmt.Fprintf(os.Stderr, "  pull failed for %s: %v\n", n, err)
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d repo(s) failed to sync", failed)
	}
	return nil
}

func cmdSpace(args []string) error {
	if len(args) < 2 || args[0] != "switch" {
		return fmt.Errorf("usage: dock space switch <space>")
	}
	target := args[1]
	tp := paths.For(target)
	if _, err := os.Stat(tp.Manifest()); err != nil {
		return fmt.Errorf("space %q not found (no %s); run 'dock setup %s' first", target, tp.Manifest(), target)
	}
	if cur, err := activeSpace(); err == nil && cur.Name != target {
		if _, err := os.Stat(cur.Compose()); err == nil {
			fmt.Println("stopping current space:", cur.Name)
			_ = run("docker", "compose", "-f", cur.Compose(), "down")
		}
	}
	if err := setActive(target); err != nil {
		return err
	}
	fmt.Printf("active space is now %q (%s)\n", target, tp.Root)
	return nil
}

func cmdUpdate(args []string) error {
	sp, m, err := loadActive()
	if err != nil {
		return err
	}
	for _, d := range []string{sp.Repos, sp.Vault, sp.Works, sp.Drydock} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	if err := writeGenerated(sp, m); err != nil {
		return err
	}
	fmt.Println("refreshed space config for", sp.Name, "(regenerated gateway + compose + Dockerfile)")
	fmt.Println("note: 'dock update' refreshes space config; use 'dock self-update' for the binary")
	return nil
}

func cmdSelfUpdate(args []string) error {
	var target string
	force := false
	for _, a := range args {
		switch {
		case a == "--force":
			force = true
		case strings.HasPrefix(a, "--version="):
			target = strings.TrimPrefix(a, "--version=")
		default:
			return fmt.Errorf("usage: dock self-update [--force] [--version=<tag>]")
		}
	}
	if dryRun {
		fmt.Printf("[dry-run] self-update from github.com/%s/%s (current %s)\n", ghOwner, ghRepo, version.Version)
		return nil
	}
	fmt.Println("checking for updates ...")
	v, err := selfupdate.Run(selfupdate.Options{
		Owner: ghOwner, Repo: ghRepo, Current: version.Version, Version: target, Force: force,
	})
	if err != nil {
		return err
	}
	if v == "" {
		fmt.Printf("already up to date (%s)\n", version.Version)
		return nil
	}
	fmt.Printf("updated %s -> %s\n", version.Version, v)
	return nil
}

func cmdWork(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: dock work <new|status|set> ...")
	}
	sp, err := activeSpace()
	if err != nil {
		return err
	}
	switch args[0] {
	case "new":
		if len(args) < 2 {
			return fmt.Errorf("usage: dock work new <ticket>")
		}
		ticket := args[1]
		dir := filepath.Join(sp.Works, ticket)
		if _, err := works.Scaffold(dir, sp.Name, ticket); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(sp.Drydock, "active_ticket"), []byte(ticket+"\n"), 0o644); err != nil {
			return err
		}
		fmt.Printf("ticket %q ready at %s (active)\n", ticket, dir)
		fmt.Println("edit ticket.md, then run the analyze skill inside 'dock shell'")
		return nil

	case "status":
		ticket, err := resolveTicket(sp, args[1:])
		if err != nil {
			return err
		}
		st, err := works.Load(filepath.Join(sp.Works, ticket, "state.yaml"))
		if err != nil {
			return err
		}
		printStatus(st)
		return nil

	case "set":
		if len(args) < 3 {
			return fmt.Errorf("usage: dock work set <phase> <status> [--model m --note n --artifact f]")
		}
		phase, status := works.Phase(args[1]), args[2]
		var model, note string
		var arts []string
		fl := args[3:]
		for i := 0; i < len(fl); i++ {
			switch fl[i] {
			case "--model":
				if i++; i < len(fl) {
					model = fl[i]
				}
			case "--note":
				if i++; i < len(fl) {
					note = fl[i]
				}
			case "--artifact":
				if i++; i < len(fl) {
					arts = append(arts, fl[i])
				}
			}
		}
		ticket, err := resolveTicket(sp, nil)
		if err != nil {
			return err
		}
		path := filepath.Join(sp.Works, ticket, "state.yaml")
		st, err := works.Load(path)
		if err != nil {
			return err
		}
		if err := st.Mark(phase, status, model, note, arts...); err != nil {
			return err
		}
		if err := st.Save(path); err != nil {
			return err
		}
		fmt.Printf("%s: %s -> %s\n", ticket, phase, status)
		return nil

	default:
		return fmt.Errorf("usage: dock work <new|status|set> ...")
	}
}

func resolveTicket(sp paths.Space, args []string) (string, error) {
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}
	b, err := os.ReadFile(filepath.Join(sp.Drydock, "active_ticket"))
	if err != nil {
		return "", fmt.Errorf("no active ticket; run 'dock work new <ticket>' or pass a ticket name")
	}
	return strings.TrimSpace(string(b)), nil
}

func printStatus(st *works.State) {
	fmt.Printf("ticket %s (space %s) — current: %s  backend: %s\n", st.Ticket, st.Space, st.Current, st.TaskBackend)
	for _, p := range works.Order {
		ps := st.Phases[p]
		gate := ""
		if ps.Gate == works.GateHuman {
			gate = "  [human gate]"
		}
		extra := ps.Model
		if len(ps.Artifacts) > 0 {
			extra = strings.TrimSpace(extra + " " + strings.Join(ps.Artifacts, ","))
		}
		fmt.Printf("  %-9s %-12s%s %s\n", p, ps.Status, gate, extra)
	}
}

func cmdArchive(args []string) error {
	clean := false
	var rest []string
	for _, a := range args {
		if a == "--clean" {
			clean = true
		} else {
			rest = append(rest, a)
		}
	}
	sp, err := activeSpace()
	if err != nil {
		return err
	}
	ticket, err := resolveTicket(sp, rest)
	if err != nil {
		return err
	}
	src := filepath.Join(sp.Works, ticket)
	statePath := filepath.Join(src, "state.yaml")
	st, err := works.Load(statePath)
	if err != nil {
		return err
	}
	if st.Phases[works.Ship].Status != works.StatusDone {
		fmt.Fprintf(os.Stderr, "warning: ship is %q (not done) for %s — archiving anyway\n", st.Phases[works.Ship].Status, ticket)
	}
	if dryRun {
		fmt.Printf("[dry-run] copy %s -> %s/inbox/%s, then run vault:ingest\n", src, sp.Vault, ticket)
		return nil
	}
	inbox, err := vault.Archive(src, sp.Vault, ticket)
	if err != nil {
		return err
	}
	_ = st.Mark(works.Archive, works.StatusDone, "", "", "vault:inbox/"+ticket)
	if err := st.Save(statePath); err != nil {
		return err
	}
	fmt.Println("staged artifacts ->", inbox)
	if hook, ok := vault.IngestHook(sp.Vault); ok {
		fmt.Println("running vault ingest hook:", hook)
		if err := run(hook, inbox); err != nil {
			return fmt.Errorf("vault ingest hook failed: %w", err)
		}
	} else {
		fmt.Println("no <vault>/bin/ingest hook — run the vault:ingest skill on the inbox (the vault project owns ingestion; design doc 8)")
	}
	if clean {
		if err := os.RemoveAll(src); err != nil {
			return err
		}
		_ = os.Remove(filepath.Join(sp.Drydock, "active_ticket"))
		fmt.Println("cleaned", src)
	}
	fmt.Printf("archived %s\n", ticket)
	return nil
}

func cmdTelemetry(args []string) error {
	jsonOut := false
	var ticket string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOut = true
		case "--ticket":
			if i++; i < len(args) {
				ticket = args[i]
			}
		}
	}
	sp, err := activeSpace()
	if err != nil {
		return err
	}
	root := filepath.Join(paths.StateHome(), "sessions", sp.Name)
	if ticket != "" {
		root = filepath.Join(root, ticket)
	}
	files := telemetry.FindEventFiles(root)
	var all []telemetry.Event
	for _, f := range files {
		evs, _ := telemetry.ReadEvents(f)
		all = append(all, evs...)
	}
	sum := telemetry.Summarize(all)
	sum.Sessions = len(files)
	if jsonOut {
		b, _ := json.MarshalIndent(sum, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	fmt.Printf("telemetry: space=%s ticket=%s | sessions=%d events=%d failures=%d tokens=%d/%d cost=$%.4f\n",
		sp.Name, orAll(ticket), sum.Sessions, sum.Events, sum.Failures, sum.TokensIn, sum.TokensOut, sum.CostUSD)
	phases := make([]string, 0, len(sum.ByPhase))
	for k := range sum.ByPhase {
		phases = append(phases, k)
	}
	sort.Strings(phases)
	for _, ph := range phases {
		st := sum.ByPhase[ph]
		fmt.Printf("  %-10s events=%-3d fail=%-2d tools=%-3d llm=%-3d cost=$%.4f\n",
			ph, st.Events, st.Failures, st.ToolCalls, st.LLMCalls, st.CostUSD)
	}
	if sum.Sessions == 0 {
		fmt.Println("  (no telemetry yet — run some lifecycle work first)")
	}
	return nil
}

func orAll(s string) string {
	if s == "" {
		return "*"
	}
	return s
}

func cmdSkill(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: dock skill <propose|list|show|apply|reject> ...")
	}
	pdir := filepath.Join(paths.StateHome(), "proposals")
	switch args[0] {
	case "propose":
		if len(args) < 2 {
			return fmt.Errorf("usage: dock skill propose <skill> --content <file> [--source s --rationale r]")
		}
		skill := args[1]
		source, rationale, contentPath := "retrospect", "", ""
		fl := args[2:]
		for i := 0; i < len(fl); i++ {
			switch fl[i] {
			case "--source":
				if i++; i < len(fl) {
					source = fl[i]
				}
			case "--rationale":
				if i++; i < len(fl) {
					rationale = fl[i]
				}
			case "--content":
				if i++; i < len(fl) {
					contentPath = fl[i]
				}
			}
		}
		if contentPath == "" {
			return fmt.Errorf("dock skill propose needs --content <file> (the revised SKILL.md, version bumped)")
		}
		content, err := os.ReadFile(contentPath)
		if err != nil {
			return err
		}
		p, err := proposal.Create(pdir, skill, source, rationale, string(content))
		if err != nil {
			return err
		}
		fmt.Printf("proposal %s created (pending) for skill %q — review with 'dock skill show %s'\n", p.ID, skill, p.ID)
		return nil

	case "list":
		ps, err := proposal.List(pdir)
		if err != nil {
			return err
		}
		if len(ps) == 0 {
			fmt.Println("no proposals")
			return nil
		}
		for _, p := range ps {
			fmt.Printf("  %-28s %-9s %-10s %s\n", p.ID, p.Status, p.Source, p.Skill)
		}
		return nil

	case "show":
		if len(args) < 2 {
			return fmt.Errorf("usage: dock skill show <id>")
		}
		p, content, err := proposal.Load(pdir, args[1])
		if err != nil {
			return err
		}
		fmt.Printf("id: %s\nskill: %s\nsource: %s\nstatus: %s\nrationale: %s\n--- proposed SKILL.md ---\n%s",
			p.ID, p.Skill, p.Source, p.Status, p.Rationale, content)
		return nil

	case "apply":
		if len(args) < 2 {
			return fmt.Errorf("usage: dock skill apply <id> [--to <skills-dir>]")
		}
		id := args[1]
		to := "skills"
		fl := args[2:]
		for i := 0; i < len(fl); i++ {
			if fl[i] == "--to" {
				if i++; i < len(fl) {
					to = fl[i]
				}
			}
		}
		p, _, err := proposal.Load(pdir, id)
		if err != nil {
			return err
		}
		target := filepath.Join(to, p.Skill, "SKILL.md")
		if err := proposal.Apply(pdir, id, target); err != nil {
			return err
		}
		fmt.Printf("applied %s -> %s\n", id, target)
		fmt.Println("review the change, then commit it (the SKILL.md should carry a bumped metadata.version)")
		return nil

	case "reject":
		if len(args) < 2 {
			return fmt.Errorf("usage: dock skill reject <id>")
		}
		if err := proposal.SetStatus(pdir, args[1], proposal.StatusRejected); err != nil {
			return err
		}
		fmt.Printf("rejected %s\n", args[1])
		return nil

	default:
		return fmt.Errorf("usage: dock skill <propose|list|show|apply|reject> ...")
	}
}

func cmdForward(args []string) error {
	if len(args) < 1 || !strings.Contains(args[0], ":") {
		return fmt.Errorf("usage: dock forward <hostPort>:<containerPort>")
	}
	fmt.Printf("ad-hoc forward (%s) lands in a later milestone. For now declare the port in space.yaml 'ports:' and run 'dock start'.\n", args[0])
	return nil
}

// --- generation ------------------------------------------------------------

func writeGenerated(sp paths.Space, m *config.Manifest) error {
	if err := os.MkdirAll(sp.Drydock, 0o755); err != nil {
		return err
	}
	lc, err := gen.LiteLLMConfig(m)
	if err != nil {
		return err
	}
	if err := os.WriteFile(sp.LiteLLM(), []byte(lc), 0o644); err != nil {
		return err
	}
	cf, err := gen.ComposeFile(m, sp)
	if err != nil {
		return err
	}
	if err := os.WriteFile(sp.Compose(), []byte(cf), 0o644); err != nil {
		return err
	}
	return os.WriteFile(sp.Dockerfile(), []byte(gen.Dockerfile(m)), 0o644)
}

// --- helpers ---------------------------------------------------------------

func repoName(url string) string {
	u := strings.TrimSuffix(strings.TrimRight(url, "/"), ".git")
	if i := strings.LastIndexAny(u, "/:"); i >= 0 {
		u = u[i+1:]
	}
	return u
}

func repoKnown(m *config.Manifest, url string) bool {
	for _, r := range m.Repos {
		if r.URL == url {
			return true
		}
	}
	return false
}

func addUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}

func dryNote() string {
	if dryRun {
		return " — clone skipped under --dry-run"
	}
	return ""
}

// run executes a command with inherited stdio, or prints it under --dry-run.
func run(name string, args ...string) error {
	if dryRun {
		fmt.Printf("[dry-run] %s %s\n", name, strings.Join(args, " "))
		return nil
	}
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s not found on PATH", name)
	}
	c := exec.Command(name, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}

func doctor() int {
	type check struct{ name, bin, hint string }
	checks := []check{
		{"docker", "docker", "install Docker Engine"},
		{"forge (ForgeCode)", "forge", "curl -fsSL https://forgecode.dev/cli | sh"},
		{"git", "git", "install git"},
	}
	missing := false
	for _, c := range checks {
		if _, err := exec.LookPath(c.bin); err != nil {
			fmt.Printf("  [MISSING] %-18s -> %s\n", c.name, c.hint)
			missing = true
		} else {
			fmt.Printf("  [ok]      %-18s\n", c.name)
		}
	}
	if sp, err := activeSpace(); err == nil {
		fmt.Printf("  [ok]      active space: %s\n", sp.Name)
	} else {
		fmt.Println("  [info]    no active space (run 'dock setup <space>')")
	}
	if missing {
		fmt.Println("doctor: some components are missing (see hints)")
		return 1
	}
	fmt.Println("doctor: environment looks good")
	return 0
}
