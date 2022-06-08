package furnace

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var MAGIC = [4]byte{'x', 'b', 'c', '1'}

type Header struct {
	Magic            [4]byte
	NumFiles         uint32
	UncompressedSize uint32
	CompressedSize   uint32
	Hash             uint32
	Name             [0x1C]byte
}

func ExtractXBC1(reader io.Reader) ([]byte, error) {
	var header Header
	if err := binary.Read(reader, binary.LittleEndian, &header); err != nil {
		return nil, errors.New("Error reading xbc1 header: " + err.Error())
	}
	if header.Magic != MAGIC {
		return nil, errors.New("Invalid xbc1 header")
	}
	if header.NumFiles != 1 {
		return nil, errors.New("Unexpected number of files in xbc1: " + fmt.Sprint(header.NumFiles))
	}

	// uncompress data
	uncompressedDataBuffer := bytes.NewBuffer(make([]byte, 0, header.UncompressedSize))
	zlibReader, err := zlib.NewReader(reader)
	if err != nil {
		return nil, errors.New("Error creating zlib reader: " + err.Error())
	}
	if _, err := io.Copy(uncompressedDataBuffer, zlibReader); err != nil {
		return nil, errors.New("Error reading zlib data: " + err.Error())
	}
	if err := zlibReader.Close(); err != nil {
		return nil, errors.New("Error closing zlib reader: " + err.Error())
	}
	return uncompressedDataBuffer.Bytes(), nil
}
