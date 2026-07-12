package ops

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LogFileInfo 历史日志文件条目。
type LogFileInfo struct {
	FileName  string `json:"fileName"`
	FileSize  int64  `json:"fileSize"`
	StartTime int64  `json:"startTime"` // ms
	EndTime   int64  `json:"endTime"`   // ms
}

type LogService struct {
	dir string
}

func NewLogService(logDir string) *LogService {
	if logDir == "" {
		logDir = "logs"
	}
	return &LogService{dir: logDir}
}

func (s *LogService) Dir() string { return s.dir }

func (s *LogService) List(query, startTime, endTime string) ([]LogFileInfo, error) {
	dir := s.dir
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogFileInfo{}, nil
		}
		return nil, fmt.Errorf("读取日志目录失败: %w", err)
	}

	var startMS, endMS int64
	if startTime != "" {
		startMS = parseTimeMS(startTime)
	}
	if endTime != "" {
		endMS = parseTimeMS(endTime)
	}

	out := make([]LogFileInfo, 0, len(entries))
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		if query != "" && !strings.Contains(name, query) {
			continue
		}
		info, err := ent.Info()
		if err != nil {
			continue
		}
		full := filepath.Join(dir, name)
		start, end := fileTimeRange(full, info.ModTime())
		if startMS > 0 && start < startMS {
			continue
		}
		if endMS > 0 && end > endMS {
			continue
		}
		out = append(out, LogFileInfo{
			FileName:  name,
			FileSize:  info.Size(),
			StartTime: start,
			EndTime:   end,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartTime > out[j].StartTime
	})
	return out, nil
}

// ResolveFile returns absolute path under log dir; rejects path traversal.
func (s *LogService) ResolveFile(fileName string) (string, error) {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	if fileName == "" || fileName == "." || fileName == ".." {
		return "", fmt.Errorf("文件名无效")
	}
	full := filepath.Join(s.dir, fileName)
	absDir, err := filepath.Abs(s.dir)
	if err != nil {
		return "", err
	}
	absFile, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absDir, absFile)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("非法路径")
	}
	st, err := os.Stat(absFile)
	if err != nil || st.IsDir() {
		return "", fmt.Errorf("文件不存在")
	}
	return absFile, nil
}

func fileTimeRange(path string, modTime time.Time) (startMS, endMS int64) {
	endMS = modTime.UnixMilli()
	startMS = endMS
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	if !sc.Scan() {
		return
	}
	line := sc.Text()
	if t := parseLogLineTime(line); !t.IsZero() {
		startMS = t.UnixMilli()
	}
	return
}

func parseLogLineTime(line string) time.Time {
	line = strings.TrimSpace(line)
	if len(line) >= 19 {
		// "2006-01-02 15:04:05..."
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", line[:19], time.Local); err == nil {
			return t
		}
	}
	// slog text: time=2006-01-02T15:04:05.000+08:00
	if idx := strings.Index(line, "time="); idx >= 0 {
		rest := line[idx+5:]
		end := strings.IndexByte(rest, ' ')
		if end < 0 {
			end = len(rest)
		}
		raw := rest[:end]
		layouts := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05.000-07:00",
			"2006-01-02T15:04:05-07:00",
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, raw); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

func parseTimeMS(raw string) int64 {
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t.UnixMilli()
		}
	}
	return 0
}
