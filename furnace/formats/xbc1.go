package formats

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/3096/furnace/furnace"
)

var XBC1_MAGIC = [4]byte{'x', 'b', 'c', '1'}

type XBC1Header struct {
	Magic            [4]byte
	NumFiles         uint32
	UncompressedSize uint32
	CompressedSize   uint32
	Hash             uint32
	Name             [0x1C]byte
}

type XBC1 []byte

const XBC1_ZLIB_COMPRESSION_LEVEL = zlib.BestCompression

func ReadXBC1Header(reader io.Reader) (XBC1Header, error) {
	var header XBC1Header
	if err := binary.Read(reader, furnace.TargetByteOrder, &header); err != nil {
		return XBC1Header{}, errors.New("Error reading xbc1 header: " + err.Error())
	}
	if header.Magic != XBC1_MAGIC {
		return XBC1Header{}, errors.New("Invalid xbc1 header")
	}
	return header, nil
}

func ExtractXBC1(reader io.Reader) (XBC1Header, []byte, error) {
	header, err := ReadXBC1Header(reader)
	if err != nil {
		return XBC1Header{}, nil, err
	}
	if header.NumFiles != 1 {
		return XBC1Header{}, nil, errors.New("Unexpected number of files in xbc1: " + fmt.Sprint(header.NumFiles))
	}

	uncompressedDataBuffer := bytes.NewBuffer(make([]byte, 0, header.UncompressedSize))
	zlibReader, err := zlib.NewReader(reader)
	defer zlibReader.Close()
	if err != nil {
		return XBC1Header{}, nil, errors.New("Error creating zlib reader: " + err.Error())
	}
	if _, err := io.Copy(uncompressedDataBuffer, zlibReader); err != nil {
		return XBC1Header{}, nil, errors.New("Error reading zlib data: " + err.Error())
	}

	return header, uncompressedDataBuffer.Bytes(), nil
}

func CompressToXBC1(name [0x1C]byte, data []byte) (XBC1, error) {
	header := XBC1Header{
		Magic:            XBC1_MAGIC,
		NumFiles:         1,
		UncompressedSize: uint32(len(data)),
		Name:             name,
	}

	compressedDataBuffer := bytes.Buffer{}
	zlibWriter, err := zlib.NewWriterLevel(&compressedDataBuffer, XBC1_ZLIB_COMPRESSION_LEVEL)
	if err != nil {
		return nil, errors.New("Error creating zlib writer: " + err.Error())
	}
	if _, err := zlibWriter.Write(data); err != nil {
		return nil, errors.New("Error writing zlib data: " + err.Error())
	}
	if err := zlibWriter.Close(); err != nil {
		return nil, errors.New("Error closing zlib writer: " + err.Error())
	}

	xbc1Buffer := bytes.Buffer{}
	header.CompressedSize = uint32(compressedDataBuffer.Len())
	if err := binary.Write(&xbc1Buffer, furnace.TargetByteOrder, &header); err != nil {
		return nil, errors.New("Error writing xbc1 header: " + err.Error())
	}
	if _, err := xbc1Buffer.Write(compressedDataBuffer.Bytes()); err != nil {
		return nil, errors.New("Error writing xbc1 data: " + err.Error())
	}

	return xbc1Buffer.Bytes(), nil
}
