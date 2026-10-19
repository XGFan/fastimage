package fastimage

import (
	"bytes"
	"math"
)

// textHeader is how much of a text-based header (netpbm, XBM, XPM) is searched.
const textHeader = windowSize

func netpbm(s *source, info *Info) {
	b := s.peek(0, textHeader)
	if len(b) < 3 {
		return
	}
	tok, i := pnmToken(b, 2)
	if b[1] == '7' && string(tok) != "332" {
		// PAM: "WIDTH n" / "HEIGHT n" lines up to ENDHDR
		for tok != nil && string(tok) != "ENDHDR" {
			key := tok
			tok, i = pnmToken(b, i)
			switch string(key) {
			case "WIDTH":
				info.Width, _ = atou(tok)
			case "HEIGHT":
				info.Height, _ = atou(tok)
			}
		}
		return
	}
	if b[1] == '7' { // XV thumbnail: "P7 332" then comments, width, height
		tok, i = pnmToken(b, i)
	}
	w, ok := atou(tok)
	if !ok {
		return
	}
	tok, _ = pnmToken(b, i)
	if h, ok := atou(tok); ok {
		info.Width, info.Height = w, h
	}
}

// pnmToken returns the next token at or after i, skipping whitespace and
// '#' comments. It returns nil if the token is not terminated within b.
func pnmToken(b []byte, i int) (tok []byte, next int) {
	for i < len(b) {
		switch c := b[i]; {
		case c == '#':
			for i < len(b) && b[i] != '\n' && b[i] != '\r' {
				i++
			}
		case isSpace(c):
			i++
		default:
			j := i
			for j < len(b) && !isSpace(b[j]) && b[j] != '#' {
				j++
			}
			if j == len(b) {
				return nil, j
			}
			return b[i:j], j
		}
	}
	return nil, i
}

// atou parses a non-empty decimal that fits in uint32.
func atou(b []byte) (n uint32, ok bool) {
	if len(b) == 0 || len(b) > 10 {
		return 0, false
	}
	var v uint64
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, false
		}
		v = v*10 + uint64(c-'0')
	}
	if v > math.MaxUint32 {
		return 0, false
	}
	return uint32(v), true
}

// xbm parses "#define name_width W" followed by "#define name_height H".
func xbm(s *source, info *Info) {
	b := s.peek(0, textHeader)
	w, i, ok := xbmDefine(b, 0, "_width")
	if !ok {
		return
	}
	if h, _, ok := xbmDefine(b, i, "_height"); ok {
		info.Width, info.Height = w, h
	}
}

func xbmDefine(b []byte, i int, suffix string) (uint32, int, bool) {
	var define, name, value []byte
	define, i = field(b, i)
	name, i = field(b, i)
	value, i = field(b, i)
	if string(define) != "#define" || !bytes.HasSuffix(name, []byte(suffix)) {
		return 0, i, false
	}
	n, ok := atou(value)
	return n, i, ok
}

// field returns the next whitespace-delimited field at or after i; nil if it
// is not terminated within b.
func field(b []byte, i int) ([]byte, int) {
	for i < len(b) && isSpace(b[i]) {
		i++
	}
	j := i
	for j < len(b) && !isSpace(b[j]) {
		j++
	}
	if i == j || j == len(b) {
		return nil, j
	}
	return b[i:j], j
}

// xpm parses the values string, the first C string after the "/* XPM */" header:
// "<width> <height> <colors> <chars per pixel>".
func xpm(s *source, info *Info) {
	b := s.peek(0, textHeader)
	for i := len("/* XPM */"); i < len(b); i++ {
		switch b[i] {
		case '/':
			if i+1 < len(b) && b[i+1] == '*' {
				end := bytes.Index(b[i+2:], []byte("*/"))
				if end < 0 {
					return
				}
				i += 2 + end + 1
			}
		case '"':
			w, j := field(b, i+1)
			h, _ := field(b, j)
			wv, ok1 := atou(w)
			hv, ok2 := atou(h)
			if ok1 && ok2 {
				info.Width, info.Height = wv, hv
			}
			return
		}
	}
}
