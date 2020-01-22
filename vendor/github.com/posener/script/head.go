package script

import (
	"bytes"
	"fmt"
	"io"
)

// Head reads only the n first lines of the given reader. If n is a negative number, all lines
// besides n first line will be read.
//
// Shell command: `head -n <n>`
func (s Stream) Head(n int) Stream {
	h := &head{n: n}
	if n < 0 {
		h.n = -n
		h.after = true
	}
	return s.Modify(h)
}

type head struct {
	n     int
	after bool
}

func (h *head) Modify(line []byte) ([]byte, error) {
	if line == nil || (h.n == 0 && !h.after) {
		return nil, io.EOF
	}
	print := !h.after || h.n == 0
	h.n--
	if print {
		return append(line, '\n'), nil
	}
	return nil, nil
}

func (h *head) Name() string {
	return fmt.Sprintf("head(%d)", h.n)
}

// Tail reads only the n last lines of the given reader. If n is a negative number, all lines
// besides the last n lines will be read.
//
// Shell command: `tail -n <n>`
func (s Stream) Tail(n int) Stream {
	t := &tail{n: n}
	if n < 0 {
		t.n = -n
		t.before = true
	}
	t.lines = make([][]byte, 0, t.n)
	return s.Modify(t)
}

type tail struct {
	n      int
	lines  [][]byte
	buf    bytes.Buffer
	before bool
}

func (t *tail) Modify(line []byte) ([]byte, error) {
	if t.n == 0 {
		return nil, io.EOF
	}
	if line == nil {
		if t.before {
			if n := len(t.lines) - t.n; n > 0 {
				t.lines = t.lines[:n]
			} else {
				return nil, io.EOF
			}
		}
		return append(bytes.Join(t.lines, []byte{'\n'}), '\n'), io.EOF
	}

	// Shift all lines and append the new line.
	if t.before || len(t.lines) < cap(t.lines) {
		t.lines = append(t.lines, line)
	} else {
		for i := 0; i < len(t.lines)-1; i++ {
			t.lines[i] = t.lines[i+1]
		}
		t.lines[len(t.lines)-1] = line
	}

	return nil, nil
}

func (t *tail) Name() string {
	return fmt.Sprintf("tail(%d)", t.n)
}
