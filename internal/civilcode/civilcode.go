package civilcode

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// DefaultCSVPath is used when Configure is not called.
const DefaultCSVPath = "configs/civilCode.csv"

type Entry struct {
	Code       string
	Name       string
	ParentCode string
}

type RegionItem struct {
	DeviceID       string `json:"deviceId"`
	Name           string `json:"name"`
	ParentDeviceID string `json:"parentDeviceId,omitempty"`
}

var (
	once    sync.Once
	byCode  map[string]Entry
	csvPath = DefaultCSVPath
	loadErr error
)

// Configure sets the civil-code CSV path (e.g. alongside config.yaml).
// Call before first GetAllChild / GetDescription; empty keeps DefaultCSVPath.
func Configure(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	csvPath = path
}

func resolvePath(path string) string {
	if path == "" {
		path = DefaultCSVPath
	}
	if filepath.IsAbs(path) {
		return path
	}
	candidates := []string{path}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, path),
			filepath.Join(dir, "..", path),
		)
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return path
}

func load() {
	byCode = make(map[string]Entry)
	path := resolvePath(csvPath)
	f, err := os.Open(path)
	if err != nil {
		loadErr = err
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// CSV lines can be long; default 64K is enough for this file but stay safe.
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum == 1 {
			continue
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		code := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		parent := ""
		if len(parts) >= 3 {
			parent = strings.TrimSpace(parts[2])
		}
		byCode[code] = Entry{Code: code, Name: name, ParentCode: parent}
	}
	if err := scanner.Err(); err != nil {
		loadErr = err
	}
}

func ensureLoaded() {
	once.Do(load)
}

// LoadError returns the last CSV open/parse error after ensureLoaded (for diagnostics).
func LoadError() error {
	ensureLoaded()
	return loadErr
}

func GetAllChild(parent string) []RegionItem {
	ensureLoaded()
	parent = strings.TrimSpace(parent)
	out := make([]RegionItem, 0)
	for _, e := range byCode {
		if parent == "" {
			if e.ParentCode == "" {
				out = append(out, RegionItem{DeviceID: e.Code, Name: e.Name})
			}
			continue
		}
		if e.ParentCode == parent {
			out = append(out, RegionItem{
				DeviceID:       e.Code,
				Name:           e.Name,
				ParentDeviceID: parent,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].DeviceID < out[j].DeviceID
	})
	return out
}

func GetDescription(code string) string {
	ensureLoaded()
	if e, ok := byCode[strings.TrimSpace(code)]; ok {
		return e.Name
	}
	return ""
}
