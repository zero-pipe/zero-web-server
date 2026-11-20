package agent

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

const manifest = "id: demo\nversion: 1.0.0\n"

func TestExtractZip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "pkg.zip")
	writeZip(t, zipPath, map[string]string{"plugin.yaml": manifest, "bin/x": "y"})
	dst := filepath.Join(dir, "out")
	if err := extractZip(zipPath, dst); err != nil {
		t.Fatalf("extract: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "plugin.yaml")); err != nil {
		t.Fatal("manifest missing after extract")
	}
}

func TestExtractZipRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "evil.zip")
	writeZip(t, zipPath, map[string]string{"../evil.txt": "x", "plugin.yaml": manifest})
	if err := extractZip(zipPath, filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestVerifyManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyManifest(dir, "demo", "1.0.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := verifyManifest(dir, "other", "1.0.0"); err == nil {
		t.Fatal("id mismatch should fail")
	}
	if err := verifyManifest(dir, "demo", "2.0.0"); err == nil {
		t.Fatal("version mismatch should fail")
	}
}

func TestCurrentPointer(t *testing.T) {
	dir := t.TempDir()
	if _, err := readCurrent(dir); err == nil {
		t.Fatal("expected missing current")
	}
	if err := writeCurrent(dir, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	cur, err := readCurrent(dir)
	if err != nil || cur != "1.2.3" {
		t.Fatalf("current=%q err=%v", cur, err)
	}
}

func TestPruneVersions(t *testing.T) {
	root := t.TempDir()
	for _, v := range []string{"1.0.0", "1.0.1", "1.0.2"} {
		if err := os.MkdirAll(filepath.Join(root, v), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	pruneVersions(root, "1.0.2", 2)
	entries, _ := os.ReadDir(root)
	var left []string
	for _, e := range entries {
		left = append(left, e.Name())
	}
	joined := strings.Join(left, ",")
	if !strings.Contains(joined, "1.0.2") || strings.Contains(joined, "1.0.0") {
		t.Fatalf("prune result: %v", left)
	}
}

func TestValidID(t *testing.T) {
	for _, ok := range []string{"demo-flow", "a1", "x.y_z"} {
		if !validID(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "Demo", "../x", "a b", strings.Repeat("a", 200)} {
		if validID(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}
