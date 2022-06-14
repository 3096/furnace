package furnace

import (
	"github.com/3096/furnace/dds"
	"github.com/3096/furnace/utils"
)

// copied from PredatorCZ/XenoLib
// https://github.com/PredatorCZ/XenoLib/blob/2f14c0bd3765ee4439e91034028c0acb0493f95f/source/LBIM.cpp#L204
// GPLv3 License https://www.gnu.org/licenses/

func GetSwizzled(data []byte, width, height uint32, format dds.DXGIFormat) []byte {
	formatInfo := dds.DXGI_FORMAT_INFO_MAP[format]
	bytesPerBlock := formatInfo.BitsPerPixel * formatInfo.BlockSideLen * formatInfo.BlockSideLen / 8
	widthBlocks := utils.Align(width, formatInfo.BlockSideLen) / formatInfo.BlockSideLen
	heightBlocks := utils.Align(height, formatInfo.BlockSideLen) / formatInfo.BlockSideLen
	xBitsShift := 3
	for i := uint32(0); i < 4; i++ {
		if ((heightBlocks - 1) & (8 << i)) != 0 {
			xBitsShift += 1
		}
	}

	swizzled := make([]byte, len(data))
	curOffset := uint32(0)
	for y := uint32(0); y < heightBlocks; y++ {
		for x := uint32(0); x < widthBlocks; x++ {
			xRaw := x * bytesPerBlock
			swizzledOffset := ((y & 0xff80) * widthBlocks * bytesPerBlock) | ((y & 0x78) << 6) | ((y & 6) << 5) | ((y & 1) << 4) |
				((xRaw & 0xffc0) << xBitsShift) | ((xRaw & 0x20) << 3) | ((xRaw & 0x10) << 1) | (xRaw & 0xf)
			copy(swizzled[swizzledOffset:swizzledOffset+bytesPerBlock], data[curOffset:curOffset+bytesPerBlock])
			curOffset += bytesPerBlock
		}
	}

	return swizzled
}
