package fastimage

import "io"

// Type represents the type of the image detected, or `Unknown`.
type Type uint64

const (
	// Unknown represents an unknown image type
	Unknown Type = iota
	// BMP represendts a BMP image
	BMP
	// BPM is a misspelling of PBM kept for compatibility; it is never returned.
	//
	// Deprecated: binary PBM (P4) is reported as PBM.
	BPM
	// GIF represendts a GIF image
	GIF
	// JPEG represendts a JPEG image
	JPEG
	// MNG represendts a MNG image
	MNG
	// PBM represendts a PBM image
	PBM
	// PCX represendts a PCX image
	PCX
	// PGM represendts a PGM image
	PGM
	// PNG represendts a PNG image
	PNG
	// PPM represendts a PPM image
	PPM
	// PSD represendts a PSD image
	PSD
	// RAS represendts a RAS image
	RAS
	// RGB represendts a RGB image
	RGB
	// TIFF represendts a TIFF image
	TIFF
	// WEBP represendts a WEBP image
	WEBP
	// XBM represendts a XBM image
	XBM
	// XPM represendts a XPM image
	XPM
	// XV represendts a XV image
	XV
	// AVIF represents an AVIF image
	AVIF
	// HEIC represents a HEVC-coded HEIF image
	HEIC
	// HEIF represents a HEIF image with another or unknown codec
	HEIF
	// JXL represents a JPEG XL image
	JXL
	// ICO represents a Windows icon
	ICO
	// CUR represents a Windows cursor
	CUR
	// QOI represents a QOI image
	QOI
)

// String return a lower name of image type
func (t Type) String() string {
	switch t {
	case BMP:
		return "bmp"
	case BPM:
		return "bpm"
	case GIF:
		return "gif"
	case JPEG:
		return "jpeg"
	case MNG:
		return "mng"
	case PBM:
		return "pbm"
	case PCX:
		return "pcx"
	case PGM:
		return "pgm"
	case PNG:
		return "png"
	case PPM:
		return "ppm"
	case PSD:
		return "psd"
	case RAS:
		return "ras"
	case RGB:
		return "rgb"
	case TIFF:
		return "tiff"
	case WEBP:
		return "webp"
	case XBM:
		return "xbm"
	case XPM:
		return "xpm"
	case XV:
		return "xv"
	case AVIF:
		return "avif"
	case HEIC:
		return "heic"
	case HEIF:
		return "heif"
	case JXL:
		return "jxl"
	case ICO:
		return "ico"
	case CUR:
		return "cur"
	case QOI:
		return "qoi"
	}
	return ""
}

// Mime return mime type of image type
func (t Type) Mime() string {
	switch t {
	case BMP:
		return "image/bmp"
	case BPM:
		return "image/x-portable-bitmap"
	case GIF:
		return "image/gif"
	case JPEG:
		return "image/jpeg"
	case MNG:
		return "video/x-mng"
	case PBM:
		return "image/x-portable-bitmap"
	case PCX:
		return "image/x-pcx"
	case PGM:
		return "image/x-portable-graymap"
	case PNG:
		return "image/png"
	case PPM:
		return "image/x-portable-pixmap"
	case PSD:
		return "image/vnd.adobe.photoshop"
	case RAS:
		return "image/x-cmu-raster"
	case RGB:
		return "image/x-rgb"
	case TIFF:
		return "image/tiff"
	case WEBP:
		return "image/webp"
	case XBM:
		return "image/x-xbitmap"
	case XPM:
		return "image/x-xpixmap"
	case XV:
		return "image/x-portable-pixmap"
	case AVIF:
		return "image/avif"
	case HEIC:
		return "image/heic"
	case HEIF:
		return "image/heif"
	case JXL:
		return "image/jxl"
	case ICO:
		return "image/vnd.microsoft.icon"
	case CUR:
		return "image/x-win-bitmap"
	case QOI:
		return "image/qoi"
	}
	return ""
}

// Info holds the type and dimensions of an image.
//
// Width and Height are the stored pixel dimensions. Orientation is the EXIF
// orientation (1-8) taken from JPEG/TIFF Exif, HEIF/AVIF irot/imir or the JPEG XL
// header; it is 0 when the image does not specify one. Use DisplaySize for the
// size the image is shown at.
type Info struct {
	Type        Type
	Width       uint32
	Height      uint32
	Orientation uint8
}

// DisplaySize returns the image size after applying Orientation, i.e. with
// width and height swapped for orientations 5-8.
func (info Info) DisplaySize() (width, height uint32) {
	if info.Orientation >= 5 && info.Orientation <= 8 {
		return info.Height, info.Width
	}
	return info.Width, info.Height
}

// GetType detects the image type of data from its signature.
func GetType(p []byte) Type {
	s := source{buf: p}
	return s.detect()
}

// GetInfo detects the image info of data. Formats whose size lives behind
// variable-length metadata (JPEG, TIFF, HEIF, ...) need enough of the file in p;
// use GetInfoReader to read only what is needed.
func GetInfo(p []byte) Info {
	s := source{buf: p}
	return s.info()
}

// GetInfoReader detects the image info of r, reading only the parts of the
// file it needs. An io.ReaderAt (such as *os.File) is read from offset 0; an
// io.ReadSeeker is read relative to its current position and left at an
// unspecified position; any other io.Reader is consumed forward only.
// The reader is not closed.
func GetInfoReader(r io.Reader) Info {
	var s source
	switch rr := r.(type) {
	case io.ReaderAt:
		s.r = rr
	case io.ReadSeeker:
		base, err := rr.Seek(0, io.SeekCurrent)
		if err != nil {
			s.r = &streamReaderAt{r: r}
		} else {
			s.r = &seekReaderAt{r: rr, base: base}
		}
	default:
		s.r = &streamReaderAt{r: r}
	}
	defer s.release()
	return s.info()
}

func (s *source) info() (info Info) {
	t := s.detect()
	switch t {
	case JPEG:
		jpeg(s, &info)
	case PNG:
		png(s, &info)
	case WEBP:
		webp(s, &info)
	case GIF:
		gif(s, &info)
	case BMP:
		bmp(s, &info)
	case PBM, PGM, PPM, XV:
		netpbm(s, &info)
	case XBM:
		xbm(s, &info)
	case XPM:
		xpm(s, &info)
	case TIFF:
		tiff(s, &info)
	case PSD:
		psd(s, &info)
	case MNG:
		mng(s, &info)
	case RGB:
		rgb(s, &info)
	case RAS:
		ras(s, &info)
	case PCX:
		pcx(s, &info)
	case AVIF, HEIC, HEIF:
		heif(s, &info)
	case JXL:
		jxl(s, &info)
	case ICO, CUR:
		ico(s, &info)
	case QOI:
		qoi(s, &info)
	}
	if info.Width == 0 || info.Height == 0 {
		return Info{}
	}
	info.Type = t
	return info
}

func (s *source) detect() Type {
	p := s.peek(0, 32)
	if len(p) < 4 {
		return Unknown
	}
	if len(p) >= 12 && string(p[4:8]) == "ftyp" {
		return ftypType(s)
	}

	switch p[0] {
	case '\xff':
		if p[1] == '\xd8' && p[2] == '\xff' {
			return JPEG
		}
		if p[1] == '\x0a' {
			return JXL
		}
	case '\x89':
		if hasPrefix(p, "\x89PNG\r\n\x1a\n") {
			return PNG
		}
	case 'R':
		if len(p) >= 12 && string(p[:4]) == "RIFF" && string(p[8:12]) == "WEBP" {
			return WEBP
		}
	case 'G':
		if len(p) >= 6 &&
			string(p[:4]) == "GIF8" &&
			(p[4] == '7' || p[4] == ',' || p[4] == '9') &&
			p[5] == 'a' {
			return GIF
		}
	case 'B':
		if p[1] == 'M' && len(p) >= 18 {
			switch littleEndian.Uint32(p[14:18]) {
			case 12, 16, 40, 52, 56, 64, 108, 124:
				return BMP
			}
		}
	case 'P':
		if isSpace(p[2]) {
			switch p[1] {
			case '1', '4':
				return PBM
			case '2', '5':
				return PGM
			case '3', '6':
				return PPM
			case '7':
				return XV
			}
		}
	case '#':
		if len(p) >= 8 && string(p[:7]) == "#define" && (p[7] == ' ' || p[7] == '\t') {
			return XBM
		}
	case '/':
		if hasPrefix(p, "/* XPM */") {
			return XPM
		}
	case 'M':
		if string(p[:4]) == "MM\x00\x2a" {
			return TIFF
		}
	case 'I':
		if string(p[:4]) == "II\x2a\x00" {
			return TIFF
		}
	case '8':
		if string(p[:4]) == "8BPS" {
			return PSD
		}
	case '\x8a':
		if hasPrefix(p, "\x8aMNG\r\n\x1a\n") {
			return MNG
		}
	case '\x01':
		if p[1] == '\xda' && p[2] <= 1 && (p[3] == 1 || p[3] == 2) {
			return RGB
		}
	case '\x59':
		if string(p[:4]) == "\x59\xa6\x6a\x95" {
			return RAS
		}
	case '\x0a':
		if p[2] == '\x01' && (p[1] <= 5 && p[1] != 1) {
			return PCX
		}
	case 'q':
		if string(p[:4]) == "qoif" {
			return QOI
		}
	case '\x00':
		if hasPrefix(p, "\x00\x00\x00\x0cJXL \r\n\x87\n") {
			return JXL
		}
		if len(p) >= 6 && p[1] == 0 && p[3] == 0 && (p[4] != 0 || p[5] != 0) {
			switch p[2] {
			case 1:
				return ICO
			case 2:
				return CUR
			}
		}
	}

	return Unknown
}

func png(s *source, info *Info) {
	b := s.at(12, 12)
	// Apple's CgBI chunk precedes IHDR in iOS-optimized PNGs.
	if b != nil && string(b[:4]) == "CgBI" {
		b = s.at(28, 12)
	}
	if b == nil || string(b[:4]) != "IHDR" {
		return
	}
	info.Width = bigEndian.Uint32(b[4:])
	info.Height = bigEndian.Uint32(b[8:])
}

func webp(s *source, info *Info) {
	// chunk header at 12, payload from 20
	b := s.at(12, 18)
	if b == nil {
		return
	}
	switch string(b[:4]) {
	case "VP8 ":
		if b[11] != 0x9d || b[12] != 0x01 || b[13] != 0x2a {
			return
		}
		info.Width = uint32(littleEndian.Uint16(b[14:])) & 0x3fff
		info.Height = uint32(littleEndian.Uint16(b[16:])) & 0x3fff
	case "VP8L":
		if b[8] != 0x2f {
			return
		}
		bits := littleEndian.Uint32(b[9:])
		info.Width = bits&0x3fff + 1
		info.Height = (bits>>14)&0x3fff + 1
	case "VP8X":
		info.Width = 1 + (uint32(b[12]) | uint32(b[13])<<8 | uint32(b[14])<<16)
		info.Height = 1 + (uint32(b[15]) | uint32(b[16])<<8 | uint32(b[17])<<16)
	}
}

func gif(s *source, info *Info) {
	b := s.at(6, 4)
	if b == nil {
		return
	}
	info.Width = uint32(littleEndian.Uint16(b))
	info.Height = uint32(littleEndian.Uint16(b[2:]))
}

func bmp(s *source, info *Info) {
	b := s.at(14, 12)
	if b == nil {
		return
	}
	if littleEndian.Uint32(b) == 12 {
		// BITMAPCOREHEADER: 16-bit dimensions
		info.Width = uint32(littleEndian.Uint16(b[4:]))
		info.Height = uint32(littleEndian.Uint16(b[6:]))
		return
	}
	w := int32(littleEndian.Uint32(b[4:]))
	h := int32(littleEndian.Uint32(b[8:]))
	if h < 0 {
		h = -h // top-down bitmap
	}
	if w > 0 && h > 0 {
		info.Width, info.Height = uint32(w), uint32(h)
	}
}

func psd(s *source, info *Info) {
	b := s.at(14, 8)
	if b == nil {
		return
	}
	info.Height = bigEndian.Uint32(b)
	info.Width = bigEndian.Uint32(b[4:])
}

func mng(s *source, info *Info) {
	b := s.at(12, 12)
	if b == nil || string(b[:4]) != "MHDR" {
		return
	}
	info.Width = bigEndian.Uint32(b[4:])
	info.Height = bigEndian.Uint32(b[8:])
}

func rgb(s *source, info *Info) {
	b := s.at(6, 4)
	if b == nil {
		return
	}
	info.Width = uint32(bigEndian.Uint16(b))
	info.Height = uint32(bigEndian.Uint16(b[2:]))
}

func ras(s *source, info *Info) {
	b := s.at(4, 8)
	if b == nil {
		return
	}
	info.Width = bigEndian.Uint32(b)
	info.Height = bigEndian.Uint32(b[4:])
}

func pcx(s *source, info *Info) {
	b := s.at(4, 8)
	if b == nil {
		return
	}
	xmin, ymin := littleEndian.Uint16(b), littleEndian.Uint16(b[2:])
	xmax, ymax := littleEndian.Uint16(b[4:]), littleEndian.Uint16(b[6:])
	if xmax < xmin || ymax < ymin {
		return
	}
	info.Width = uint32(xmax-xmin) + 1
	info.Height = uint32(ymax-ymin) + 1
}

func qoi(s *source, info *Info) {
	b := s.at(4, 8)
	if b == nil {
		return
	}
	info.Width = bigEndian.Uint32(b)
	info.Height = bigEndian.Uint32(b[4:])
}

// ico reports the largest image in an icon or cursor directory.
func ico(s *source, info *Info) {
	n, ok := s.le16(4)
	if !ok {
		return
	}
	var best uint64
	for i := int64(0); i < int64(n); i++ {
		e := s.at(6+16*i, 16)
		if e == nil {
			info.Width, info.Height = 0, 0
			return
		}
		w, h := uint32(e[0]), uint32(e[1])
		if w == 0 || h == 0 {
			// 0 means 256 or more; PNG-compressed entries carry the real size.
			if w, h, ok = icoPNGSize(s, int64(littleEndian.Uint32(e[12:])), w, h); !ok {
				info.Width, info.Height = 0, 0
				return
			}
		}
		if a := uint64(w) * uint64(h); a > best {
			best = a
			info.Width, info.Height = w, h
		}
	}
}

// icoPNGSize reads the size of a PNG entry; ok is false if its data is missing.
func icoPNGSize(s *source, off int64, w, h uint32) (uint32, uint32, bool) {
	b := s.at(off, 24)
	if b == nil {
		return 0, 0, false
	}
	if hasPrefix(b, "\x89PNG\r\n\x1a\n") && string(b[12:16]) == "IHDR" {
		return bigEndian.Uint32(b[16:]), bigEndian.Uint32(b[20:]), true
	}
	if w == 0 {
		w = 256
	}
	if h == 0 {
		h = 256
	}
	return w, h, true
}

func (s *source) le16(off int64) (uint16, bool) {
	b := s.at(off, 2)
	if b == nil {
		return 0, false
	}
	return littleEndian.Uint16(b), true
}

func hasPrefix(p []byte, prefix string) bool {
	return len(p) >= len(prefix) && string(p[:len(prefix)]) == prefix
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\v' || c == '\f'
}

type byteOrder interface {
	Uint16([]byte) uint16
	Uint32([]byte) uint32
}

var littleEndian littleOrder

type littleOrder struct{}

func (littleOrder) Uint16(b []byte) uint16 {
	_ = b[1]
	return uint16(b[0]) | uint16(b[1])<<8
}

func (littleOrder) Uint32(b []byte) uint32 {
	_ = b[3]
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

var bigEndian bigOrder

type bigOrder struct{}

func (bigOrder) Uint16(b []byte) uint16 {
	_ = b[1]
	return uint16(b[1]) | uint16(b[0])<<8
}

func (bigOrder) Uint32(b []byte) uint32 {
	_ = b[3]
	return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
}
