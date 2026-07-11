package civilcode

import (
	"bufio"
	_ "embed"
	"sort"
	"strings"
	"sync"
)

//go:embed civilCode.csv
var csvData string

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
	once sync.Once
	byCode map[string]Entry
)

func load() {
	byCode = make(map[string]Entry)
	scanner := bufio.NewScanner(strings.NewReader(csvData))
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
}

func ensureLoaded() {
	once.Do(load)
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
	// 按行政区划编码升序，保证省/市/区列表稳定可读
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
