package furnace

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/3096/furnace/furnace"
)

const MSRD_MAGIC uint32 = 'M'<<24 | 'S'<<16 | 'R'<<8 | 'D'

type MSRDHeader struct {
	Magic          uint32
	Version        uint32
	MetaDataSize   uint32
	MetaDataOffset uint32
}

type MSRDMetaDataHeader struct {
	Tag      uint32
	Revision uint32

	DataItemsCount       uint32
	DataItemsTableOffset uint32
	FileCount            uint32
	FileTableOffset      uint32

	Unk1 [0x1C]byte

	TextureIdsCount    uint32
	TextureIdsOffset   uint32
	TextureCountOffset uint32
}

type MSRDDataItem struct {
	Offset           uint32
	Size             uint32
	FileIndexPlusOne uint16
	Type             MSRDDataItemTypes
	Unk              [8]byte
}

type MSRDDataItemTypes uint16

const (
	MSRD_DATA_ITEM_TYPE_MODEL        = 0
	MSRD_DATA_ITEM_TYPE_SHADERBUNDLE = 1
	MSRD_DATA_ITEM_TYPE_TEXTURECACHE = 2
	MSRD_DATA_ITEM_TYPE_TEXTURE      = 3
)

type MSRDFileItem struct {
	CompressedSize   uint32
	UncompressedSize uint32
	Offset           uint32
}

type MSRD struct {
	Header     MSRDHeader
	MetaData   []byte
	Files      [][]byte
	MetaHeader MSRDMetaDataHeader
	DataItems  []MSRDDataItem
}

func ReadMSRD(reader io.ReadSeeker) (MSRD, error) {
	var header MSRDHeader
	if err := binary.Read(reader, furnace.TargetByteOrder, &header); err != nil {
		return MSRD{}, errors.New("Error reading msrd header: " + err.Error())
	}
	if header.Magic != MSRD_MAGIC {
		return MSRD{}, errors.New("Invalid msrd header")
	}

	reader.Seek(int64(header.MetaDataOffset), io.SeekStart)
	metaData := make([]byte, header.MetaDataSize)
	if _, err := reader.Read(metaData); err != nil {
		return MSRD{}, errors.New("Error reading msrd metadata: " + err.Error())
	}

	metaHeader := MSRDMetaDataHeader{}
	metaDataBuffer := bytes.NewBuffer(metaData)
	if err := binary.Read(metaDataBuffer, furnace.TargetByteOrder, &metaHeader); err != nil {
		return MSRD{}, errors.New("Error reading msrd meta header: " + err.Error())
	}

	reader.Seek(int64(header.MetaDataOffset+metaHeader.DataItemsTableOffset), io.SeekStart)
	dataItems := make([]MSRDDataItem, metaHeader.DataItemsCount)
	if err := binary.Read(reader, furnace.TargetByteOrder, &dataItems); err != nil {
		return MSRD{}, errors.New("Error reading msrd data items: " + err.Error())
	}

	reader.Seek(int64(header.MetaDataOffset+metaHeader.FileTableOffset), io.SeekStart)
	fileItems := make([]MSRDFileItem, metaHeader.FileCount)
	if err := binary.Read(reader, furnace.TargetByteOrder, &fileItems); err != nil {
		return MSRD{}, errors.New("Error reading msrd file items: " + err.Error())
	}

	files := make([][]byte, metaHeader.FileCount)
	for i := range fileItems {
		files[i] = make([]byte, fileItems[i].CompressedSize)
		reader.Seek(int64(fileItems[i].Offset), io.SeekStart)
		if _, err := reader.Read(files[i]); err != nil {
			return MSRD{}, errors.New("Error reading msrd file " + fmt.Sprint(i) + ": " + err.Error())
		}
	}

	return MSRD{
		Header:     header,
		MetaData:   metaData,
		MetaHeader: metaHeader,
		DataItems:  dataItems,
		Files:      files,
	}, nil
}
