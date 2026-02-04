package monitor

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// FileLogReader reads new lines from a log file using position tracking
type FileLogReader struct {
	logFile  string
	posFile  string
	maxLines int
}

// NewFileLogReader creates a new file-based log reader
func NewFileLogReader(logFile, posFile string) *FileLogReader {
	return &FileLogReader{
		logFile:  logFile,
		posFile:  posFile,
		maxLines: 100,
	}
}

// ReadNewLines reads new lines since last read
func (r *FileLogReader) ReadNewLines() ([]string, error) {
	lastPos := r.loadPosition()

	f, err := os.Open(r.logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	currentSize := info.Size()

	// Log rotation detected
	if currentSize < lastPos {
		lastPos = 0
	}

	if lastPos > 0 {
		if _, err := f.Seek(lastPos, 0); err != nil {
			return nil, err
		}
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	count := 0

	for scanner.Scan() && count < r.maxLines {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return lines, err
	}

	// Save actual read position, not initial file size
	newPos, err := f.Seek(0, 1) // Get current position
	if err != nil {
		newPos = currentSize // Fallback to file size
	}
	r.savePosition(newPos)

	return lines, nil
}

func (r *FileLogReader) loadPosition() int64 {
	data, err := os.ReadFile(r.posFile)
	if err != nil {
		return 0
	}
	pos, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return pos
}

func (r *FileLogReader) savePosition(pos int64) error {
	return os.WriteFile(r.posFile, []byte(strconv.FormatInt(pos, 10)), 0644)
}
