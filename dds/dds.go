package dds

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/3096/furnace/utils"
)

var MAGIC = [4]byte{'D', 'D', 'S', ' '}

type DDSHeader struct {
	Size              uint32
	Flags             uint32
	Height            uint32
	Width             uint32
	PitchOrLinearSize uint32
	Depth             uint32
	MipMapCount       uint32
	Reserved1         [11]uint32
	Dddpf             DDSPixelFormat
	Caps              uint32
	Caps2             uint32
	Caps3             uint32
	Caps4             uint32
	Reserved2         uint32
}

type DDSHeaderDXT10 struct {
	DxgiFormat        uint32
	ResourceDimension uint32
	MiscFlag          uint32
	ArraySize         uint32
	MiscFlags2        uint32
}

type DDSPixelFormat struct {
	Size        uint32
	Flags       uint32
	FourCC      [4]byte
	RGBBitCount uint32
	RBitMask    uint32
	GBitMask    uint32
	BBitMask    uint32
	ABitMask    uint32
}

var SUPPORTED_FORMAT = [4]byte{'D', 'X', '1', '0'}

func LoadDDS(ddsFileReader io.Reader) (DDSHeader, DDSHeaderDXT10, [][][]byte, error) {
	byteOrder := utils.NativeByteOrder()

	var magic [4]byte
	if _, err := ddsFileReader.Read(magic[:]); err != nil {
		return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Error when reading file: " + err.Error())
	}
	if magic != MAGIC {
		return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Invalid DDS file.")
	}

	var header DDSHeader
	if err := binary.Read(ddsFileReader, byteOrder, &header); err != nil {
		return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Error when reading file: " + err.Error())
	}
	if header.Size != 124 {
		return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Invalid DDS file size: " + fmt.Sprint(header.Size))
	}
	if header.Dddpf.FourCC != SUPPORTED_FORMAT {
		return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Unsupported DDS format: " + fmt.Sprint(header.Dddpf.FourCC))
	}

	var headerDXT10 DDSHeaderDXT10
	if err := binary.Read(ddsFileReader, byteOrder, &headerDXT10); err != nil {
		return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Error when reading file: " + err.Error())
	}

	dxgiFormatInfo, dxgiFormatInfoSupported := DXGI_FORMAT_INFO_MAP[headerDXT10.DxgiFormat]
	if !dxgiFormatInfoSupported {
		return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Unsupported DXGI format: " + fmt.Sprint(headerDXT10.DxgiFormat))
	}

	surfaces := make([][][]byte, headerDXT10.ArraySize)
	for i := uint32(0); i < headerDXT10.ArraySize; i++ {
		surfaces[i] = make([][]byte, header.MipMapCount)
		w := uint(header.Width)
		h := uint(header.Height)
		for mipmapLevel := 0; mipmapLevel < int(header.MipMapCount); mipmapLevel++ {
			if w == 0 || h == 0 {
				return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Invalid DDS file: width or height reaches 0 at surface " + fmt.Sprint(i) + " mipmap " + fmt.Sprint(mipmapLevel))
			}
			blockWidth := utils.Align(w, dxgiFormatInfo.PixelBlockSize) / dxgiFormatInfo.PixelBlockSize
			blockHeight := utils.Align(h, dxgiFormatInfo.PixelBlockSize) / dxgiFormatInfo.PixelBlockSize
			surfaces[i][mipmapLevel] = make([]byte, blockWidth*dxgiFormatInfo.PixelBlockSize*blockHeight*dxgiFormatInfo.PixelBlockSize*dxgiFormatInfo.BitsPerPixel/8)
			if _, err := ddsFileReader.Read(surfaces[i][mipmapLevel]); err != nil {
				return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Error when reading file: " + err.Error())
			}
			w /= 2
			h /= 2
		}
	}

	if _, err := ddsFileReader.Read(make([]byte, 1)); err == nil {
		return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Unexpected data after DDS file, unsupported.")
	}

	return header, headerDXT10, surfaces, nil
}

const (
	DXGI_FORMAT_UNKNOWN                    uint32 = 0
	DXGI_FORMAT_R32G32B32A32_TYPELESS      uint32 = 1
	DXGI_FORMAT_R32G32B32A32_FLOAT         uint32 = 2
	DXGI_FORMAT_R32G32B32A32_UINT          uint32 = 3
	DXGI_FORMAT_R32G32B32A32_SINT          uint32 = 4
	DXGI_FORMAT_R32G32B32_TYPELESS         uint32 = 5
	DXGI_FORMAT_R32G32B32_FLOAT            uint32 = 6
	DXGI_FORMAT_R32G32B32_UINT             uint32 = 7
	DXGI_FORMAT_R32G32B32_SINT             uint32 = 8
	DXGI_FORMAT_R16G16B16A16_TYPELESS      uint32 = 9
	DXGI_FORMAT_R16G16B16A16_FLOAT         uint32 = 10
	DXGI_FORMAT_R16G16B16A16_UNORM         uint32 = 11
	DXGI_FORMAT_R16G16B16A16_UINT          uint32 = 12
	DXGI_FORMAT_R16G16B16A16_SNORM         uint32 = 13
	DXGI_FORMAT_R16G16B16A16_SINT          uint32 = 14
	DXGI_FORMAT_R32G32_TYPELESS            uint32 = 15
	DXGI_FORMAT_R32G32_FLOAT               uint32 = 16
	DXGI_FORMAT_R32G32_UINT                uint32 = 17
	DXGI_FORMAT_R32G32_SINT                uint32 = 18
	DXGI_FORMAT_R32G8X24_TYPELESS          uint32 = 19
	DXGI_FORMAT_D32_FLOAT_S8X24_UINT       uint32 = 20
	DXGI_FORMAT_R32_FLOAT_X8X24_TYPELESS   uint32 = 21
	DXGI_FORMAT_X32_TYPELESS_G8X24_UINT    uint32 = 22
	DXGI_FORMAT_R10G10B10A2_TYPELESS       uint32 = 23
	DXGI_FORMAT_R10G10B10A2_UNORM          uint32 = 24
	DXGI_FORMAT_R10G10B10A2_UINT           uint32 = 25
	DXGI_FORMAT_R11G11B10_FLOAT            uint32 = 26
	DXGI_FORMAT_R8G8B8A8_TYPELESS          uint32 = 27
	DXGI_FORMAT_R8G8B8A8_UNORM             uint32 = 28
	DXGI_FORMAT_R8G8B8A8_UNORM_SRGB        uint32 = 29
	DXGI_FORMAT_R8G8B8A8_UINT              uint32 = 30
	DXGI_FORMAT_R8G8B8A8_SNORM             uint32 = 31
	DXGI_FORMAT_R8G8B8A8_SINT              uint32 = 32
	DXGI_FORMAT_R16G16_TYPELESS            uint32 = 33
	DXGI_FORMAT_R16G16_FLOAT               uint32 = 34
	DXGI_FORMAT_R16G16_UNORM               uint32 = 35
	DXGI_FORMAT_R16G16_UINT                uint32 = 36
	DXGI_FORMAT_R16G16_SNORM               uint32 = 37
	DXGI_FORMAT_R16G16_SINT                uint32 = 38
	DXGI_FORMAT_R32_TYPELESS               uint32 = 39
	DXGI_FORMAT_D32_FLOAT                  uint32 = 40
	DXGI_FORMAT_R32_FLOAT                  uint32 = 41
	DXGI_FORMAT_R32_UINT                   uint32 = 42
	DXGI_FORMAT_R32_SINT                   uint32 = 43
	DXGI_FORMAT_R24G8_TYPELESS             uint32 = 44
	DXGI_FORMAT_D24_UNORM_S8_UINT          uint32 = 45
	DXGI_FORMAT_R24_UNORM_X8_TYPELESS      uint32 = 46
	DXGI_FORMAT_X24_TYPELESS_G8_UINT       uint32 = 47
	DXGI_FORMAT_R8G8_TYPELESS              uint32 = 48
	DXGI_FORMAT_R8G8_UNORM                 uint32 = 49
	DXGI_FORMAT_R8G8_UINT                  uint32 = 50
	DXGI_FORMAT_R8G8_SNORM                 uint32 = 51
	DXGI_FORMAT_R8G8_SINT                  uint32 = 52
	DXGI_FORMAT_R16_TYPELESS               uint32 = 53
	DXGI_FORMAT_R16_FLOAT                  uint32 = 54
	DXGI_FORMAT_D16_UNORM                  uint32 = 55
	DXGI_FORMAT_R16_UNORM                  uint32 = 56
	DXGI_FORMAT_R16_UINT                   uint32 = 57
	DXGI_FORMAT_R16_SNORM                  uint32 = 58
	DXGI_FORMAT_R16_SINT                   uint32 = 59
	DXGI_FORMAT_R8_TYPELESS                uint32 = 60
	DXGI_FORMAT_R8_UNORM                   uint32 = 61
	DXGI_FORMAT_R8_UINT                    uint32 = 62
	DXGI_FORMAT_R8_SNORM                   uint32 = 63
	DXGI_FORMAT_R8_SINT                    uint32 = 64
	DXGI_FORMAT_A8_UNORM                   uint32 = 65
	DXGI_FORMAT_R1_UNORM                   uint32 = 66
	DXGI_FORMAT_R9G9B9E5_SHAREDEXP         uint32 = 67
	DXGI_FORMAT_R8G8_B8G8_UNORM            uint32 = 68
	DXGI_FORMAT_G8R8_G8B8_UNORM            uint32 = 69
	DXGI_FORMAT_BC1_TYPELESS               uint32 = 70
	DXGI_FORMAT_BC1_UNORM                  uint32 = 71
	DXGI_FORMAT_BC1_UNORM_SRGB             uint32 = 72
	DXGI_FORMAT_BC2_TYPELESS               uint32 = 73
	DXGI_FORMAT_BC2_UNORM                  uint32 = 74
	DXGI_FORMAT_BC2_UNORM_SRGB             uint32 = 75
	DXGI_FORMAT_BC3_TYPELESS               uint32 = 76
	DXGI_FORMAT_BC3_UNORM                  uint32 = 77
	DXGI_FORMAT_BC3_UNORM_SRGB             uint32 = 78
	DXGI_FORMAT_BC4_TYPELESS               uint32 = 79
	DXGI_FORMAT_BC4_UNORM                  uint32 = 80
	DXGI_FORMAT_BC4_SNORM                  uint32 = 81
	DXGI_FORMAT_BC5_TYPELESS               uint32 = 82
	DXGI_FORMAT_BC5_UNORM                  uint32 = 83
	DXGI_FORMAT_BC5_SNORM                  uint32 = 84
	DXGI_FORMAT_B5G6R5_UNORM               uint32 = 85
	DXGI_FORMAT_B5G5R5A1_UNORM             uint32 = 86
	DXGI_FORMAT_B8G8R8A8_UNORM             uint32 = 87
	DXGI_FORMAT_B8G8R8X8_UNORM             uint32 = 88
	DXGI_FORMAT_R10G10B10_XR_BIAS_A2_UNORM uint32 = 89
	DXGI_FORMAT_B8G8R8A8_TYPELESS          uint32 = 90
	DXGI_FORMAT_B8G8R8A8_UNORM_SRGB        uint32 = 91
	DXGI_FORMAT_B8G8R8X8_TYPELESS          uint32 = 92
	DXGI_FORMAT_B8G8R8X8_UNORM_SRGB        uint32 = 93
	DXGI_FORMAT_BC6H_TYPELESS              uint32 = 94
	DXGI_FORMAT_BC6H_UF16                  uint32 = 95
	DXGI_FORMAT_BC6H_SF16                  uint32 = 96
	DXGI_FORMAT_BC7_TYPELESS               uint32 = 97
	DXGI_FORMAT_BC7_UNORM                  uint32 = 98
	DXGI_FORMAT_BC7_UNORM_SRGB             uint32 = 99
	DXGI_FORMAT_AYUV                       uint32 = 100
	DXGI_FORMAT_Y410                       uint32 = 101
	DXGI_FORMAT_Y416                       uint32 = 102
	DXGI_FORMAT_NV12                       uint32 = 103
	DXGI_FORMAT_P010                       uint32 = 104
	DXGI_FORMAT_P016                       uint32 = 105
	DXGI_FORMAT_420_OPAQUE                 uint32 = 106
	DXGI_FORMAT_YUY2                       uint32 = 107
	DXGI_FORMAT_Y210                       uint32 = 108
	DXGI_FORMAT_Y216                       uint32 = 109
	DXGI_FORMAT_NV11                       uint32 = 110
	DXGI_FORMAT_AI44                       uint32 = 111
	DXGI_FORMAT_IA44                       uint32 = 112
	DXGI_FORMAT_P8                         uint32 = 113
	DXGI_FORMAT_A8P8                       uint32 = 114
	DXGI_FORMAT_B4G4R4A4_UNORM             uint32 = 115
	DXGI_FORMAT_P208                       uint32 = 130
	DXGI_FORMAT_V208                       uint32 = 131
	DXGI_FORMAT_V408                       uint32 = 132
	DXGI_FORMAT_FORCE_UINT                 uint32 = 0xffffffff
)

type DxgiFormatInfo struct {
	Name           string
	BitsPerPixel   uint
	PixelBlockSize uint
}

var DXGI_FORMAT_INFO_MAP = map[uint32]DxgiFormatInfo{
	DXGI_FORMAT_BC7_UNORM: {
		Name:           "BC7_UNORM",
		BitsPerPixel:   8,
		PixelBlockSize: 4,
	},
}
