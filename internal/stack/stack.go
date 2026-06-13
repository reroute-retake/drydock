// Package stack detects a repository's tech stack from its files, so a space's
// container image can install the right per-stack toolchain layers (design doc
// 7.6 / M2). Detection is a pure function over a file-name set, plus a thin
// directory-reading wrapper.
package stack

import (
	"os"
	"strings"

	"github.com/reroute-retake/drydock/internal/config"
)

// Result is a detected stack and the image-layer id it maps to.
type Result struct {
	Stack config.Stack
	Layer string // "" when unknown
}

// DetectFromNames classifies a repo from the base names of its top-level files.
func DetectFromNames(names []string) Result {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[strings.ToLower(n)] = true
	}
	switch {
	case set["pom.xml"]:
		return Result{config.Stack{Lang: "java", Version: "21", Build: "maven"}, "jdk21-maven"}
	case set["build.gradle"] || set["build.gradle.kts"]:
		return Result{config.Stack{Lang: "java", Version: "21", Build: "gradle"}, "jdk21-gradle"}
	case set["go.mod"]:
		return Result{config.Stack{Lang: "go", Build: "go"}, "go"}
	case set["package.json"]:
		build := "npm"
		switch {
		case set["pnpm-lock.yaml"]:
			build = "pnpm"
		case set["yarn.lock"]:
			build = "yarn"
		}
		return Result{config.Stack{Lang: "node", Build: build}, "node"}
	case set["pyproject.toml"] || set["requirements.txt"] || set["setup.py"]:
		return Result{config.Stack{Lang: "python", Build: "pip"}, "python"}
	case set["cargo.toml"]:
		return Result{config.Stack{Lang: "rust", Build: "cargo"}, "rust"}
	default:
		return Result{}
	}
}

// Detect reads the top level of dir and classifies its stack.
func Detect(dir string) Result {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return Result{}
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		names = append(names, e.Name())
	}
	return DetectFromNames(names)
}
