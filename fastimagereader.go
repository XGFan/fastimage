package fastimage

import (
	"bufio"
	"io"
)

// GetInfoReader detects a image info of data.
func GetInfoReader(file io.ReadSeekCloser) (info Info) {
	defer file.Close()
	p := make([]byte, 32)
	_, err := file.Read(p[:12])
	if err != nil {
		return
	}
	switch p[0] {
	case '\xff':
		if p[1] == '\xd8' {
			JpegReader(file, &info)
		}
	case '\x89':
		if p[1] == 'P' &&
			p[2] == 'N' &&
			p[3] == 'G' &&
			p[4] == '\x0d' &&
			p[5] == '\x0a' &&
			p[6] == '\x1a' &&
			p[7] == '\x0a' {
			file.Read(p[12:24])
			png(p, &info) //24
		}
	case 'R':
		if p[1] == 'I' &&
			p[2] == 'F' &&
			p[3] == 'F' &&
			p[8] == 'W' &&
			p[9] == 'E' &&
			p[10] == 'B' &&
			p[11] == 'P' {
			file.Read(p[12:30])
			webp(p, &info) //30
		}
	case 'G':
		if p[1] == 'I' &&
			p[2] == 'F' &&
			p[3] == '8' &&
			(p[4] == '7' || p[4] == ',' || p[4] == '9') &&
			p[5] == 'a' {
			gif(p, &info) //12
		}
	case 'B':
		if p[1] == 'M' {
			file.Read(p[12:26])
			bmp(p, &info) //26
		}
	case 'P':
		switch p[1] {
		case '1', '2', '3', '4', '5', '6', '7':
			PpmReader(file, &info)
		}
	case '#':
		if p[1] == 'd' &&
			p[2] == 'e' &&
			p[3] == 'f' &&
			p[4] == 'i' &&
			p[5] == 'n' &&
			p[6] == 'e' &&
			(p[7] == ' ' || p[7] == '\t') {
			XbmReader(file, &info)
		}
	case '/':
		if p[1] == '*' &&
			p[2] == ' ' &&
			p[3] == 'X' &&
			p[4] == 'P' &&
			p[5] == 'M' &&
			p[6] == ' ' &&
			p[7] == '*' &&
			p[8] == '/' {
			XpmReader(file, &info)
		}
	case 'M':
		if p[1] == 'M' && p[2] == '\x00' && p[3] == '\x2a' {
			tiff(p, &info, bigEndian) //TODO
		}
	case 'I':
		if p[1] == 'I' && p[2] == '\x2a' && p[3] == '\x00' {
			tiff(p, &info, littleEndian) //TODO
		}
	case '8':
		if p[1] == 'B' && p[2] == 'P' && p[3] == 'S' {
			file.Read(p[12:22])
			psd(p, &info) //22
		}
	case '\x8a':
		if p[1] == 'M' &&
			p[2] == 'N' &&
			p[3] == 'G' &&
			p[4] == '\x0d' &&
			p[5] == '\x0a' &&
			p[6] == '\x1a' &&
			p[7] == '\x0a' {
			file.Read(p[12:24])
			mng(p, &info) //24
		}
	case '\x01':
		if p[1] == '\xda' &&
			p[2] == '[' &&
			p[3] == '\x01' &&
			p[4] == '\x00' &&
			p[5] == ']' {
			rgb(p, &info) //10
		}
	case '\x59':
		if p[1] == '\xa6' && p[2] == '\x6a' && p[3] == '\x95' {
			ras(p, &info) //12
		}
	case '\x0a':
		if p[2] == '\x01' {
			pcx(p, &info) //12
		}
	}

	return
}

func JpegReader(file io.ReadSeeker, info *Info) {
	i := 2
	_, err := file.Seek(int64(i), 0)
	if err != nil {
		return
	}
	b := make([]byte, 9)
	for {
		n, err := file.Read(b)
		if err != nil || n != 9 {
			return
		}
		length := int(b[3]) | int(b[2])<<8
		code := b[1]
		marker := b[0]
		switch {
		case marker != 0xff:
			return
		case code >= 0xc0 && code <= 0xc3:
			info.Type = JPEG
			info.Width = uint32(b[8]) | uint32(b[7])<<8
			info.Height = uint32(b[6]) | uint32(b[5])<<8 //7,8,9,10
			return
		default:
			i += 2 + int(length)
			_, err = file.Seek(int64(i), 0)
			if err != nil {
				return
			}
		}
	}
}

func PpmReader(file io.ReadSeeker, info *Info) {
	_, _ = file.Seek(0, 0)
	reader := bufio.NewReader(file)
	_, err := reader.Discard(1)
	t, err := reader.ReadByte()
	if err != nil {
		return //FIXME
	}
	switch t {
	case '1':
		info.Type = PBM
	case '2', '5':
		info.Type = PGM
	case '3', '6':
		info.Type = PPM
	case '4':
		info.Type = BPM
	case '7':
		info.Type = XV
	}
	b := readerSkipSpace(reader)
	info.Width = parseUint32B(b, reader)
	b = readerSkipSpace(reader)
	info.Height = parseUint32B(b, reader)

	if info.Width == 0 || info.Height == 0 {
		info.Type = Unknown
	}
}

func XbmReader(file io.ReadSeeker, info *Info) {
	var p []byte
	_, _ = file.Seek(0, 0)
	reader := bufio.NewReader(file)
	readerReadNonSpace(reader)
	readerSkipSpace(reader)
	readerReadNonSpace(reader)
	b := readerSkipSpace(reader)
	info.Width = parseUint32B(b, reader)

	b = readerSkipSpace(reader)
	p = readerReadNonSpaceSlice(reader)
	if !(len(p) == 7 &&
		p[6] == 'e' &&
		p[0] == '#' &&
		p[1] == 'd' &&
		p[2] == 'e' &&
		p[3] == 'f' &&
		p[4] == 'i' &&
		p[5] == 'n') {
		return
	}
	b = readerSkipSpace(reader)
	readerReadNonSpace(reader)
	b = readerSkipSpace(reader)
	info.Height = parseUint32B(b, reader)

	if info.Width != 0 && info.Height != 0 {
		info.Type = XBM
	}
}

func XpmReader(file io.ReadSeeker, info *Info) {
	var line []byte
	var j int
	_, _ = file.Seek(0, 0)
	reader := bufio.NewReader(file)
	for {
		line = readerReadLine(reader)
		if len(line) == 0 {
			break
		}
		j = skipSpace(line, 0)
		if line[j] != '"' {
			continue
		}
		info.Width, j = parseUint32(line, j+1)
		j = skipSpace(line, j)
		info.Height, j = parseUint32(line, j)
		break
	}

	if info.Width != 0 && info.Height != 0 {
		info.Type = XPM
	}
}

func TiffReader(file io.ReadSeeker, info *Info, order byteOrder) {
	_, _ = file.Seek(0, 0)
	bytes := make([]byte, 8)
	_, _ = file.Read(bytes) //FIXME
	i := int(order.Uint32(bytes[4:8]))
	file.Seek(int64(i), 0)
	bytes = make([]byte, 6)
	_, _ = file.Read(bytes) //FIXME
	n := int(order.Uint16(bytes[2:4]))

	for i = 0; i < n; i++ {
		bytes = make([]byte, 12)
		_, _ = file.Read(bytes) //FIXME

		tag := order.Uint16(bytes[:2])
		datatype := order.Uint16(bytes[2:4])

		var value uint32
		switch datatype {
		case 1, 6:
			value = uint32(bytes[9])
		case 3, 8:
			value = uint32(order.Uint16(bytes[8:10]))
		case 4, 9:
			value = order.Uint32(bytes[8:12])
		default:
			return
		}

		switch tag {
		case 256:
			info.Width = value
		case 257:
			info.Height = value
		}

		if info.Width > 0 && info.Height > 0 {
			info.Type = TIFF
			return
		}
	}
}

/*
从当前游标开始跳过space,返回第一个不为space的值
如果读取的第一个不是space，那么返回第一个读取的值
*/
func readerSkipSpace(reader *bufio.Reader) byte {
	for {
		readByte, err := reader.ReadByte()
		if err != nil {
			return 0 //FIXME
		}
		if readByte != ' ' && readByte != '\t' && readByte != '\r' && readByte != '\n' {
			return readByte
		}
	}
}

/*
从当前游标开始读取，
读取到space，则停止，返回读取的值（即space中的一种
*/
func readerReadNonSpace(reader *bufio.Reader) byte {
	for {
		readByte, err := reader.ReadByte()
		if err != nil {
			return 0 //FIXME
		}
		if readByte == ' ' || readByte == '\t' || readByte == '\r' || readByte == '\n' {
			return readByte
		}
	}
}

/*
从当前游标开始读取，
读取到space
返回读取到的slice(不包括space
*/
func readerReadNonSpaceSlice(reader *bufio.Reader) []byte {
	bytes := make([]byte, 0, 16)
	for {
		readByte, err := reader.ReadByte()
		if err != nil {
			return nil //FIXME
		}
		if readByte == ' ' || readByte == '\t' || readByte == '\r' || readByte == '\n' {
			break
		} else {
			bytes = append(bytes, readByte)
		}
	}
	return bytes
}

/*
读取一行数据（包括最后的\n）
*/
func readerReadLine(reader *bufio.Reader) []byte {
	bytes := make([]byte, 0, 16)
	for {
		readByte, err := reader.ReadByte()
		if err != nil {
			break //FIXME
		}
		bytes = append(bytes, readByte)
		if readByte == '\n' {
			return bytes
		}
	}
	return bytes
}

func parseUint32B(b byte, bf *bufio.Reader) (n uint32) {
	x := uint32(b - '0')
	//goland:noinspection GoBoolExpressions
	for x >= 0 && x <= 9 {
		n = n*10 + x
		b, _ := bf.ReadByte()
		x = uint32(b - '0')
	}
	return
}
