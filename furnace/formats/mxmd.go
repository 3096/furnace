package formats

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"github.com/3096/furnace/furnace"
)

const MXMD_MAGIC uint32 = 'M'<<24 | 'X'<<16 | 'M'<<8 | 'D'

type MXMDHeader struct {
	Magic                  uint32
	Version                uint32
	ModelsOffset           uint32
	MaterialsOffset        uint32
	Unk0                   uint32
	VertexBufferOffset     uint32
	ShadersOffset          uint32
	CachedTexturesOffset   uint32
	Unk1                   uint32
	UncachedTexturesOffset uint32
}

type MXMD []byte

func (mxmd *MXMD) GetHeader() (*MXMDHeader, error) {
	var header MXMDHeader
	err := binary.Read(bytes.NewReader(*mxmd), furnace.TargetByteOrder, &header)
	if err != nil {
		return nil, err
	}
	return &header, nil
}

func ReadMXMD(reader io.Reader) (MXMD, error) {
	mxmd := bytes.NewBuffer(make(MXMD, 0))
	if _, err := io.Copy(mxmd, reader); err != nil {
		return nil, err
	}
	return mxmd.Bytes(), nil
}

func WriteMXMD(writer io.Writer, mxmd MXMD) error {
	if err := binary.Write(writer, furnace.TargetByteOrder, mxmd); err != nil {
		return errors.New("Error writing mxmd: " + err.Error())
	}
	return nil
}
