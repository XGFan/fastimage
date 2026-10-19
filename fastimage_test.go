package fastimage

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"strings"
	"testing"
)

func TestTypeString(t *testing.T) {
	cases := []struct {
		Type   Type
		String string
	}{
		{BMP, "bmp"},
		{BPM, "bpm"},
		{GIF, "gif"},
		{JPEG, "jpeg"},
		{MNG, "mng"},
		{PBM, "pbm"},
		{PCX, "pcx"},
		{PGM, "pgm"},
		{PNG, "png"},
		{PPM, "ppm"},
		{PSD, "psd"},
		{RAS, "ras"},
		{RGB, "rgb"},
		{TIFF, "tiff"},
		{WEBP, "webp"},
		{XBM, "xbm"},
		{XPM, "xpm"},
		{XV, "xv"},
		{AVIF, "avif"},
		{HEIC, "heic"},
		{HEIF, "heif"},
		{JXL, "jxl"},
		{ICO, "ico"},
		{CUR, "cur"},
		{QOI, "qoi"},
		{Unknown, ""},
	}

	for _, c := range cases {
		if got, want := c.Type.String(), c.String; got != want {
			t.Errorf("get type string error, type=%+v, got=%+v, want=%+v,", c.Type, got, want)
		}
	}
}

func TestTypeMime(t *testing.T) {
	for typ := BMP; typ <= QOI; typ++ {
		if got := typ.Mime(); got == "" {
			t.Errorf("get type mime empty, type: %+v, string: %+v", typ, typ.String())
		}
	}
}

// Expected values come from `magick identify`, `heif-info` and `jxlinfo`.
var fixtures = []struct {
	File string
	Info Info
}{
	{"letter_T.jpg", Info{Type: JPEG, Width: 52, Height: 54}},
	{"progressive.jpg", Info{Type: JPEG, Width: 37, Height: 21}},
	{"orient6.jpg", Info{Type: JPEG, Width: 37, Height: 21, Orientation: 6}},
	{"orient5_le.jpg", Info{Type: JPEG, Width: 37, Height: 21, Orientation: 5}},
	{"4.sm.webp", Info{Type: WEBP, Width: 320, Height: 241}},
	{"2_webp_a.webp", Info{Type: WEBP, Width: 386, Height: 395}},
	{"2_webp_ll.webp", Info{Type: WEBP, Width: 386, Height: 395}},
	{"4_webp_ll.webp", Info{Type: WEBP, Width: 421, Height: 163}},
	{"tall_ll.webp", Info{Type: WEBP, Width: 7, Height: 1500}},
	{"pass-1_s.png", Info{Type: PNG, Width: 90, Height: 60}},
	{"pak38.gif", Info{Type: GIF, Width: 333, Height: 194}},
	{"tiny.gif", Info{Type: GIF, Width: 1, Height: 1}},
	{"xterm.bmp", Info{Type: BMP, Width: 64, Height: 38}},
	{"core.bmp", Info{Type: BMP, Width: 37, Height: 21}},
	{"letter_N.ppm", Info{Type: PPM, Width: 66, Height: 57}},
	{"pbm_p1.pbm", Info{Type: PBM, Width: 37, Height: 21}},
	{"pbm_p4.pbm", Info{Type: PBM, Width: 37, Height: 21}},
	{"pgm_p5.pgm", Info{Type: PGM, Width: 37, Height: 21}},
	{"pam_p7.pam", Info{Type: XV, Width: 37, Height: 21}},
	{"spacer50.xbm", Info{Type: XBM, Width: 50, Height: 10}},
	{"xterm.xpm", Info{Type: XPM, Width: 64, Height: 38}},
	{"bexjdic.tif", Info{Type: TIFF, Width: 35, Height: 32, Orientation: 1}},
	{"lexjdic.tif", Info{Type: TIFF, Width: 35, Height: 32, Orientation: 1}},
	{"orient8.tif", Info{Type: TIFF, Width: 37, Height: 21, Orientation: 8}},
	{"letter_T.psd", Info{Type: PSD, Width: 52, Height: 54}},
	{"468x60.psd", Info{Type: PSD, Width: 468, Height: 60}},
	{"letter_T.mng", Info{Type: MNG, Width: 52, Height: 54}},
	{"letter_T.ras", Info{Type: RAS, Width: 52, Height: 54}},
	{"letter_T.pcx", Info{Type: PCX, Width: 52, Height: 54}},
	{"plain.heic", Info{Type: HEIC, Width: 40, Height: 24}},
	{"rot90.heic", Info{Type: HEIC, Width: 40, Height: 24, Orientation: 6}},
	// heif-enc converts the source JPEG's EXIF orientation N into irot/imir
	{"orient2.heic", Info{Type: HEIC, Width: 40, Height: 24, Orientation: 2}},
	{"orient4.heic", Info{Type: HEIC, Width: 40, Height: 24, Orientation: 4}},
	{"orient5.heic", Info{Type: HEIC, Width: 40, Height: 24, Orientation: 5}},
	{"orient7.heic", Info{Type: HEIC, Width: 40, Height: 24, Orientation: 7}},
	{"plain.avif", Info{Type: AVIF, Width: 40, Height: 24}},
	{"rot180.avif", Info{Type: AVIF, Width: 40, Height: 24, Orientation: 3}},
	{"ratio.jxl", Info{Type: JXL, Width: 160, Height: 120}},
	{"big.jxl", Info{Type: JXL, Width: 1001, Height: 777}},
	{"container.jxl", Info{Type: JXL, Width: 40, Height: 24}},
	{"orient6.jxl", Info{Type: JXL, Width: 37, Height: 21, Orientation: 6}},
	{"multi.ico", Info{Type: ICO, Width: 48, Height: 48}},
	{"cursor.cur", Info{Type: CUR, Width: 24, Height: 40}},
	{"icon.qoi", Info{Type: QOI, Width: 37, Height: 21}},
}

func readFixture(t testing.TB, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// seekOnly hides io.ReaderAt so GetInfoReader takes the io.ReadSeeker path.
type seekOnly struct{ io.ReadSeeker }

// streamOnly hides everything but io.Reader.
type streamOnly struct{ io.Reader }

func TestFixtures(t *testing.T) {
	for _, c := range fixtures {
		t.Run(c.File, func(t *testing.T) {
			data := readFixture(t, c.File)
			if got := GetType(data); got != c.Info.Type {
				t.Errorf("GetType = %v, want %v", got, c.Info.Type)
			}
			if got := GetInfo(data); got != c.Info {
				t.Errorf("GetInfo = %+v, want %+v", got, c.Info)
			}
			if got := GetInfoReader(bytes.NewReader(data)); got != c.Info {
				t.Errorf("GetInfoReader(ReaderAt) = %+v, want %+v", got, c.Info)
			}
			if got := GetInfoReader(seekOnly{bytes.NewReader(data)}); got != c.Info {
				t.Errorf("GetInfoReader(ReadSeeker) = %+v, want %+v", got, c.Info)
			}
			if got := GetInfoReader(streamOnly{bytes.NewReader(data)}); got != c.Info {
				t.Errorf("GetInfoReader(Reader) = %+v, want %+v", got, c.Info)
			}
			f, err := os.Open("testdata/" + c.File)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			if got := GetInfoReader(f); got != c.Info {
				t.Errorf("GetInfoReader(*os.File) = %+v, want %+v", got, c.Info)
			}
		})
	}
}

// A file cut off anywhere must report either nothing or the full answer.
func TestTruncated(t *testing.T) {
	for _, c := range fixtures {
		data := readFixture(t, c.File)
		for n := 0; n < len(data) && n <= 2*windowSize; n++ {
			got := GetInfo(data[:n])
			if got != (Info{}) && got != c.Info {
				t.Fatalf("%s[:%d] = %+v, want empty or %+v", c.File, n, got, c.Info)
			}
			if r := GetInfoReader(streamOnly{bytes.NewReader(data[:n])}); r != got {
				t.Fatalf("%s[:%d]: reader %+v != bytes %+v", c.File, n, r, got)
			}
		}
	}
}

func TestDisplaySize(t *testing.T) {
	for o := uint8(0); o <= 9; o++ {
		w, h := Info{Width: 4, Height: 3, Orientation: o}.DisplaySize()
		swapped := o >= 5 && o <= 8
		if swapped && (w != 3 || h != 4) || !swapped && (w != 4 || h != 3) {
			t.Errorf("orientation %d: DisplaySize = %dx%d", o, w, h)
		}
	}
}

func TestShortInputs(t *testing.T) {
	inputs := []string{
		"", "\xff", "\xff\xd8", "\xff\xd8\xff", "\xff\x0a", "BM", "P6", "P6\n", "P6\n12", "P7\nWIDTH 3\n",
		"#define", "#define a_width 3\n", "/* XPM */", "/* XPM */ /* \"1 1", "/* XPM */\n\"",
		"II*\x00", "MM\x00*\x00\x00\x00\x08", "\x00\x00\x00\x0cftyp", "\x00\x00\x01\x00\x01\x00",
		"RIFF\x00\x00\x00\x00WEBPVP8L", "qoif", "\x00\x00\x00\x0cJXL \r\n\x87\n",
	}
	for _, in := range inputs {
		if got := GetInfo([]byte(in)); got != (Info{}) {
			t.Errorf("GetInfo(%q) = %+v, want empty", in, got)
		}
	}
}

func jpegSegment(marker byte, payload []byte) []byte {
	b := []byte{0xff, marker, 0, 0}
	binary.BigEndian.PutUint16(b[2:], uint16(len(payload)+2))
	return append(b, payload...)
}

func sof(marker byte, w, h uint16) []byte {
	p := []byte{8, 0, 0, 0, 0, 1, 1, 0x11, 0}
	binary.BigEndian.PutUint16(p[1:], h)
	binary.BigEndian.PutUint16(p[3:], w)
	return jpegSegment(marker, p)
}

func TestJPEGMarkers(t *testing.T) {
	big := make([]byte, 60000) // far larger than one reader window
	cases := []struct {
		name string
		data []byte
		want Info
	}{
		{"fill bytes and standalone markers", concat([]byte{0xff, 0xd8, 0xff, 0xff, 0xff, 0xd0, 0xff, 0x01}, sof(0xc0, 640, 480)), Info{Type: JPEG, Width: 640, Height: 480}},
		{"SOF after large APP segment", concat([]byte{0xff, 0xd8}, jpegSegment(0xe2, big), sof(0xc2, 4000, 3000)), Info{Type: JPEG, Width: 4000, Height: 3000}},
		{"arithmetic SOF9", concat([]byte{0xff, 0xd8}, sof(0xc9, 7, 9)), Info{Type: JPEG, Width: 7, Height: 9}},
		{"DHT is not a frame header", concat([]byte{0xff, 0xd8}, jpegSegment(0xc4, []byte{1, 2, 3, 4, 5, 6, 7}), sof(0xc1, 7, 9)), Info{Type: JPEG, Width: 7, Height: 9}},
		{"SOS before SOF", concat([]byte{0xff, 0xd8}, jpegSegment(0xda, []byte{1}), sof(0xc0, 7, 9)), Info{}},
		{"invalid segment length", []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x01, 0, 0, 0, 0}, Info{}},
		{"junk bytes between segments", concat([]byte{0xff, 0xd8}, jpegSegment(0xe0, []byte("JFIF\x00")), []byte{0, 0, 0xff, 0x00}, sof(0xc0, 7, 9)), Info{Type: JPEG, Width: 7, Height: 9}},
		{"XMP APP1 is not Exif", concat([]byte{0xff, 0xd8}, jpegSegment(0xe1, []byte("http://ns.adobe.com/xap/1.0/\x00")), sof(0xc0, 7, 9)), Info{Type: JPEG, Width: 7, Height: 9}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GetInfo(c.data); got != c.want {
				t.Errorf("GetInfo = %+v, want %+v", got, c.want)
			}
			if got := GetInfoReader(streamOnly{bytes.NewReader(c.data)}); got != c.want {
				t.Errorf("GetInfoReader(Reader) = %+v, want %+v", got, c.want)
			}
		})
	}
}

func concat(parts ...[]byte) []byte {
	return bytes.Join(parts, nil)
}

func TestBMPTopDown(t *testing.T) {
	data := readFixture(t, "xterm.bmp")
	data = append([]byte(nil), data...)
	h := int32(binary.LittleEndian.Uint32(data[22:]))
	binary.LittleEndian.PutUint32(data[22:], uint32(-h))
	if got, want := GetInfo(data), (Info{Type: BMP, Width: 64, Height: 38}); got != want {
		t.Errorf("GetInfo = %+v, want %+v", got, want)
	}
}

func TestPCXInvalidWindow(t *testing.T) {
	data := append([]byte(nil), readFixture(t, "letter_T.pcx")...)
	binary.LittleEndian.PutUint16(data[4:], 100) // xmin > xmax
	if got := GetInfo(data); got != (Info{}) {
		t.Errorf("GetInfo = %+v, want empty", got)
	}
}

func TestNetpbmHeaders(t *testing.T) {
	cases := []struct {
		in   string
		want Info
	}{
		{"P6\n# made by hand\n12 # width\n34\n255\n", Info{Type: PPM, Width: 12, Height: 34}},
		{"P5 7#c\n9 255 ", Info{Type: PGM, Width: 7, Height: 9}},
		{"P7 332\n#XVVERSION:Version 2.28\n#IMGINFO:48x40 RGB\n#END_OF_COMMENTS\n48 40 255\n", Info{Type: XV, Width: 48, Height: 40}},
		{"P7\nHEIGHT 5\nWIDTH 6\nDEPTH 3\nENDHDR\n", Info{Type: XV, Width: 6, Height: 5}},
		{"P6\n12 99999999999\n255\n", Info{}},
		{"P6\n-1 2\n255\n", Info{}},
		{"P6x 1 2\n", Info{}},
	}
	for _, c := range cases {
		if got := GetInfo([]byte(c.in)); got != c.want {
			t.Errorf("GetInfo(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestSGIDetection(t *testing.T) {
	// magic, storage (verbatim/RLE), bytes per channel, dimension, xsize, ysize, zsize
	for _, storage := range []byte{0, 1} {
		data := []byte{0x01, 0xda, storage, 1, 0, 3, 0, 20, 0, 10, 0, 3}
		if got, want := GetInfo(data), (Info{Type: RGB, Width: 20, Height: 10}); got != want {
			t.Errorf("storage %d: GetInfo = %+v, want %+v", storage, got, want)
		}
	}
}

func TestPNGCgBI(t *testing.T) {
	data := concat([]byte("\x89PNG\r\n\x1a\n"),
		[]byte{0, 0, 0, 4}, []byte("CgBI"), []byte{0x50, 0, 0x20, 0x06}, []byte{1, 2, 3, 4},
		[]byte{0, 0, 0, 13}, []byte("IHDR"), []byte{0, 0, 0, 57, 0, 0, 0, 114, 8, 6, 0, 0, 0})
	if got, want := GetInfo(data), (Info{Type: PNG, Width: 57, Height: 114}); got != want {
		t.Errorf("GetInfo = %+v, want %+v", got, want)
	}
}

func TestICOPNGEntry(t *testing.T) {
	png := concat([]byte("\x89PNG\r\n\x1a\n"), []byte{0, 0, 0, 13}, []byte("IHDR"), []byte{0, 0, 2, 0, 0, 0, 2, 0, 8, 6, 0, 0, 0})
	dir := []byte{0, 0, 1, 0, 2, 0,
		16, 16, 0, 0, 1, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 1, 0, 32, 0, 0, 0, 0, 0, 38, 0, 0, 0}
	data := concat(dir, png)
	if got, want := GetInfo(data), (Info{Type: ICO, Width: 512, Height: 512}); got != want {
		t.Errorf("GetInfo = %+v, want %+v", got, want)
	}
	// PNG data cut off: no partial answer
	for _, n := range []int{len(dir), len(dir) + 12} {
		if got := GetInfo(data[:n]); got != (Info{}) {
			t.Errorf("GetInfo(data[:%d]) = %+v, want empty", n, got)
		}
	}
}

func isoBox(typ string, payload ...[]byte) []byte {
	body := concat(payload...)
	b := make([]byte, 8, 8+len(body))
	binary.BigEndian.PutUint32(b, uint32(8+len(body)))
	copy(b[4:], typ)
	return append(b, body...)
}

func u32(v uint32) []byte { return binary.BigEndian.AppendUint32(nil, v) }

// heifFile builds a minimal HEIF whose primary item 1 has the given properties
// associated in order; item 2 gets a decoy ispe.
func heifFile(brand string, props ...[]byte) []byte {
	assoc := []byte{0, 1, byte(len(props))}
	for i := range props {
		assoc = append(assoc, byte(i+1))
	}
	decoy := isoBox("ispe", u32(0), u32(9), u32(9))
	assoc = append(assoc, 0, 2, 1, byte(len(props)+1))
	ipco := isoBox("ipco", append(props, decoy)...)
	ipma := isoBox("ipma", u32(0), u32(2), assoc)
	meta := isoBox("meta", u32(0),
		isoBox("hdlr", u32(0), u32(0), []byte("pict"), make([]byte, 13)),
		isoBox("pitm", u32(0), []byte{0, 1}),
		isoBox("iprp", ipco, ipma))
	return concat(isoBox("ftyp", []byte(brand), u32(0), []byte("mif1"), []byte(brand)), meta)
}

func TestHEIFProperties(t *testing.T) {
	ispe := isoBox("ispe", u32(0), u32(400), u32(300))
	irot := func(ccw byte) []byte { return isoBox("irot", []byte{ccw}) }
	imir := func(axis byte) []byte { return isoBox("imir", []byte{axis}) }
	clap := isoBox("clap", u32(399), u32(2), u32(300), u32(1), u32(0), u32(1), u32(0), u32(1))
	cases := []struct {
		name  string
		props [][]byte
		want  Info
	}{
		{"ispe only", [][]byte{ispe}, Info{Type: HEIC, Width: 400, Height: 300}},
		{"irot 90 ccw", [][]byte{ispe, irot(1)}, Info{Type: HEIC, Width: 400, Height: 300, Orientation: 8}},
		{"irot 270 ccw", [][]byte{ispe, irot(3)}, Info{Type: HEIC, Width: 400, Height: 300, Orientation: 6}},
		{"imir 0 exchanges top and bottom", [][]byte{ispe, imir(0)}, Info{Type: HEIC, Width: 400, Height: 300, Orientation: 4}},
		{"imir 1 exchanges left and right", [][]byte{ispe, imir(1)}, Info{Type: HEIC, Width: 400, Height: 300, Orientation: 2}},
		{"irot 90 ccw then left-right mirror", [][]byte{ispe, irot(1), imir(1)}, Info{Type: HEIC, Width: 400, Height: 300, Orientation: 7}},
		{"irot 270 ccw then left-right mirror", [][]byte{ispe, irot(3), imir(1)}, Info{Type: HEIC, Width: 400, Height: 300, Orientation: 5}},
		{"clap rounds", [][]byte{ispe, clap}, Info{Type: HEIC, Width: 200, Height: 300}},
		{"no ispe", [][]byte{irot(1)}, Info{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := heifFile("heic", c.props...)
			if got := GetInfo(data); got != c.want {
				t.Errorf("GetInfo = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestFtypBrands(t *testing.T) {
	cases := []struct {
		brands []string
		want   Type
	}{
		{[]string{"avif", "mif1"}, AVIF},
		{[]string{"mif1", "heic"}, HEIC},
		{[]string{"mif1", "avif"}, AVIF},
		{[]string{"mif1", "miaf"}, HEIF},
		{[]string{"isom", "mp41"}, Unknown},
		{[]string{"qt  "}, Unknown},
	}
	for _, c := range cases {
		ftyp := isoBox("ftyp", []byte(c.brands[0]), u32(0), []byte(strings.Join(c.brands[1:], "")))
		if got := GetType(ftyp); got != c.want {
			t.Errorf("brands %v: GetType = %v, want %v", c.brands, got, c.want)
		}
	}
}

func TestReadSeekerKeepsBase(t *testing.T) {
	data := readFixture(t, "orient6.jpg")
	r := bytes.NewReader(append([]byte("junk"), data...))
	if _, err := r.Seek(4, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if got, want := GetInfoReader(seekOnly{r}), (Info{Type: JPEG, Width: 37, Height: 21, Orientation: 6}); got != want {
		t.Errorf("GetInfoReader = %+v, want %+v", got, want)
	}
}

// countingReaderAt counts reads to catch offsets that make the parser jump around.
type countingReaderAt struct {
	*bytes.Reader
	reads int
}

func (c *countingReaderAt) ReadAt(p []byte, off int64) (int, error) {
	c.reads++
	return c.Reader.ReadAt(p, off)
}

func TestExifOffsetsStayInSegment(t *testing.T) {
	// Many Exif APP1 segments whose IFD offset points far beyond the segment.
	seg := jpegSegment(0xe1, concat([]byte("Exif\x00\x00II*\x00"), []byte{0, 0, 0x10, 0}))
	data := concat([]byte{0xff, 0xd8}, bytes.Repeat(seg, 50000), sof(0xc0, 7, 9))
	r := &countingReaderAt{Reader: bytes.NewReader(data)}
	GetInfoReader(r)
	if max := len(data)/windowSize + 2; r.reads > max {
		t.Errorf("%d reads for a %d byte file, want at most %d", r.reads, len(data), max)
	}
}

func TestReadBudget(t *testing.T) {
	// SOF behind more large APP segments than the read budget allows.
	data := concat([]byte{0xff, 0xd8}, bytes.Repeat(jpegSegment(0xe2, make([]byte, windowSize)), maxFills+10), sof(0xc0, 7, 9))
	r := &countingReaderAt{Reader: bytes.NewReader(data)}
	if got := GetInfoReader(r); got != (Info{}) {
		t.Errorf("GetInfoReader = %+v, want empty", got)
	}
	if r.reads > maxFills {
		t.Errorf("%d reads, want at most %d", r.reads, maxFills)
	}
}

// infiniteZeros is an endless stream, like a hostile network body.
type infiniteZeros struct{}

func (infiniteZeros) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func TestStreamSkipIsBounded(t *testing.T) {
	// Exif IFD offset of ~4 GB on a forward-only reader.
	tiff := concat([]byte("Exif\x00\x00II*\x00"), []byte{0xf0, 0xff, 0xff, 0xff})
	head := concat([]byte{0xff, 0xd8}, jpegSegment(0xe1, tiff))
	r := &countingReader{r: io.MultiReader(bytes.NewReader(head), infiniteZeros{})}
	GetInfoReader(r)
	if r.n > maxStream+windowSize {
		t.Errorf("read %d bytes, want at most %d", r.n, maxStream+windowSize)
	}
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

func TestBoxHeaderAcrossWindowEdge(t *testing.T) {
	// A largesize meta box whose 16-byte header straddles the first reader window.
	ftyp := isoBox("ftyp", []byte("heic"), u32(0), []byte("mif1heic"))
	meta := heifFile("heic", isoBox("ispe", u32(0), u32(400), u32(300)))[len(ftyp):]
	for pad := windowSize - len(ftyp) - 20; pad < windowSize-len(ftyp); pad++ {
		large := concat(u32(1), meta[4:8], []byte{0, 0, 0, 0}, u32(uint32(len(meta)+8)), meta[8:])
		data := concat(ftyp, isoBox("free", make([]byte, pad-8)), large, isoBox("mdat", make([]byte, 10000)))
		want := Info{Type: HEIC, Width: 400, Height: 300}
		if got := GetInfo(data); got != want {
			t.Fatalf("pad %d: GetInfo = %+v, want %+v", pad, got, want)
		}
		if got := GetInfoReader(bytes.NewReader(data)); got != want {
			t.Fatalf("pad %d: GetInfoReader = %+v, want %+v", pad, got, want)
		}
	}
}

func TestPitmV0AtEnd(t *testing.T) {
	ispe := isoBox("ispe", u32(0), u32(400), u32(300))
	meta := isoBox("meta", u32(0),
		isoBox("iprp", isoBox("ipco", ispe), isoBox("ipma", u32(0), u32(1), []byte{0, 1, 1, 1})),
		isoBox("pitm", u32(0), []byte{0, 1}))
	data := concat(isoBox("ftyp", []byte("heic"), u32(0), []byte("mif1")), meta)
	if got, want := GetInfo(data), (Info{Type: HEIC, Width: 400, Height: 300}); got != want {
		t.Errorf("GetInfo = %+v, want %+v", got, want)
	}
}

func FuzzGetInfo(f *testing.F) {
	for _, c := range fixtures {
		data := readFixture(f, c.File)
		f.Add(data[:min(len(data), 1024)])
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		info := GetInfo(data)
		// Equal for inputs this small; huge inputs may exceed the reader's maxFills budget.
		if r := GetInfoReader(bytes.NewReader(data)); r != info {
			t.Fatalf("ReaderAt %+v != bytes %+v", r, info)
		}
		// A forward-only reader may give up where random access succeeds, but never disagree.
		if r := GetInfoReader(streamOnly{bytes.NewReader(data)}); r != info && r != (Info{}) {
			t.Fatalf("Reader %+v != bytes %+v", r, info)
		}
		if info.Type != Unknown && GetType(data) != info.Type {
			t.Fatalf("GetType %v != GetInfo type %v", GetType(data), info.Type)
		}
	})
}
