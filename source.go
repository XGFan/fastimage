package fastimage

import (
	"io"
	"math"
	"sync"
)

// windowSize is how much a reader-backed source fetches per read. Most headers
// fit in one window; JPEG needs one more after skipping a large APP segment.
const windowSize = 4096

// maxPeek bounds a single at() request so hostile lengths cannot force huge reads.
const maxPeek = 1 << 16

// maxFills bounds how many windows one call may read (at most ~4 MB), so
// offsets crafted to jump around a large file cannot turn into thousands of
// reads. Only files with over a thousand >4 KB segments before the size hit it.
const maxFills = 1024

// maxStream bounds how far into a forward-only reader GetInfoReader will go.
const maxStream = 64 << 20

// source gives bounds-checked random access to image bytes. In bytes mode buf
// holds the whole input and r is nil; in reader mode buf is a window at offset
// off that is refilled from r on demand.
type source struct {
	buf   []byte
	off   int64
	r     io.ReaderAt
	mem   *[]byte
	fills int
}

// at returns n bytes at offset off, or nil if they are not available.
// In reader mode the result is only valid until the next at or peek.
func (s *source) at(off int64, n int) []byte {
	if off >= s.off && n >= 0 && off-s.off <= int64(len(s.buf))-int64(n) {
		i := off - s.off
		return s.buf[i : i+int64(n)]
	}
	if s.r == nil {
		return nil
	}
	return s.fill(off, n, n)
}

// peek returns up to n bytes at offset off; it is shorter only at end of input.
func (s *source) peek(off int64, n int) []byte {
	if off >= s.off && n >= 0 && off-s.off <= int64(len(s.buf))-int64(n) {
		i := off - s.off
		return s.buf[i : i+int64(n)]
	}
	if s.r == nil {
		if off < 0 || off >= int64(len(s.buf)) || n < 0 {
			return nil
		}
		return s.buf[off : off+min(int64(len(s.buf))-off, int64(n))]
	}
	return s.fill(off, n, 1)
}

// fill moves the window to off and returns at least need (at most n) bytes.
// Bytes already in the window are kept, so the underlying reader only ever
// moves forward when requests do.
func (s *source) fill(off int64, n, need int) []byte {
	if off < 0 || n < 0 || n > maxPeek || off > math.MaxInt64-maxPeek || s.fills >= maxFills {
		return nil
	}
	s.fills++
	if s.mem == nil {
		s.mem = bufPool.Get().(*[]byte)
	}
	size := max(n, windowSize)
	if cap(*s.mem) < size {
		*s.mem = make([]byte, size)
	}
	buf := (*s.mem)[:size]
	kept := 0
	if off >= s.off && off < s.off+int64(len(s.buf)) {
		kept = copy(buf, s.buf[off-s.off:])
	}
	nr, _ := s.r.ReadAt(buf[kept:], off+int64(kept))
	nr += kept
	s.buf, s.off = buf[:nr], off
	if nr < need {
		return nil
	}
	return s.buf[:min(n, nr)]
}

func (s *source) release() {
	if s.mem != nil {
		bufPool.Put(s.mem)
		s.mem, s.buf = nil, nil
	}
}

var bufPool = sync.Pool{New: func() any {
	b := make([]byte, windowSize)
	return &b
}}

func (s *source) u8(off int64) (uint8, bool) {
	b := s.at(off, 1)
	if b == nil {
		return 0, false
	}
	return b[0], true
}

func (s *source) be32(off int64) (uint32, bool) {
	b := s.at(off, 4)
	if b == nil {
		return 0, false
	}
	return bigEndian.Uint32(b), true
}

// seekReaderAt adapts an io.ReadSeeker. Offsets are relative to the position
// the reader had when it was handed to GetInfoReader.
type seekReaderAt struct {
	r    io.ReadSeeker
	base int64
}

func (s *seekReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if _, err := s.r.Seek(s.base+off, io.SeekStart); err != nil {
		return 0, err
	}
	return io.ReadFull(s.r, p)
}

// streamReaderAt adapts a plain io.Reader by only moving forward: earlier
// offsets are served from the last read, anything before it is unavailable.
type streamReaderAt struct {
	r   io.Reader
	pos int64
}

func (s *streamReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off < s.pos {
		return 0, io.ErrUnexpectedEOF
	}
	if off > maxStream {
		return 0, io.ErrUnexpectedEOF
	}
	if off > s.pos {
		n, err := io.CopyN(io.Discard, s.r, off-s.pos)
		s.pos += n
		if err != nil {
			return 0, err
		}
	}
	n, err := io.ReadFull(s.r, p)
	s.pos += int64(n)
	return n, err
}
