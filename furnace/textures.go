package furnace

import (
	"github.com/3096/furnace/dds"
	"github.com/3096/furnace/utils"
)

// copied from PredatorCZ/XenoLib
// https://github.com/PredatorCZ/XenoLib/blob/2f14c0bd3765ee4439e91034028c0acb0493f95f/source/LBIM.cpp#L204
// GPLv3 License https://www.gnu.org/licenses/

func GetSwizzled(data []byte, width, height uint32, format dds.DxgiFormat) []byte {
	formatInfo := dds.DXGI_FORMAT_INFO_MAP[format]
	bytesPerBlock := formatInfo.BitsPerPixel * formatInfo.PixelBlockSize * formatInfo.PixelBlockSize / 8
	blockWidth := utils.Align(uint(width), formatInfo.PixelBlockSize) / formatInfo.PixelBlockSize
	blockHeight := utils.Align(uint(height), formatInfo.PixelBlockSize) / formatInfo.PixelBlockSize
	xBitsShift := 3
	for i := uint(0); i < formatInfo.PixelBlockSize; i++ {
		if ((height - 1) & (8 << i)) != 0 {
			xBitsShift += 1
		}
	}

	swizzled := make([]byte, len(data))
	curOffset := uint(0)
	for y := uint(0); y < blockHeight; y++ {
		for x := uint(0); x < blockWidth; x++ {
			xRaw := x * bytesPerBlock
			swizzledOffset := ((y & 0xff80) * blockWidth * bytesPerBlock) | ((y & 0x78) << 6) | ((y & 6) << 5) | ((y & 1) << 4) |
				((xRaw & 0xffc0) << xBitsShift) | ((xRaw & 0x20) << 3) | ((xRaw & 0x10) << 1) | (xRaw & 0xf)
			copy(swizzled[swizzledOffset:swizzledOffset+bytesPerBlock], data[curOffset:curOffset+bytesPerBlock])
			curOffset += bytesPerBlock
		}
	}

	return swizzled
}
