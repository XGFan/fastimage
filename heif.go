package fastimage

import "math"

// maxBoxes bounds how many sibling boxes are walked at one level.
const maxBoxes = 1024

// Box types as big-endian FourCC values.
const (
	boxMeta = 'm'<<24 | 'e'<<16 | 't'<<8 | 'a'
	boxPitm = 'p'<<24 | 'i'<<16 | 't'<<8 | 'm'
	boxIprp = 'i'<<24 | 'p'<<16 | 'r'<<8 | 'p'
	boxIpco = 'i'<<24 | 'p'<<16 | 'c'<<8 | 'o'
	boxIpma = 'i'<<24 | 'p'<<16 | 'm'<<8 | 'a'
	boxIspe = 'i'<<24 | 's'<<16 | 'p'<<8 | 'e'
	boxClap = 'c'<<24 | 'l'<<16 | 'a'<<8 | 'p'
	boxIrot = 'i'<<24 | 'r'<<16 | 'o'<<8 | 't'
	boxImir = 'i'<<24 | 'm'<<16 | 'i'<<8 | 'r'
	boxJxlc = 'j'<<24 | 'x'<<16 | 'l'<<8 | 'c'
	boxJxlp = 'j'<<24 | 'x'<<16 | 'l'<<8 | 'p'
)

// ftypType classifies an ISOBMFF file by its ftyp brands; videos and other
// ISOBMFF files are Unknown.
func ftypType(s *source) Type {
	size, ok := s.be32(0)
	if !ok || size < 16 {
		return Unknown
	}
	b := s.peek(8, int(min(size, 256))-8)
	heif := false
	for i := 0; i+4 <= len(b); i += 4 {
		if i == 4 {
			continue // minor_version
		}
		switch string(b[i : i+4]) {
		case "avif", "avis":
			return AVIF
		case "heic", "heix", "heim", "heis", "hevc", "hevx", "hevm", "hevs":
			return HEIC
		case "mif1", "msf1":
			heif = true
		}
	}
	if heif {
		return HEIF
	}
	return Unknown
}

// box reads the box header at off. It returns the box type, where its payload
// starts and where the box ends; boxes extending past limit are rejected.
func box(s *source, off, limit int64) (typ uint32, body, end int64, ok bool) {
	b := s.at(off, 8)
	if b == nil {
		return
	}
	size := int64(bigEndian.Uint32(b))
	typ = bigEndian.Uint32(b[4:]) // b is invalid after the next read
	body = off + 8
	switch size {
	case 0: // extends to the end of the enclosing box or file
		size = limit - off
	case 1:
		l := s.at(body, 8)
		if l == nil {
			return
		}
		u := uint64(bigEndian.Uint32(l))<<32 | uint64(bigEndian.Uint32(l[4:]))
		if u > math.MaxInt64 {
			return
		}
		size = int64(u)
		body += 8
	}
	if size < body-off || size > limit-off {
		return
	}
	return typ, body, off + size, true
}

// heif reads the primary item's ispe (size), clap (crop) and irot/imir
// (orientation) properties from the top-level meta box.
func heif(s *source, info *Info) {
	off := int64(0)
	for i := 0; i < maxBoxes; i++ {
		typ, body, end, ok := box(s, off, math.MaxInt64)
		if !ok {
			return
		}
		if typ == boxMeta {
			heifMeta(s, body+4, end, info) // skip FullBox version/flags
			return
		}
		off = end
	}
}

func heifMeta(s *source, off, end int64, info *Info) {
	var primary uint32
	var hasPrimary bool
	var ipcoBody, ipcoEnd, ipmaBody, ipmaEnd int64
	for i := 0; i < maxBoxes && off < end; i++ {
		typ, body, next, ok := box(s, off, end)
		if !ok {
			return
		}
		switch typ {
		case boxPitm:
			v, ok := s.u8(body)
			if !ok {
				return
			}
			if v == 0 {
				b := s.at(body+4, 2)
				if b == nil {
					return
				}
				primary = uint32(bigEndian.Uint16(b))
			} else {
				b := s.at(body+4, 4)
				if b == nil {
					return
				}
				primary = bigEndian.Uint32(b)
			}
			hasPrimary = true
		case boxIprp:
			for j, o := 0, body; j < maxBoxes && o < next; j++ {
				t, b, n, ok := box(s, o, next)
				if !ok {
					break
				}
				switch t {
				case boxIpco:
					ipcoBody, ipcoEnd = b, n
				case boxIpma:
					if ipmaEnd == 0 {
						ipmaBody, ipmaEnd = b, n
					}
				}
				o = n
			}
		}
		off = next
	}
	if !hasPrimary || ipcoEnd == 0 || ipmaEnd == 0 {
		return
	}

	h := s.at(ipmaBody, 8)
	if h == nil {
		return
	}
	version, wide := h[0], h[3]&1 == 1
	count := bigEndian.Uint32(h[4:])
	idSize := int64(2)
	if version >= 1 {
		idSize = 4
	}
	idxSize := int64(1)
	if wide {
		idxSize = 2
	}
	// Results are kept local so a file cut off midway reports nothing.
	var size Info
	var t transform
	o := ipmaBody + 8
	for i := uint32(0); i < count && o < ipmaEnd; i++ {
		b := s.at(o, int(idSize)+1)
		if b == nil {
			return
		}
		var id uint32
		if idSize == 2 {
			id = uint32(bigEndian.Uint16(b))
		} else {
			id = bigEndian.Uint32(b)
		}
		n := int64(b[idSize])
		o += idSize + 1
		if id != primary {
			o += n * idxSize
			continue
		}
		for k := int64(0); k < n; k++ {
			a := s.at(o, int(idxSize))
			if a == nil {
				return
			}
			var index int
			if wide {
				index = int(bigEndian.Uint16(a) & 0x7fff)
			} else {
				index = int(a[0] & 0x7f)
			}
			o += idxSize
			if !heifProperty(s, ipcoBody, ipcoEnd, index, &size, &t) {
				return
			}
		}
		info.Width, info.Height = size.Width, size.Height
		if t != (transform{}) {
			info.Orientation = t.orientation()
		}
		return
	}
}

// heifProperty applies the index-th (1-based) property box of ipco. It
// returns false if the input ends before the property could be read.
func heifProperty(s *source, off, end int64, index int, info *Info, t *transform) bool {
	if index == 0 || index > maxBoxes {
		return true
	}
	for i := 1; off < end; i++ {
		typ, body, next, ok := box(s, off, end)
		if !ok {
			return false
		}
		if i < index {
			off = next
			continue
		}
		switch typ {
		case boxIspe:
			b := s.at(body+4, 8)
			if b == nil {
				return false
			}
			if body+12 <= next {
				info.Width = bigEndian.Uint32(b)
				info.Height = bigEndian.Uint32(b[4:])
			}
		case boxClap:
			b := s.at(body, 16)
			if b == nil {
				return false
			}
			if body+16 <= next {
				if w := ratio(bigEndian.Uint32(b), bigEndian.Uint32(b[4:])); w > 0 {
					info.Width = w
				}
				if h := ratio(bigEndian.Uint32(b[8:]), bigEndian.Uint32(b[12:])); h > 0 {
					info.Height = h
				}
			}
		case boxIrot:
			b, ok := s.u8(body)
			if !ok {
				return false
			}
			if body < next {
				// counter-clockwise quarter turns
				t.rotate(4 - int(b&3))
			}
		case boxImir:
			b, ok := s.u8(body)
			if !ok {
				return false
			}
			if body < next {
				if b&1 == 1 {
					t.flipH() // left and right exchanged
				} else {
					t.flipH()
					t.rotate(2) // top and bottom exchanged
				}
			}
		}
		return true
	}
	return true
}

// ratio rounds the clap fraction n/d; it returns 0 for invalid fractions.
func ratio(n, d uint32) uint32 {
	if d == 0 || int32(n) <= 0 || int32(d) <= 0 {
		return 0
	}
	return (n + d/2) / d
}

// transform is "mirror horizontally if flip, then rotate clockwise by rot
// quarter turns", the decomposition EXIF orientations use.
type transform struct {
	flip bool
	rot  int
}

// rotate appends a clockwise rotation of q quarter turns.
func (t *transform) rotate(q int) { t.rot = (t.rot + q) % 4 }

// flipH appends a horizontal mirror: H·R(r) = R(-r)·H.
func (t *transform) flipH() {
	t.flip = !t.flip
	t.rot = (4 - t.rot) % 4
}

func (t transform) orientation() uint8 {
	if !t.flip {
		return [4]uint8{1, 6, 3, 8}[t.rot]
	}
	return [4]uint8{2, 7, 4, 5}[t.rot]
}
