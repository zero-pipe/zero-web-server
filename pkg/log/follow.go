package log

import (
	"io"
	"os"
)

// FileFollower follows the active log file like `tail -f`.
type FileFollower struct {
	path    string
	offset  int64
	partial string
}

// NewFileFollower opens follow state at end of file (call SeedRecent first if needed).
func NewFileFollower() *FileFollower {
	path := FilePath()
	f := &FileFollower{path: path}
	if path == "" {
		return f
	}
	if st, err := os.Stat(path); err == nil {
		f.offset = st.Size()
	}
	return f
}

// SeekEnd moves the follow cursor to current EOF (after seeding historical lines).
func (f *FileFollower) SeekEnd() {
	if f == nil || f.path == "" {
		return
	}
	if st, err := os.Stat(f.path); err == nil {
		f.offset = st.Size()
	}
	f.partial = ""
}

// Poll reads newly appended complete lines since last offset.
func (f *FileFollower) Poll() []string {
	if f == nil || f.path == "" {
		return nil
	}
	st, err := os.Stat(f.path)
	if err != nil {
		return nil
	}
	size := st.Size()
	// 日志轮转：文件变小则从头跟
	if size < f.offset {
		f.offset = 0
		f.partial = ""
	}
	if size == f.offset {
		return nil
	}

	file, err := os.Open(f.path)
	if err != nil {
		return nil
	}
	defer file.Close()

	if _, err := file.Seek(f.offset, io.SeekStart); err != nil {
		return nil
	}
	buf := make([]byte, size-f.offset)
	n, err := io.ReadFull(file, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil
	}
	f.offset += int64(n)

	text := f.partial + string(buf[:n])
	f.partial = ""
	parts := splitLines(text)
	// splitLines 会保留最后一段；若原文不以换行结束，最后一段是未完成行
	if n > 0 && buf[n-1] != '\n' {
		if len(parts) == 0 {
			f.partial = text
			return nil
		}
		f.partial = parts[len(parts)-1]
		parts = parts[:len(parts)-1]
	}

	out := make([]string, 0, len(parts))
	for _, line := range parts {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
