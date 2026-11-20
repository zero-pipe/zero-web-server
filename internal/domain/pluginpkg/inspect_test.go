package pluginpkg

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

const goodManifest = `spec: 1
id: demo-plugin
name: 演示插件
vendor: Demo
version: 1.0.0
protocols: [demo]
rpc: v1
executable:
  linux-amd64: bin/linux-amd64/demo
capabilities: [login, live]
stream_transports: [rtsp_url]
`

func buildZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestInspectValid(t *testing.T) {
	data := buildZip(t, map[string]string{
		"plugin.yaml":            goodManifest,
		"bin/linux-amd64/demo":   "binary",
		"config.schema.json":     `{"type":"object"}`,
	})
	res, err := FromBytes(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Manifest.ID != "demo-plugin" || res.Manifest.Version != "1.0.0" {
		t.Fatalf("manifest mismatch: %+v", res.Manifest)
	}
	if len(res.SHA256) != 64 {
		t.Fatalf("sha256 malformed: %s", res.SHA256)
	}
	if len(res.Architectures) != 1 || res.Architectures[0] != "linux-amd64" {
		t.Fatalf("architectures: %v", res.Architectures)
	}
}

func TestInspectRejects(t *testing.T) {
	cases := map[string]map[string]string{
		"missing manifest": {"bin/x": "y"},
		"bad id":           {"plugin.yaml": strings.Replace(goodManifest, "demo-plugin", "Demo_Plugin", 1), "bin/linux-amd64/demo": "x"},
		"bad version":      {"plugin.yaml": strings.Replace(goodManifest, "1.0.0", "v1", 1), "bin/linux-amd64/demo": "x"},
		"missing binary":   {"plugin.yaml": goodManifest},
		"unknown cap":      {"plugin.yaml": strings.Replace(goodManifest, "login, live", "login, teleport", 1), "bin/linux-amd64/demo": "x"},
		"bad transport":    {"plugin.yaml": strings.Replace(goodManifest, "rtsp_url", "carrier_pigeon", 1), "bin/linux-amd64/demo": "x"},
		"bad rpc":          {"plugin.yaml": strings.Replace(goodManifest, "rpc: v1", "rpc: v9", 1), "bin/linux-amd64/demo": "x"},
		"bad schema": {"plugin.yaml": strings.Replace(goodManifest, "stream_transports: [rtsp_url]",
			"stream_transports: [rtsp_url]\nconfig_schema: bad.json", 1), "bin/linux-amd64/demo": "x", "bad.json": "not-json"},
	}
	for name, files := range cases {
		if _, err := FromBytes(buildZip(t, files)); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

func TestCheckEntryName(t *testing.T) {
	for _, bad := range []string{"../evil", "/abs", "C:/evil", `a\b`} {
		if err := checkEntryName(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
	for _, ok := range []string{"plugin.yaml", "bin/linux-amd64/demo", "docs/README.md"} {
		if err := checkEntryName(ok); err != nil {
			t.Errorf("%q should be accepted: %v", ok, err)
		}
	}
}
