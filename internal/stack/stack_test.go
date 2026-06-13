package stack

import "testing"

func TestDetectFromNames(t *testing.T) {
	cases := []struct {
		name  string
		files []string
		lang  string
		build string
		layer string
	}{
		{"maven", []string{"pom.xml", "src"}, "java", "maven", "jdk21-maven"},
		{"gradle", []string{"build.gradle.kts"}, "java", "gradle", "jdk21-gradle"},
		{"go", []string{"go.mod", "main.go"}, "go", "go", "go"},
		{"npm", []string{"package.json"}, "node", "npm", "node"},
		{"pnpm", []string{"package.json", "pnpm-lock.yaml"}, "node", "pnpm", "node"},
		{"yarn", []string{"package.json", "yarn.lock"}, "node", "yarn", "node"},
		{"python", []string{"pyproject.toml"}, "python", "pip", "python"},
		{"rust", []string{"Cargo.toml"}, "rust", "cargo", "rust"},
		{"unknown", []string{"README.md"}, "", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := DetectFromNames(c.files)
			if string(r.Stack.Lang) != c.lang || r.Stack.Build != c.build || r.Layer != c.layer {
				t.Fatalf("got lang=%q build=%q layer=%q, want %q/%q/%q",
					r.Stack.Lang, r.Stack.Build, r.Layer, c.lang, c.build, c.layer)
			}
		})
	}
}

func TestMavenPinsJDK21(t *testing.T) {
	r := DetectFromNames([]string{"pom.xml"})
	if string(r.Stack.Version) != "21" {
		t.Fatalf("java version=%q want 21", r.Stack.Version)
	}
}
