package formats

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/3096/furnace/furnace"
	"github.com/3096/furnace/utils"
)

const MSRD_MAGIC uint32 = 'M'<<24 | 'S'<<16 | 'R'<<8 | 'D'

type MSRDHeader struct {
	Magic          uint32
	Version        uint32
	MetaDataSize   uint32
	MetaDataOffset uint32
}

type MSRDMetaData []byte

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

type MSRDFile []byte

const MSRD_FILE_ALIGN uint32 = 0x10

const MSRD_FILE_INDEX_0 = 0
const MSRD_FILE_INDEX_MIPS = 1
const MSRD_FILE_INDEX_TEXTURE_START = 2

type MSRD struct {
	Header     MSRDHeader
	MetaData   MSRDMetaData
	Files      []MSRDFile
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
	metaData := make(MSRDMetaData, header.MetaDataSize)
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

	files := (make([]MSRDFile, metaHeader.FileCount))
	for i := range fileItems {
		files[i] = make(MSRDFile, fileItems[i].CompressedSize)
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

func WriteMSRD(writer io.WriteSeeker, msrd MSRD) error {
	if err := binary.Write(utils.NewInPlaceWriter(msrd.MetaData, 0), furnace.TargetByteOrder, &msrd.MetaHeader); err != nil {
		return errors.New("Error writing msrd meta header: " + err.Error())
	}

	if err := binary.Write(utils.NewInPlaceWriter(msrd.MetaData, int(msrd.MetaHeader.DataItemsTableOffset)), furnace.TargetByteOrder, &msrd.DataItems); err != nil {
		return errors.New("Error writing msrd data items: " + err.Error())
	}

	fileItemsWriter := utils.NewInPlaceWriter(msrd.MetaData, int(msrd.MetaHeader.FileTableOffset))
	curFileOffset := msrd.Header.MetaDataOffset + msrd.Header.MetaDataSize
	for i := range msrd.Files {
		xbc1Header, err := ReadXBC1Header(bytes.NewReader(msrd.Files[i]))
		if err != nil {
			return errors.New("Error reading xbc1 header: " + err.Error())
		}
		if err := binary.Write(fileItemsWriter, furnace.TargetByteOrder, &MSRDFileItem{
			CompressedSize:   uint32(len(msrd.Files[i])),
			UncompressedSize: xbc1Header.UncompressedSize,
			Offset:           curFileOffset,
		}); err != nil {
			return errors.New("Error writing msrd file items: " + err.Error())
		}
		curFileOffset += uint32(len(msrd.Files[i]))
	}

	if err := binary.Write(writer, furnace.TargetByteOrder, &msrd.Header); err != nil {
		return errors.New("Error writing msrd header: " + err.Error())
	}
	if _, err := writer.Write(msrd.MetaData); err != nil {
		return errors.New("Error writing msrd metadata: " + err.Error())
	}
	for i := range msrd.Files {
		if _, err := writer.Write(msrd.Files[i]); err != nil {
			return errors.New("Error writing msrd file " + fmt.Sprint(i) + ": " + err.Error())
		}
	}
	return nil
}

func (msrd *MSRD) SetFileData(index int, data XBC1) {
	msrd.Files[index] = append([]byte(data), make([]byte, MSRD_FILE_ALIGN-uint32(len(data))%MSRD_FILE_ALIGN)...)
}

func (msrd *MSRD) GetSplitMips() ([]MIBL, error) {
	var mips []MIBL
	_, jointMipsFile, err := ExtractXBC1(bytes.NewReader(msrd.Files[MSRD_FILE_INDEX_MIPS]))
	if err != nil {
		return nil, errors.New("Error extracting mips file: " + err.Error())
	}
	for _, dataItem := range msrd.DataItems {
		if dataItem.Type == MSRD_DATA_ITEM_TYPE_TEXTURE {
			mips = append(mips, MIBL(jointMipsFile[dataItem.Offset:dataItem.Offset+dataItem.Size]))
		}
	}
	if len(mips) != int(msrd.MetaHeader.TextureIdsCount) {
		return nil, errors.New("Invalid mips count")
	}
	return mips, nil
}

func (msrd *MSRD) SetMips(splitMips []MIBL) error {
	if len(splitMips) != int(msrd.MetaHeader.TextureIdsCount) {
		return errors.New("Invalid number of mips")
	}

	jointMipsMSRDFileBuffer := bytes.NewBuffer(make([]byte, 0))
	mipsOffsets := []uint32{0}
	for i, curMips := range splitMips {
		jointMipsMSRDFileBuffer.Write(curMips)
		mipsOffsets = append(mipsOffsets, mipsOffsets[i]+uint32(len(curMips)))
	}
	jointMipsMSRDFileHeader, err := ReadXBC1Header(bytes.NewReader(msrd.Files[MSRD_FILE_INDEX_MIPS]))
	if err != nil {
		return errors.New("Error reading mips header: " + err.Error())
	}
	jointMipsMSRDFileData, err := CompressToXBC1(jointMipsMSRDFileHeader.Name, jointMipsMSRDFileBuffer.Bytes())
	if err != nil {
		return errors.New("Error writing mips file: " + err.Error())
	}
	msrd.SetFileData(MSRD_FILE_INDEX_MIPS, jointMipsMSRDFileData)

	for i, dataItem := range msrd.DataItems {
		if dataItem.Type == MSRD_DATA_ITEM_TYPE_TEXTURE {
			textureIndex := dataItem.FileIndexPlusOne - 1 - MSRD_FILE_INDEX_TEXTURE_START
			msrd.DataItems[i].Size = uint32(len(splitMips[textureIndex]))
			msrd.DataItems[i].Offset = mipsOffsets[textureIndex]
		}
	}

	return nil
}
