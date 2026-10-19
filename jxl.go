package fastimage

import "math"

// jxl reads the SizeHeader and the orientation from ImageMetadata, from a bare
// codestream or from the first jxlc/jxlp box of a container.
func jxl(s *source, info *Info) {
	off := int64(0)
	if b := s.at(0, 2); b != nil && b[0] == 0 { // container
		off = -1
		for i, o := 0, int64(0); i < maxBoxes; i++ {
			typ, body, end, ok := box(s, o, math.MaxInt64)
			if !ok {
				return
			}
			if typ == boxJxlc {
				off = body
				break
			}
			if typ == boxJxlp {
				off = body + 4 // part index
				break
			}
			o = end
		}
		if off < 0 {
			return
		}
	}
	b := s.peek(off, 16)
	if len(b) < 2 || b[0] != 0xff || b[1] != 0x0a {
		return
	}
	r := bitReader{b: b[2:]}

	var w, h uint32
	small := r.read(1) == 1
	if small {
		h = (r.read(5) + 1) * 8
	} else {
		h = r.u32size()
	}
	if ratio := r.read(3); ratio != 0 {
		num := [8]uint64{0, 1, 12, 4, 3, 16, 5, 2}[ratio]
		den := [8]uint64{0, 1, 10, 3, 2, 9, 4, 1}[ratio]
		w = uint32(uint64(h) * num / den)
	} else if small {
		w = (r.read(5) + 1) * 8
	} else {
		w = r.u32size()
	}

	var orientation uint8
	if allDefault := r.read(1) == 1; !allDefault {
		if extraFields := r.read(1) == 1; extraFields {
			orientation = uint8(r.read(3)) + 1
		}
	}
	if r.bad {
		return
	}
	info.Width, info.Height, info.Orientation = w, h, orientation
}

// bitReader reads JPEG XL's LSB-first bit stream.
type bitReader struct {
	b   []byte
	pos int
	bad bool
}

func (r *bitReader) read(n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		if r.pos>>3 >= len(r.b) {
			r.bad = true
			return 0
		}
		v |= uint32(r.b[r.pos>>3]>>(r.pos&7)&1) << i
		r.pos++
	}
	return v
}

// u32size reads a SizeHeader dimension: U32(1+u(9), 1+u(13), 1+u(18), 1+u(30)).
func (r *bitReader) u32size() uint32 {
	return 1 + r.read([4]int{9, 13, 18, 30}[r.read(2)])
}
