package formats

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"unsafe"

	"github.com/3096/furnace/dds"
	"github.com/3096/furnace/furnace"
	"github.com/3096/furnace/utils"
)

const MIBL_MAGIC uint32 = 'M'<<24 | 'I'<<16 | 'B'<<8 | 'L'
const MIBL_VERSION uint32 = 10001
const MIBL_ALIGN_SIZE uint32 = 0x1000
const MIBL_MIN_WIDTH uint32 = 16
const MIBL_MIN_HEIGHT uint32 = 32

// I guessed some of these, might not be entirely correct
type MIBLFooter struct {
	DataSize      uint32
	AlignSize     uint32
	Width         uint32
	Height        uint32
	LastMipWidth  uint32
	LastMipHeight uint32
	Format        MIBLFormat
	MipCount      uint32
	Version       uint32
	Magic         uint32
}

type MIBL []byte

type MIBLFormat uint32

const (
	MIBL_FORMAT_R8G8B8A8_UNORM            = 37
	MIBL_FORMAT_BC1_UNORM      MIBLFormat = 66
	MIBL_FORMAT_BC2_UNORM      MIBLFormat = 67
	MIBL_FORMAT_BC3_UNORM      MIBLFormat = 68
	MIBL_FORMAT_BC4_UNORM      MIBLFormat = 73
	MIBL_FORMAT_BC5_UNORM      MIBLFormat = 75
	MIBL_FORMAT_BC7_UNORM      MIBLFormat = 77
)

var DXGIFormatToMIBLFormat = map[dds.DXGIFormat]MIBLFormat{
	dds.DXGI_FORMAT_R8G8B8A8_UNORM: MIBL_FORMAT_R8G8B8A8_UNORM,
	dds.DXGI_FORMAT_BC1_UNORM:      MIBL_FORMAT_BC1_UNORM,
	dds.DXGI_FORMAT_BC2_UNORM:      MIBL_FORMAT_BC2_UNORM,
	dds.DXGI_FORMAT_BC3_UNORM:      MIBL_FORMAT_BC3_UNORM,
	dds.DXGI_FORMAT_BC4_UNORM:      MIBL_FORMAT_BC4_UNORM,
	dds.DXGI_FORMAT_BC5_UNORM:      MIBL_FORMAT_BC5_UNORM,
	dds.DXGI_FORMAT_BC7_UNORM:      MIBL_FORMAT_BC7_UNORM,
}

func max(a, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}

func NewMIBL(mipData [][]byte, width, height uint32, format dds.DXGIFormat, startingIndex int) (MIBL, error) {
	miblFormat, found := DXGIFormatToMIBLFormat[format]
	if !found {
		return nil, errors.New("Unsupported DXGI format: " + fmt.Sprint(format))
	}

	formatInfo := dds.DXGI_FORMAT_INFO_MAP[format]
	bytesPerBlock := formatInfo.BitsPerPixel * formatInfo.BlockSideLen * formatInfo.BlockSideLen / 8
	miblBuffer := bytes.Buffer{}

	if startingIndex < 0 || startingIndex >= len(mipData) {
		return nil, errors.New("Invalid starting index: " + fmt.Sprint(startingIndex))
	}
	curMipWidth := width >> startingIndex << 1
	curMipHeight := height >> startingIndex << 1

	for i := startingIndex; i < len(mipData); i++ {
		curMipWidth /= 2
		curMipHeight /= 2
		adjustedWidth := utils.Align(max(curMipWidth, MIBL_MIN_WIDTH), formatInfo.BlockSideLen)
		adjustedHeight := utils.Align(max(curMipHeight, MIBL_MIN_HEIGHT), formatInfo.BlockSideLen)
		adjustedMipData := make([]byte, adjustedWidth*adjustedHeight*formatInfo.BitsPerPixel/8)

		if curMipWidth < (MIBL_MIN_WIDTH) {
			heightBlocks := utils.Align(curMipHeight, formatInfo.BlockSideLen) / formatInfo.BlockSideLen
			rowSize := utils.Align(curMipWidth, formatInfo.BlockSideLen) / formatInfo.BlockSideLen * bytesPerBlock
			for row := uint32(0); row < heightBlocks; row++ {
				rowOffset := row * rowSize
				copy(adjustedMipData[row*adjustedWidth/formatInfo.BlockSideLen*bytesPerBlock:],
					mipData[i][rowOffset:rowOffset+rowSize])
			}
		} else {
			copy(adjustedMipData, mipData[i])
		}

		miblBuffer.Write(furnace.GetSwizzled(adjustedMipData, adjustedWidth, adjustedHeight, format))
	}

	miblFooter := MIBLFooter{
		DataSize:      utils.Align(uint32(miblBuffer.Len())+uint32(unsafe.Sizeof(MIBLFooter{})), MIBL_ALIGN_SIZE),
		AlignSize:     MIBL_ALIGN_SIZE,
		Width:         width >> startingIndex,
		Height:        height >> startingIndex,
		LastMipWidth:  curMipWidth,
		LastMipHeight: curMipHeight,
		Format:        miblFormat,
		MipCount:      uint32(len(mipData) - startingIndex),
		Version:       MIBL_VERSION,
		Magic:         MIBL_MAGIC,
	}
	miblBuffer.Write(make([]byte, uint(miblFooter.DataSize)-uint(miblBuffer.Len())-uint(unsafe.Sizeof(miblFooter))))
	binary.Write(&miblBuffer, furnace.TargetByteOrder, miblFooter)

	return miblBuffer.Bytes(), nil
}

func (mibl *MIBL) GetFooter() (MIBLFooter, error) {
	if len(*mibl) < int(unsafe.Sizeof(MIBLFooter{})) {
		return MIBLFooter{}, errors.New("Invalid MIBL length")
	}
	var footer MIBLFooter
	err := binary.Read(bytes.NewReader((*mibl)[len(*mibl)-int(unsafe.Sizeof(MIBLFooter{})):]), furnace.TargetByteOrder, &footer)
	if err != nil {
		return MIBLFooter{}, err
	}
	return footer, nil
}
