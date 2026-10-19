package fastimage

import (
	"image"
	_ "image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkGetInfo(b *testing.B) {
	for _, name := range []string{"letter_T.jpg", "orient6.jpg", "pass-1_s.png", "4.sm.webp", "orient8.tif", "rot90.heic", "big.jxl", "xterm.xpm"} {
		data := readFixture(b, name)
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				GetInfo(data)
			}
		})
	}
}

// largeExifJPEG writes a JPEG whose frame header sits behind 60 KB of APP data,
// like a camera photo with an embedded thumbnail.
func largeExifJPEG(b *testing.B) string {
	data := readFixture(b, "orient6.jpg")
	data = concat(data[:2], jpegSegment(0xe2, make([]byte, 60000)), data[2:])
	name := filepath.Join(b.TempDir(), "large.jpg")
	if err := os.WriteFile(name, data, 0o644); err != nil {
		b.Fatal(err)
	}
	return name
}

func BenchmarkGetInfoReaderFile(b *testing.B) {
	name := largeExifJPEG(b)
	f, err := os.Open(name)
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if GetInfoReader(f).Width == 0 {
			b.Fatal("no size")
		}
	}
}

// BenchmarkDecodeConfigFile is the standard library baseline for the same file.
func BenchmarkDecodeConfigFile(b *testing.B) {
	name := largeExifJPEG(b)
	f, err := os.Open(name)
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := f.Seek(0, 0); err != nil {
			b.Fatal(err)
		}
		if _, _, err := image.DecodeConfig(f); err != nil {
			b.Fatal(err)
		}
	}
}
