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
	DxgiFormat        DXGIFormat
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

var DX10_FORMAT = [4]byte{'D', 'X', '1', '0'}
var NON_DX10_FORMAT_MAP = map[[4]byte]DXGIFormat{
	{'D', 'X', 'T', '1'}: DXGI_FORMAT_BC1_UNORM,
	{'D', 'X', 'T', '2'}: DXGI_FORMAT_BC2_UNORM,
	{'D', 'X', 'T', '3'}: DXGI_FORMAT_BC2_UNORM,
	{'D', 'X', 'T', '4'}: DXGI_FORMAT_BC3_UNORM,
	{'D', 'X', 'T', '5'}: DXGI_FORMAT_BC3_UNORM,
	{'A', 'T', 'I', '1'}: DXGI_FORMAT_BC4_UNORM,
	{'A', 'T', 'I', '2'}: DXGI_FORMAT_BC5_UNORM,
}

func LoadDDS(ddsFileReader io.Reader) (DDSHeader, DDSHeaderDXT10, [][][]byte, error) {
	byteOrder := utils.NativeByteOrder()

	var magic [4]byte
	if _, err := ddsFileReader.Read(magic[:]); err != nil {
		return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Error when reading dds file: " + err.Error())
	}
	if magic != MAGIC {
		return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Invalid DDS file.")
	}

	var header DDSHeader
	if err := binary.Read(ddsFileReader, byteOrder, &header); err != nil {
		return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Error when reading dds file: " + err.Error())
	}
	if header.Size != 124 {
		return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Invalid DDS file size: " + fmt.Sprint(header.Size))
	}

	var headerDXT10 DDSHeaderDXT10
	if header.Dddpf.FourCC == DX10_FORMAT {
		if err := binary.Read(ddsFileReader, byteOrder, &headerDXT10); err != nil {
			return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Error when reading dds file: " + err.Error())
		}
	} else {
		dxgiFormat, found := NON_DX10_FORMAT_MAP[header.Dddpf.FourCC]
		if !found {
			return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Invalid DDS file format: " + fmt.Sprint(header.Dddpf.FourCC))
		}
		headerDXT10.DxgiFormat = dxgiFormat
	}

	dxgiFormatInfo, dxgiFormatInfoSupported := DXGI_FORMAT_INFO_MAP[headerDXT10.DxgiFormat]
	if !dxgiFormatInfoSupported {
		return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Unsupported DXGI format: " + fmt.Sprint(headerDXT10.DxgiFormat))
	}

	arraySize := headerDXT10.ArraySize
	if arraySize == 0 {
		arraySize = 1
	}
	surfaces := make([][][]byte, arraySize)
	for i := uint32(0); i < arraySize; i++ {
		surfaces[i] = make([][]byte, header.MipMapCount)
		w := header.Width
		h := header.Height
		for mipmapLevel := 0; mipmapLevel < int(header.MipMapCount); mipmapLevel++ {
			if w == 0 {
				w = 1
			}
			if h == 0 {
				h = 1
			}
			blockWidth := utils.Align(w, dxgiFormatInfo.BlockSideLen) / dxgiFormatInfo.BlockSideLen
			blockHeight := utils.Align(h, dxgiFormatInfo.BlockSideLen) / dxgiFormatInfo.BlockSideLen
			surfaces[i][mipmapLevel] = make([]byte, blockWidth*dxgiFormatInfo.BlockSideLen*blockHeight*dxgiFormatInfo.BlockSideLen*dxgiFormatInfo.BitsPerPixel/8)
			if _, err := ddsFileReader.Read(surfaces[i][mipmapLevel]); err != nil {
				return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Error when reading dds file: " + err.Error())
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

type DXGIFormat uint32

const (
	DXGI_FORMAT_UNKNOWN                    DXGIFormat = 0
	DXGI_FORMAT_R32G32B32A32_TYPELESS      DXGIFormat = 1
	DXGI_FORMAT_R32G32B32A32_FLOAT         DXGIFormat = 2
	DXGI_FORMAT_R32G32B32A32_UINT          DXGIFormat = 3
	DXGI_FORMAT_R32G32B32A32_SINT          DXGIFormat = 4
	DXGI_FORMAT_R32G32B32_TYPELESS         DXGIFormat = 5
	DXGI_FORMAT_R32G32B32_FLOAT            DXGIFormat = 6
	DXGI_FORMAT_R32G32B32_UINT             DXGIFormat = 7
	DXGI_FORMAT_R32G32B32_SINT             DXGIFormat = 8
	DXGI_FORMAT_R16G16B16A16_TYPELESS      DXGIFormat = 9
	DXGI_FORMAT_R16G16B16A16_FLOAT         DXGIFormat = 10
	DXGI_FORMAT_R16G16B16A16_UNORM         DXGIFormat = 11
	DXGI_FORMAT_R16G16B16A16_UINT          DXGIFormat = 12
	DXGI_FORMAT_R16G16B16A16_SNORM         DXGIFormat = 13
	DXGI_FORMAT_R16G16B16A16_SINT          DXGIFormat = 14
	DXGI_FORMAT_R32G32_TYPELESS            DXGIFormat = 15
	DXGI_FORMAT_R32G32_FLOAT               DXGIFormat = 16
	DXGI_FORMAT_R32G32_UINT                DXGIFormat = 17
	DXGI_FORMAT_R32G32_SINT                DXGIFormat = 18
	DXGI_FORMAT_R32G8X24_TYPELESS          DXGIFormat = 19
	DXGI_FORMAT_D32_FLOAT_S8X24_UINT       DXGIFormat = 20
	DXGI_FORMAT_R32_FLOAT_X8X24_TYPELESS   DXGIFormat = 21
	DXGI_FORMAT_X32_TYPELESS_G8X24_UINT    DXGIFormat = 22
	DXGI_FORMAT_R10G10B10A2_TYPELESS       DXGIFormat = 23
	DXGI_FORMAT_R10G10B10A2_UNORM          DXGIFormat = 24
	DXGI_FORMAT_R10G10B10A2_UINT           DXGIFormat = 25
	DXGI_FORMAT_R11G11B10_FLOAT            DXGIFormat = 26
	DXGI_FORMAT_R8G8B8A8_TYPELESS          DXGIFormat = 27
	DXGI_FORMAT_R8G8B8A8_UNORM             DXGIFormat = 28
	DXGI_FORMAT_R8G8B8A8_UNORM_SRGB        DXGIFormat = 29
	DXGI_FORMAT_R8G8B8A8_UINT              DXGIFormat = 30
	DXGI_FORMAT_R8G8B8A8_SNORM             DXGIFormat = 31
	DXGI_FORMAT_R8G8B8A8_SINT              DXGIFormat = 32
	DXGI_FORMAT_R16G16_TYPELESS            DXGIFormat = 33
	DXGI_FORMAT_R16G16_FLOAT               DXGIFormat = 34
	DXGI_FORMAT_R16G16_UNORM               DXGIFormat = 35
	DXGI_FORMAT_R16G16_UINT                DXGIFormat = 36
	DXGI_FORMAT_R16G16_SNORM               DXGIFormat = 37
	DXGI_FORMAT_R16G16_SINT                DXGIFormat = 38
	DXGI_FORMAT_R32_TYPELESS               DXGIFormat = 39
	DXGI_FORMAT_D32_FLOAT                  DXGIFormat = 40
	DXGI_FORMAT_R32_FLOAT                  DXGIFormat = 41
	DXGI_FORMAT_R32_UINT                   DXGIFormat = 42
	DXGI_FORMAT_R32_SINT                   DXGIFormat = 43
	DXGI_FORMAT_R24G8_TYPELESS             DXGIFormat = 44
	DXGI_FORMAT_D24_UNORM_S8_UINT          DXGIFormat = 45
	DXGI_FORMAT_R24_UNORM_X8_TYPELESS      DXGIFormat = 46
	DXGI_FORMAT_X24_TYPELESS_G8_UINT       DXGIFormat = 47
	DXGI_FORMAT_R8G8_TYPELESS              DXGIFormat = 48
	DXGI_FORMAT_R8G8_UNORM                 DXGIFormat = 49
	DXGI_FORMAT_R8G8_UINT                  DXGIFormat = 50
	DXGI_FORMAT_R8G8_SNORM                 DXGIFormat = 51
	DXGI_FORMAT_R8G8_SINT                  DXGIFormat = 52
	DXGI_FORMAT_R16_TYPELESS               DXGIFormat = 53
	DXGI_FORMAT_R16_FLOAT                  DXGIFormat = 54
	DXGI_FORMAT_D16_UNORM                  DXGIFormat = 55
	DXGI_FORMAT_R16_UNORM                  DXGIFormat = 56
	DXGI_FORMAT_R16_UINT                   DXGIFormat = 57
	DXGI_FORMAT_R16_SNORM                  DXGIFormat = 58
	DXGI_FORMAT_R16_SINT                   DXGIFormat = 59
	DXGI_FORMAT_R8_TYPELESS                DXGIFormat = 60
	DXGI_FORMAT_R8_UNORM                   DXGIFormat = 61
	DXGI_FORMAT_R8_UINT                    DXGIFormat = 62
	DXGI_FORMAT_R8_SNORM                   DXGIFormat = 63
	DXGI_FORMAT_R8_SINT                    DXGIFormat = 64
	DXGI_FORMAT_A8_UNORM                   DXGIFormat = 65
	DXGI_FORMAT_R1_UNORM                   DXGIFormat = 66
	DXGI_FORMAT_R9G9B9E5_SHAREDEXP         DXGIFormat = 67
	DXGI_FORMAT_R8G8_B8G8_UNORM            DXGIFormat = 68
	DXGI_FORMAT_G8R8_G8B8_UNORM            DXGIFormat = 69
	DXGI_FORMAT_BC1_TYPELESS               DXGIFormat = 70
	DXGI_FORMAT_BC1_UNORM                  DXGIFormat = 71
	DXGI_FORMAT_BC1_UNORM_SRGB             DXGIFormat = 72
	DXGI_FORMAT_BC2_TYPELESS               DXGIFormat = 73
	DXGI_FORMAT_BC2_UNORM                  DXGIFormat = 74
	DXGI_FORMAT_BC2_UNORM_SRGB             DXGIFormat = 75
	DXGI_FORMAT_BC3_TYPELESS               DXGIFormat = 76
	DXGI_FORMAT_BC3_UNORM                  DXGIFormat = 77
	DXGI_FORMAT_BC3_UNORM_SRGB             DXGIFormat = 78
	DXGI_FORMAT_BC4_TYPELESS               DXGIFormat = 79
	DXGI_FORMAT_BC4_UNORM                  DXGIFormat = 80
	DXGI_FORMAT_BC4_SNORM                  DXGIFormat = 81
	DXGI_FORMAT_BC5_TYPELESS               DXGIFormat = 82
	DXGI_FORMAT_BC5_UNORM                  DXGIFormat = 83
	DXGI_FORMAT_BC5_SNORM                  DXGIFormat = 84
	DXGI_FORMAT_B5G6R5_UNORM               DXGIFormat = 85
	DXGI_FORMAT_B5G5R5A1_UNORM             DXGIFormat = 86
	DXGI_FORMAT_B8G8R8A8_UNORM             DXGIFormat = 87
	DXGI_FORMAT_B8G8R8X8_UNORM             DXGIFormat = 88
	DXGI_FORMAT_R10G10B10_XR_BIAS_A2_UNORM DXGIFormat = 89
	DXGI_FORMAT_B8G8R8A8_TYPELESS          DXGIFormat = 90
	DXGI_FORMAT_B8G8R8A8_UNORM_SRGB        DXGIFormat = 91
	DXGI_FORMAT_B8G8R8X8_TYPELESS          DXGIFormat = 92
	DXGI_FORMAT_B8G8R8X8_UNORM_SRGB        DXGIFormat = 93
	DXGI_FORMAT_BC6H_TYPELESS              DXGIFormat = 94
	DXGI_FORMAT_BC6H_UF16                  DXGIFormat = 95
	DXGI_FORMAT_BC6H_SF16                  DXGIFormat = 96
	DXGI_FORMAT_BC7_TYPELESS               DXGIFormat = 97
	DXGI_FORMAT_BC7_UNORM                  DXGIFormat = 98
	DXGI_FORMAT_BC7_UNORM_SRGB             DXGIFormat = 99
	DXGI_FORMAT_AYUV                       DXGIFormat = 100
	DXGI_FORMAT_Y410                       DXGIFormat = 101
	DXGI_FORMAT_Y416                       DXGIFormat = 102
	DXGI_FORMAT_NV12                       DXGIFormat = 103
	DXGI_FORMAT_P010                       DXGIFormat = 104
	DXGI_FORMAT_P016                       DXGIFormat = 105
	DXGI_FORMAT_420_OPAQUE                 DXGIFormat = 106
	DXGI_FORMAT_YUY2                       DXGIFormat = 107
	DXGI_FORMAT_Y210                       DXGIFormat = 108
	DXGI_FORMAT_Y216                       DXGIFormat = 109
	DXGI_FORMAT_NV11                       DXGIFormat = 110
	DXGI_FORMAT_AI44                       DXGIFormat = 111
	DXGI_FORMAT_IA44                       DXGIFormat = 112
	DXGI_FORMAT_P8                         DXGIFormat = 113
	DXGI_FORMAT_A8P8                       DXGIFormat = 114
	DXGI_FORMAT_B4G4R4A4_UNORM             DXGIFormat = 115
	DXGI_FORMAT_P208                       DXGIFormat = 130
	DXGI_FORMAT_V208                       DXGIFormat = 131
	DXGI_FORMAT_V408                       DXGIFormat = 132
	DXGI_FORMAT_FORCE_UINT                 DXGIFormat = 0xffffffff
)

type DxgiFormatInfo struct {
	Name         string
	BitsPerPixel uint32
	BlockSideLen uint32
}

var DXGI_FORMAT_INFO_MAP = map[DXGIFormat]DxgiFormatInfo{
	DXGI_FORMAT_BC1_UNORM: {
		Name:         "BC1_UNORM",
		BitsPerPixel: 4,
		BlockSideLen: 4,
	},
	DXGI_FORMAT_BC2_UNORM: {
		Name:         "BC2_UNORM",
		BitsPerPixel: 8,
		BlockSideLen: 4,
	},
	DXGI_FORMAT_BC3_UNORM: {
		Name:         "BC3_UNORM",
		BitsPerPixel: 8,
		BlockSideLen: 4,
	},
	DXGI_FORMAT_BC4_UNORM: {
		Name:         "BC4_UNORM",
		BitsPerPixel: 4,
		BlockSideLen: 4,
	},
	DXGI_FORMAT_BC5_UNORM: {
		Name:         "BC5_UNORM",
		BitsPerPixel: 8,
		BlockSideLen: 4,
	},
	DXGI_FORMAT_BC7_UNORM: {
		Name:         "BC7_UNORM",
		BitsPerPixel: 8,
		BlockSideLen: 4,
	},
}
