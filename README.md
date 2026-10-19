# fastimage - fast image info for go

Fork of [phuslu/fastimage](https://github.com/phuslu/fastimage): reads an image's type and dimensions from its header without decoding pixels.

## Features

* Header only - `GetInfoReader` reads just the bytes it needs (one 4 KB window for most formats, skipping over large JPEG metadata)
* Safe on untrusted input - every read is bounds-checked; truncated or malformed files report `Unknown` instead of panicking or guessing
* Orientation aware - EXIF orientation of JPEG/TIFF, `irot`/`imir` of HEIF/AVIF and the JPEG XL header
* Widely Format - avif/bmp/cur/gif/heic/heif/ico/jpeg/jxl/mng/pbm/pcx/pgm/png/ppm/psd/qoi/ras/rgb/tiff/webp/xbm/xpm/xv(pam)

### Getting Started

```go
package main

import (
	"fmt"
	"os"

	"github.com/XGFan/fastimage"
)

func main() {
	f, err := os.Open("photo.jpg")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	info := fastimage.GetInfoReader(f)
	w, h := info.DisplaySize()
	fmt.Printf("%s %dx%d orientation=%d display=%dx%d\n", info.Type, info.Width, info.Height, info.Orientation, w, h)
}
```

### API

| Function | Description |
| --- | --- |
| `GetType(p []byte) Type` | Type from the file signature only. |
| `GetInfo(p []byte) Info` | Type and size from bytes in memory; `p` must contain the header (for JPEG/TIFF/HEIF that can be well past the first KB). |
| `GetInfoReader(r io.Reader) Info` | Same from a reader, reading only what is needed. `io.ReaderAt` (e.g. `*os.File`) is read from offset 0, `io.ReadSeeker` relative to its current position, any other `io.Reader` forward only. The reader is not closed. |

Reader mode stops after about 4 MB of reads (1024 windows) and a forward-only reader is not followed past 64 MB, so a crafted file cannot cause unbounded I/O; a file whose size lies beyond that reports `Unknown`.

`Info.Width`/`Info.Height` are the stored pixel dimensions. `Info.Orientation` is the EXIF orientation (1-8, 0 when the file has none); `Info.DisplaySize()` returns the size after applying it (swapped for 5-8). ICO/CUR report their largest image.

### Command Tool
```bash
$ go install github.com/XGFan/fastimage/cmd/fastimage@latest
$ fastimage banner.png
png image/png 320 50
```

### Testing

```bash
go test .
go test -run '^$' -fuzz FuzzGetInfo -fuzztime 60s .
go test -run '^$' -bench . -benchmem .
```

Fixtures in `testdata/` were generated with ImageMagick, libheif `heif-enc` and libjxl `cjxl`; expected values in the tests come from `magick identify`, `heif-info` and `jxlinfo`.
