package furnace

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

type MIBLFormat uint32

const (
	MIBL_FORMAT_BC7_UNORM MIBLFormat = 77
)

var DXGIFormatToMIBLFormat = map[dds.DXGIFormat]MIBLFormat{
	dds.DXGI_FORMAT_BC7_UNORM: MIBL_FORMAT_BC7_UNORM,
}

func max(a, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}

func GetMIBL(mipData [][]byte, width, height uint32, format dds.DXGIFormat) ([]byte, error) {
	MIBLFormat, found := DXGIFormatToMIBLFormat[format]
	if !found {
		return nil, errors.New("Unsupported DXGI format: %v" + fmt.Sprint(format))
	}
	formatInfo := dds.DXGI_FORMAT_INFO_MAP[format]

	bytesPerBlock := formatInfo.BitsPerPixel * formatInfo.BlockSideLen * formatInfo.BlockSideLen / 8
	miblBuffer := bytes.Buffer{}
	curMipWidth := width
	curMipHeight := height
	for i := 1; i < len(mipData); i++ {
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
		Width:         width / 2,
		Height:        height / 2,
		LastMipWidth:  curMipWidth,
		LastMipHeight: curMipHeight,
		Format:        MIBLFormat,
		MipCount:      uint32(len(mipData) - 1),
		Version:       MIBL_VERSION,
		Magic:         MIBL_MAGIC,
	}
	miblBuffer.Write(make([]byte, uint(miblFooter.DataSize)-uint(miblBuffer.Len())-uint(unsafe.Sizeof(miblFooter))))
	binary.Write(&miblBuffer, furnace.TargetByteOrder, miblFooter)

	return miblBuffer.Bytes(), nil
}
