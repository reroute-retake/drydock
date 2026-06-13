// Command dock is the drydock host-side orchestrator.
//
// M1: setup/build/start/shell/stop are functional (build/start/shell/stop shell
// out to `docker compose`; pass --dry-run to print the commands instead of
// running them). version and doctor are functional. addrepo/sync/space/update/
// self-update remain stubs for later milestones.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/reroute-retake/drydock/internal/config"
	"github.com/reroute-retake/drydock/internal/gen"
	"github.com/reroute-retake/drydock/internal/paths"
	"github.com/reroute-retake/drydock/internal/telemetry"
	"github.com/reroute-retake/drydock/internal/version"
)

var dryRun bool

const usage = `dock — drydock host-side orchestrator

Usage: dock [--dry-run] <command> [args]

Environment:
  setup <space>      Scaffold a space (repos/vault/works, manifest, .env) and make it active
  addrepo <url>      Clone a repo into the active space and detect its stack   [stub]
  build              Generate gateway+compose from the manifest and validate
  start              Start the space container + LiteLLM gateway; publish ports
  shell              Attach a shell to the running space container
  forward <h:c>      Ad-hoc port forward host:container                         [M2]
  stop               Stop the space containers
  sync               git pull all repos in the active space                     [stub]
  space switch <s>   Switch the active space                                    [stub]

Maintenance:
  update             Refresh the active space's config/scaffolding (NOT the binary) [stub]
  self-update        Replace the dock binary from the latest release               [stub]
  doctor             Diagnose the local environment
  version            Print version

Flags:
  --dry-run          Print docker/compose commands instead of executing them
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
	case "build":
		err = cmdBuild(args)
	case "start":
		err = cmdStart(args)
	case "shell":
		err = cmdShell(args)
	case "stop":
		err = cmdStop(args)
	case "forward":
		err = cmdForward(args)
	case "addrepo", "sync", "space", "update", "self-update", "run":
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
	return os.WriteFile(paths.StateHome()+"/active", []byte(space+"\n"), 0o644)
}

func activeSpace() (paths.Space, error) {
	b, err := os.ReadFile(paths.StateHome() + "/active")
	if err != nil {
		return paths.Space{}, fmt.Errorf("no active space; run 'dock setup <space>' first")
	}
	name := strings.TrimSpace(string(b))
	if name == "" {
		return paths.Space{}, fmt.Errorf("active space is empty; run 'dock setup <space>'")
	}
	return paths.For(name), nil
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
	fmt.Println("next: edit space.yaml + .env, then 'dock build' and 'dock start'")
	return nil
}

const defaultEnv = `# drydock space secrets — NEVER commit (P12/C8).
ANTHROPIC_API_KEY=
OPENAI_API_KEY=
LITELLM_MASTER_KEY=sk-drydock-local
`

// writeGenerated renders the gateway config and compose file into .drydock/.
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
	return os.WriteFile(sp.Compose(), []byte(cf), 0o644)
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

func cmdBuild(args []string) error {
	sp, m, err := loadActive()
	if err != nil {
		return err
	}
	if err := writeGenerated(sp, m); err != nil {
		return err
	}
	fmt.Println("generated", sp.LiteLLM())
	fmt.Println("generated", sp.Compose())
	// Validate the compose file. (Per-stack image building lands in M2.)
	return run("docker", "compose", "-f", sp.Compose(), "config", "-q")
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
	// Named volumes (caches/indexes) persist; use 'down -v' manually to wipe.
	return run("docker", "compose", "-f", sp.Compose(), "down")
}

func cmdForward(args []string) error {
	if len(args) < 1 || !strings.Contains(args[0], ":") {
		return fmt.Errorf("usage: dock forward <hostPort>:<containerPort>")
	}
	fmt.Printf("ad-hoc forward (%s) lands in M2. For now declare the port in space.yaml 'ports:' and run 'dock start'.\n", args[0])
	return nil
}

// --- helpers ---------------------------------------------------------------

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
