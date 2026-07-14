package log

import (
	"io"
	"os"
)

// TailRecent returns up to maxLines from the end of the active log file.
// Used by realtime log WebSocket to seed the viewer on connect.
func TailRecent(maxLines int) []string {
	if maxLines <= 0 {
		maxLines = 200
	}
	path := FilePath()
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	lines, err := readLastLines(f, maxLines)
	if err != nil {
		return nil
	}
	return lines
}

func readLastLines(f *os.File, maxLines int) ([]string, error) {
	const chunk = 32 * 1024
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()
	if size <= 0 {
		return nil, nil
	}

	var (
		data   []byte
		offset = size
	)
	for offset > 0 && countNewlines(data) <= maxLines {
		readSize := int64(chunk)
		if offset < readSize {
			readSize = offset
		}
		offset -= readSize
		buf := make([]byte, readSize)
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return nil, err
		}
		n, err := io.ReadFull(f, buf)
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			return nil, err
		}
		data = append(buf[:n], data...)
		if offset == 0 {
			break
		}
	}

	raw := splitLines(string(data))
	if len(raw) > maxLines {
		raw = raw[len(raw)-maxLines:]
	}
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		if line != "" {
			out = append(out, line)
		}
	}
	return out, nil
}

func countNewlines(b []byte) int {
	n := 0
	for _, c := range b {
		if c == '\n' {
			n++
		}
	}
	return n
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	out := make([]string, 0, 64)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			out = append(out, line)
			start = i + 1
		}
	}
	if start < len(s) {
		line := s[start:]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		out = append(out, line)
	}
	return out
}
