package log

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FileHandler writes log lines to a file on disk.
// It is a thread-safe impelmentation.
type FileHandler struct {
	mu         *sync.Mutex
	wr         *bufio.Writer
	buff       []byte
	level      slog.Level
	groups     []string
	attrs      []slog.Attr
	path       string
	name       string
	currPrefix string
	delimiter  byte
}

// NewFileHandler creates a new file handler
func NewFileHandler(path string, name string) slog.Handler {
	return &FileHandler{
		mu:        &sync.Mutex{},
		level:     slog.Level(math.MinInt),
		path:      path,
		name:      name,
		delimiter: ';',
	}
}

// NewfileHandler creates a new file handler that only logs lines with a level
// equalt to or higher than the provided level.
func NewFileHandlerWithLevel(path string, name string, l slog.Level) slog.Handler {
	return &FileHandler{
		mu:        &sync.Mutex{},
		level:     l,
		path:      path,
		name:      name,
		delimiter: ';',
	}
}

// GetDelimiter returns the delimiter used in the log file.
// Default delimiter = ';'
func (h *FileHandler) GetDelimiter() byte {
	return h.delimiter
}

// SetDelimiter allows you to set the delimiter
func (h *FileHandler) SetDelimiter(d byte) {
	h.delimiter = d
}

// Handle implements slog.Handler
func (h *FileHandler) Handle(ctx context.Context, r slog.Record) error {
	if !h.Enabled(ctx, r.Level) {
		return nil
	}

	go func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		if err := h.ensureWriter(r.Time); err != nil {
			return
		}

		buff := h.format(h.buff, r)
		h.wr.Write(buff)
		h.wr.Flush()
		h.buff = buff[:0]
	}()
	return nil
}

// Enabled implements slog.Handler
func (h *FileHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level
}

// WithGroup implements slog.Handler
func (h *FileHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	groups := make([]string, 0, len(h.groups)+1)
	if len(h.groups) > 0 {
		groups = append(groups, h.groups...)
	}
	groups = append(groups, name)

	return &FileHandler{
		mu:         h.mu,
		wr:         h.wr,
		level:      h.level,
		groups:     groups,
		attrs:      h.attrs[:],
		path:       h.path,
		name:       h.name,
		currPrefix: h.currPrefix,
	}
}

// WithAttrs implements slog.Handler.
func (h *FileHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	attrList := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	if len(h.attrs) > 0 {
		attrList = append(attrList, h.attrs...)
	}
	if len(h.groups) > 0 {
		for _, attr := range attrs {
			key := strings.Join(h.groups, ".")
			key = strings.Join([]string{key, attr.Key}, ".")
			attr.Key = key
			attrList = append(attrList, attr)
		}
	} else {
		attrList = append(attrList, attrs...)
	}

	return &FileHandler{
		mu:         h.mu,
		wr:         h.wr,
		level:      h.level,
		groups:     h.groups[:],
		attrs:      attrList,
		path:       h.path,
		name:       h.name,
		currPrefix: h.currPrefix,
	}
}

// ensureWrite makes sure the correct log file is opened and available for
// writing.
func (h *FileHandler) ensureWriter(t time.Time) error {
	prefix := t.Format("2006-01-02")
	if h.wr == nil {
		return h.openFile(prefix)
	}
	if h.currPrefix != prefix {
		// We past midnight, new file
		h.wr.Flush()
		return h.openFile(prefix)
	}
	return nil
}

// openFile opens the log file for the provided prefix
func (h *FileHandler) openFile(prefix string) error {
	h.currPrefix = prefix
	fpath := path.Join(h.path, fmt.Sprintf("%s_%s.log", prefix, h.name))

	of := os.O_WRONLY
	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		of |= os.O_CREATE
	} else {
		of |= os.O_APPEND
	}

	fh, err := os.OpenFile(fpath, of, 0700)
	if err != nil {
		return err
	}

	h.wr = bufio.NewWriter(fh)
	return nil
}

// format formats the slog record as a CSV file.
func (h *FileHandler) format(buf []byte, r slog.Record) []byte {
	if buf == nil {
		buf = make([]byte, 0, 70)
	}
	b := bytes.NewBuffer(buf)

	msg := strconv.Quote(r.Message)

	b.WriteString(r.Level.String())
	b.WriteByte(h.delimiter)
	h.writeTimeFormat(b, r.Time)
	b.WriteByte(h.delimiter)
	b.WriteString(msg)
	h.formatAttributes(b, r)
	b.WriteByte('\n')
	return b.Bytes()
}

func (h *FileHandler) writeTimeFormat(buf *bytes.Buffer, t time.Time) {
	buf.WriteString(t.Format("2006-01-02 15:04:05.000"))
}

func (h *FileHandler) formatAttributes(buf *bytes.Buffer, r slog.Record) {
	writeAttr := func(attr slog.Attr) {
		buf.WriteByte(h.delimiter)
		buf.WriteString(strconv.Quote(attr.String()))
	}

	for _, attr := range h.attrs {
		writeAttr(attr)
	}
	r.Attrs(func(attr slog.Attr) bool {
		if len(h.groups) > 0 {
			key := strings.Join(h.groups, ".")
			key = strings.Join([]string{key, attr.Key}, ".")
			writeAttr(slog.Attr{
				Key:   key,
				Value: attr.Value,
			})
		} else {
			writeAttr(attr)
		}
		return true
	})
}
