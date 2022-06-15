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

	TextureIdsCount   uint32
	TextureIdsOffset  uint32
	TextureInfoOffset uint32
}

type MSRDDataItem struct {
	Offset           uint32
	Size             uint32
	FileIndexPlusOne uint16
	Type             MSRDDataItemType
	Unk              [8]byte
}

type MSRDDataItemType uint16

const (
	MSRD_DATA_ITEM_TYPE_MODEL        MSRDDataItemType = 0
	MSRD_DATA_ITEM_TYPE_SHADERBUNDLE MSRDDataItemType = 1
	MSRD_DATA_ITEM_TYPE_TEXTURECACHE MSRDDataItemType = 2
	MSRD_DATA_ITEM_TYPE_TEXTURE      MSRDDataItemType = 3
)

type MSRDFileItem struct {
	CompressedSize   uint32
	UncompressedSize uint32
	Offset           uint32
}

const MSRD_FILE_ALIGN uint32 = 0x10

const MSRD_FILE_INDEX_0 = 0
const MSRD_FILE_INDEX_MIPS = 1
const MSRD_FILE_INDEX_TEXTURE_START = 2

type MSRDTextureId uint16

type MSRDTextureInfoHeader struct {
	TextureCount       uint32
	TextureBlockSize   uint32
	Unk                uint32
	TextureNamesOffset uint32
}

type MSRDTextureInfoItem struct {
	Unk         [4]byte
	CacheSize   uint32
	CacheOffset uint32
	NameOffset  uint32
}

type MSRD struct {
	Header              MSRDHeader
	MetaData            MSRDMetaData
	CompressedFiles     []XBC1
	MetaHeader          MSRDMetaDataHeader
	DataItems           []MSRDDataItem
	TextureIdToIndexMap map[MSRDTextureId]int
	TextureInfoHeader   MSRDTextureInfoHeader
	TextureInfoItems    []MSRDTextureInfoItem
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

	compressedFiles := (make([]XBC1, metaHeader.FileCount))
	for i := range fileItems {
		compressedFiles[i] = make(XBC1, fileItems[i].CompressedSize)
		reader.Seek(int64(fileItems[i].Offset), io.SeekStart)
		if _, err := reader.Read(compressedFiles[i]); err != nil {
			return MSRD{}, errors.New("Error reading msrd file " + fmt.Sprint(i) + ": " + err.Error())
		}
	}

	reader.Seek(int64(header.MetaDataOffset+metaHeader.TextureIdsOffset), io.SeekStart)
	textureIds := make([]MSRDTextureId, metaHeader.TextureIdsCount)
	if err := binary.Read(reader, furnace.TargetByteOrder, &textureIds); err != nil {
		return MSRD{}, errors.New("Error reading msrd texture ids: " + err.Error())
	}

	textureIdToIndexMap := make(map[MSRDTextureId]int)
	for i, id := range textureIds {
		textureIdToIndexMap[id] = i
	}

	textureInfoHeader := MSRDTextureInfoHeader{}
	reader.Seek(int64(header.MetaDataOffset+metaHeader.TextureInfoOffset), io.SeekStart)
	if err := binary.Read(reader, furnace.TargetByteOrder, &textureInfoHeader); err != nil {
		return MSRD{}, errors.New("Error reading msrd texture info header: " + err.Error())
	}

	textureInfoItems := make([]MSRDTextureInfoItem, textureInfoHeader.TextureCount)
	if err := binary.Read(reader, furnace.TargetByteOrder, &textureInfoItems); err != nil {
		return MSRD{}, errors.New("Error reading msrd texture info items: " + err.Error())
	}

	return MSRD{
		Header:              header,
		MetaData:            metaData,
		MetaHeader:          metaHeader,
		DataItems:           dataItems,
		CompressedFiles:     compressedFiles,
		TextureIdToIndexMap: textureIdToIndexMap,
		TextureInfoHeader:   textureInfoHeader,
		TextureInfoItems:    textureInfoItems,
	}, nil
}

func WriteMSRD(writer io.WriteSeeker, msrd MSRD) error {
	// I wrote this assuming the data size of metadata won't change...
	// if the need arises, however, it might need some overhaul

	if err := binary.Write(utils.NewInPlaceWriter(msrd.MetaData, 0), furnace.TargetByteOrder, &msrd.MetaHeader); err != nil {
		return errors.New("Error writing msrd meta header: " + err.Error())
	}

	if err := binary.Write(utils.NewInPlaceWriter(msrd.MetaData, int(msrd.MetaHeader.DataItemsTableOffset)), furnace.TargetByteOrder, &msrd.DataItems); err != nil {
		return errors.New("Error writing msrd data items: " + err.Error())
	}

	fileItemsWriter := utils.NewInPlaceWriter(msrd.MetaData, int(msrd.MetaHeader.FileTableOffset))
	curFileOffset := msrd.Header.MetaDataOffset + msrd.Header.MetaDataSize
	for i := range msrd.CompressedFiles {
		xbc1Header, err := ReadXBC1Header(bytes.NewReader(msrd.CompressedFiles[i]))
		if err != nil {
			return errors.New("Error reading xbc1 header: " + err.Error())
		}
		if err := binary.Write(fileItemsWriter, furnace.TargetByteOrder, &MSRDFileItem{
			CompressedSize:   uint32(len(msrd.CompressedFiles[i])),
			UncompressedSize: xbc1Header.UncompressedSize,
			Offset:           curFileOffset,
		}); err != nil {
			return errors.New("Error writing msrd file items: " + err.Error())
		}
		curFileOffset += uint32(len(msrd.CompressedFiles[i]))
	}

	textureInfoWriter := utils.NewInPlaceWriter(msrd.MetaData, int(msrd.MetaHeader.TextureInfoOffset))
	if err := binary.Write(textureInfoWriter, furnace.TargetByteOrder, &msrd.TextureInfoHeader); err != nil {
		return errors.New("Error writing msrd texture info header: " + err.Error())
	}
	if err := binary.Write(textureInfoWriter, furnace.TargetByteOrder, &msrd.TextureInfoItems); err != nil {
		return errors.New("Error writing msrd texture info items: " + err.Error())
	}

	if err := binary.Write(writer, furnace.TargetByteOrder, &msrd.Header); err != nil {
		return errors.New("Error writing msrd header: " + err.Error())
	}
	if _, err := writer.Write(msrd.MetaData); err != nil {
		return errors.New("Error writing msrd metadata: " + err.Error())
	}
	for i := range msrd.CompressedFiles {
		if _, err := writer.Write(msrd.CompressedFiles[i]); err != nil {
			return errors.New("Error writing msrd file " + fmt.Sprint(i) + ": " + err.Error())
		}
	}
	return nil
}

func (msrd *MSRD) GetDataItemsByType(dataItemType MSRDDataItemType) []MSRDDataItem {
	var result []MSRDDataItem
	for _, item := range msrd.DataItems {
		if item.Type == dataItemType {
			result = append(result, item)
		}
	}
	return result
}

func (msrd *MSRD) SetCompressedFileData(index int, data XBC1) {
	msrd.CompressedFiles[index] = append([]byte(data), make([]byte, MSRD_FILE_ALIGN-uint32(len(data))%MSRD_FILE_ALIGN)...)
}

func (msrd *MSRD) GetSplitMips() ([]MIBL, error) {
	var mips []MIBL
	_, jointMipsFile, err := ExtractXBC1(bytes.NewReader(msrd.CompressedFiles[MSRD_FILE_INDEX_MIPS]))
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
	jointMipsMSRDFileHeader, err := ReadXBC1Header(bytes.NewReader(msrd.CompressedFiles[MSRD_FILE_INDEX_MIPS]))
	if err != nil {
		return errors.New("Error reading mips header: " + err.Error())
	}
	jointMipsMSRDFileData, err := CompressToXBC1(jointMipsMSRDFileHeader.Name, jointMipsMSRDFileBuffer.Bytes())
	if err != nil {
		return errors.New("Error writing mips file: " + err.Error())
	}
	msrd.SetCompressedFileData(MSRD_FILE_INDEX_MIPS, jointMipsMSRDFileData)

	for i, dataItem := range msrd.DataItems {
		if dataItem.Type == MSRD_DATA_ITEM_TYPE_TEXTURE {
			textureIndex := dataItem.FileIndexPlusOne - 1 - MSRD_FILE_INDEX_TEXTURE_START
			msrd.DataItems[i].Size = uint32(len(splitMips[textureIndex]))
			msrd.DataItems[i].Offset = mipsOffsets[textureIndex]
		}
	}

	return nil
}

func (msrd *MSRD) GetCachedTextures() ([]MIBL, error) {
	var textures []MIBL

	cachedTextureDataItems := msrd.GetDataItemsByType(MSRD_DATA_ITEM_TYPE_TEXTURECACHE)
	if len(cachedTextureDataItems) != 1 {
		return nil, errors.New("Invalid number of cached texture data items")
	}
	cachedTextureDataOffset := cachedTextureDataItems[0].Offset

	_, file0Content, err := ExtractXBC1(bytes.NewReader(msrd.CompressedFiles[MSRD_FILE_INDEX_0]))
	if err != nil {
		return nil, errors.New("Error extracting file 0: " + err.Error())
	}

	for _, textureInfoItem := range msrd.TextureInfoItems {
		textureCacheOffset := cachedTextureDataOffset + textureInfoItem.CacheOffset
		textures = append(textures, MIBL(file0Content[textureCacheOffset:textureCacheOffset+textureInfoItem.CacheSize]))
	}

	return textures, nil
}

func (msrd *MSRD) SetCachedTextures(textures []MIBL) error {
	if len(textures) != int(msrd.TextureInfoHeader.TextureCount) {
		return errors.New("Invalid number of textures")
	}

	file0XBC1Header, file0Content, err := ExtractXBC1(bytes.NewReader(msrd.CompressedFiles[MSRD_FILE_INDEX_0]))
	if err != nil {
		return errors.New("Error extracting file 0: " + err.Error())
	}

	cachedTextureDataItems := msrd.GetDataItemsByType(MSRD_DATA_ITEM_TYPE_TEXTURECACHE)
	if len(cachedTextureDataItems) != 1 {
		return errors.New("Invalid number of cached texture data items")
	}
	cachedTextureDataOffset := cachedTextureDataItems[0].Offset
	if int(cachedTextureDataOffset+cachedTextureDataItems[0].Size) != len(file0Content) {
		// I'm assuming texture cache is at the end of file 0 cuz I'm feeling lazy
		// if it's not, this should catch it
		return errors.New("Additional data after texture cache detected in file 0, unsupported")
	}

	textureCacheWriter := utils.NewInPlaceWriter(file0Content, int(cachedTextureDataOffset))
	curTextureCacheOffset := uint32(0)
	for i, curTexture := range textures {
		msrd.TextureInfoItems[i].CacheOffset = curTextureCacheOffset
		msrd.TextureInfoItems[i].CacheSize = uint32(len(curTexture))
		textureCacheWriter.Write(curTexture)
		curTextureCacheOffset += uint32(len(curTexture))
	}

	file0Compressed, err := CompressToXBC1(file0XBC1Header.Name, file0Content)
	if err != nil {
		return errors.New("Error writing file 0: " + err.Error())
	}
	msrd.SetCompressedFileData(MSRD_FILE_INDEX_0, file0Compressed)

	return nil
}
