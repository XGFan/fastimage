package fastimage

import "math"

func jpeg(s *source, info *Info) {
	off := int64(2)
	exif := false
	for {
		b := s.at(off, 4)
		if b == nil {
			return
		}
		marker := b[1]
		switch {
		case b[0] != 0xff || marker == 0x00: // junk between segments, skipped like libjpeg does
			off++
			continue
		case marker == 0xff: // fill byte
			off++
			continue
		case marker == 0x01 || marker >= 0xd0 && marker <= 0xd8: // TEM, RSTn, SOI carry no length
			off += 2
			continue
		case marker == 0xd9 || marker == 0xda: // EOI or SOS before any frame header
			return
		}
		length := int64(b[2])<<8 | int64(b[3])
		if length < 2 {
			return
		}
		switch {
		case marker >= 0xc0 && marker <= 0xcf && marker != 0xc4 && marker != 0xc8 && marker != 0xcc: // SOFn
			f := s.at(off+5, 4)
			if f == nil {
				return
			}
			info.Height = uint32(bigEndian.Uint16(f))
			info.Width = uint32(bigEndian.Uint16(f[2:]))
			return
		case marker == 0xe1 && !exif: // APP1, possibly Exif; only the first Exif counts
			if h := s.at(off+4, 6); h != nil && string(h) == "Exif\x00\x00" {
				exif = true
				t, ok := readTIFF(s, off+10, off+2+length)
				if !ok {
					return
				}
				info.Orientation = t.orientation
			}
		}
		off += 2 + length
	}
}

func tiff(s *source, info *Info) {
	t, ok := readTIFF(s, 0, math.MaxInt64)
	if !ok {
		return
	}
	info.Width, info.Height, info.Orientation = t.width, t.height, t.orientation
}

type tiffTags struct {
	width, height uint32
	orientation   uint8
}

// readTIFF reads ImageWidth, ImageLength and Orientation from IFD0 of the TIFF
// structure whose header is at base. Offsets inside it are relative to base and
// nothing at or beyond end is read. ok is false when the input ends early.
func readTIFF(s *source, base, end int64) (t tiffTags, ok bool) {
	h := s.at(base, 8)
	if h == nil || base+8 > end {
		return t, h != nil
	}
	var order byteOrder
	switch string(h[:4]) {
	case "II\x2a\x00":
		order = littleEndian
	case "MM\x00\x2a":
		order = bigEndian
	default:
		return t, true
	}
	ifd := base + int64(order.Uint32(h[4:]))
	if ifd+2 > end { // never follow offsets out of the structure
		return t, true
	}
	c := s.at(ifd, 2)
	if c == nil {
		return t, false
	}
	n := int64(order.Uint16(c))
	for e := ifd + 2; e < ifd+2+12*n && e+12 <= end; e += 12 {
		b := s.at(e, 12)
		if b == nil {
			return t, false
		}
		tag := order.Uint16(b)
		if tag > 274 { // entries are sorted by tag
			break
		}
		var v uint32
		switch order.Uint16(b[2:]) {
		case 1, 6: // (S)BYTE
			v = uint32(b[8])
		case 3, 8: // (S)SHORT
			v = uint32(order.Uint16(b[8:]))
		case 4, 9: // (S)LONG
			v = order.Uint32(b[8:])
		default:
			continue
		}
		switch tag {
		case 256:
			t.width = v
		case 257:
			t.height = v
		case 274:
			if v >= 1 && v <= 8 {
				t.orientation = uint8(v)
			}
		}
	}
	return t, true
}
