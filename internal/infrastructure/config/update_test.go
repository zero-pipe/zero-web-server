package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateSectionPreservesOtherSectionsAndComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	orig := `# top comment
server:
  port: 18080 # http port

mysql:
  host: 127.0.0.1
  port: 3306
  user: root
  password: "old"
  database: zws
  charset: utf8mb4
  max_idle_conns: 10
  max_open_conns: 100

redis:
  host: 127.0.0.1
  port: 6379
`
	if err := os.WriteFile(path, []byte(orig), 0o600); err != nil {
		t.Fatal(err)
	}
	err := UpdateSection(path, "mysql", map[string]any{
		"host": "10.0.0.2", "port": 3307, "password": "new",
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	text := string(raw)
	if !contains(text, "10.0.0.2") || !contains(text, "3307") || !contains(text, "new") {
		t.Fatalf("updated values missing:\n%s", text)
	}
	if !contains(text, "# top comment") || !contains(text, "# http port") {
		t.Fatalf("comments were not preserved:\n%s", text)
	}
	if !contains(text, "redis:") {
		t.Fatalf("other section missing:\n%s", text)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
}

func TestUpdateSectionCreatesSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(path, []byte("server:\n  port: 1\n"), 0o600)
	if err := UpdateSection(path, "mqtt", map[string]any{"broker": "tcp://x:1883"}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if !contains(string(raw), "mqtt:") || !contains(string(raw), "tcp://x:1883") {
		t.Fatalf("section not created:\n%s", raw)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
