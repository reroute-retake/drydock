// Command dock is the drydock host-side orchestrator.
//
// M0 status: this is the CLI skeleton. `version` and `doctor` are functional;
// the environment/maintenance subcommands are stubs that state their intent so
// the command surface and wiring can be exercised before M1 implements them.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/reroute-retake/drydock/internal/version"
)

const usage = `dock — drydock host-side orchestrator

Usage: dock <command> [args]

Environment:
  setup <space>      Scaffold a space (repos/, vault clone, manifest, .env)
  addrepo <url>      Clone a repo into the active space and detect its stack
  build              Build the space container image (base + per-stack layers)
  start              Start the space container + LiteLLM gateway; publish ports
  shell              Attach a shell to the running space container
  forward <h:c>      Ad-hoc port forward host:container for a mid-session service
  stop               Stop the space containers
  sync               git pull all repos in the active space
  space switch <s>   Switch the active space

Maintenance:
  update             Refresh the active space's config/scaffolding (NOT the binary)
  self-update        Replace the dock binary from the latest release
  doctor             Diagnose the local environment
  version            Print version

M0: only 'version' and 'doctor' are functional; other commands print intent.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "version", "-v", "--version":
		fmt.Println(version.String())
	case "doctor":
		os.Exit(doctor())
	case "help", "-h", "--help":
		fmt.Print(usage)
	case "setup", "addrepo", "build", "start", "shell", "forward",
		"stop", "sync", "space", "update", "self-update", "run":
		stub(os.Args[1], os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "dock: unknown command %q\n\n", os.Args[1])
		fmt.Print(usage)
		os.Exit(2)
	}
}

func stub(cmd string, args []string) {
	fmt.Printf("[M0 stub] dock %s %s\n", cmd, strings.Join(args, " "))
	fmt.Println("  Not implemented in M0. See the design doc command spec (section 9).")
}

// doctor performs best-effort environment checks. Missing optional tools are
// reported (not fatal) so the user can see and fix gaps.
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
	if v := os.Getenv("OPENAI_URL"); v != "" {
		fmt.Printf("  [ok]      gateway OPENAI_URL=%s\n", v)
	} else {
		fmt.Println("  [info]    OPENAI_URL unset (point it at the LiteLLM gateway, e.g. http://localhost:4000/v1)")
	}
	if missing {
		fmt.Println("doctor: some components are missing (see hints above)")
		return 1
	}
	fmt.Println("doctor: environment looks good")
	return 0
}
