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
	DxgiFormat        DxgiFormat
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
	if header.Dddpf.FourCC != SUPPORTED_FORMAT {
		return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Unsupported DDS format: " + fmt.Sprint(header.Dddpf.FourCC))
	}

	var headerDXT10 DDSHeaderDXT10
	if err := binary.Read(ddsFileReader, byteOrder, &headerDXT10); err != nil {
		return DDSHeader{}, DDSHeaderDXT10{}, nil, errors.New("Error when reading dds file: " + err.Error())
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

type DxgiFormat uint32

const (
	DXGI_FORMAT_UNKNOWN                    DxgiFormat = 0
	DXGI_FORMAT_R32G32B32A32_TYPELESS      DxgiFormat = 1
	DXGI_FORMAT_R32G32B32A32_FLOAT         DxgiFormat = 2
	DXGI_FORMAT_R32G32B32A32_UINT          DxgiFormat = 3
	DXGI_FORMAT_R32G32B32A32_SINT          DxgiFormat = 4
	DXGI_FORMAT_R32G32B32_TYPELESS         DxgiFormat = 5
	DXGI_FORMAT_R32G32B32_FLOAT            DxgiFormat = 6
	DXGI_FORMAT_R32G32B32_UINT             DxgiFormat = 7
	DXGI_FORMAT_R32G32B32_SINT             DxgiFormat = 8
	DXGI_FORMAT_R16G16B16A16_TYPELESS      DxgiFormat = 9
	DXGI_FORMAT_R16G16B16A16_FLOAT         DxgiFormat = 10
	DXGI_FORMAT_R16G16B16A16_UNORM         DxgiFormat = 11
	DXGI_FORMAT_R16G16B16A16_UINT          DxgiFormat = 12
	DXGI_FORMAT_R16G16B16A16_SNORM         DxgiFormat = 13
	DXGI_FORMAT_R16G16B16A16_SINT          DxgiFormat = 14
	DXGI_FORMAT_R32G32_TYPELESS            DxgiFormat = 15
	DXGI_FORMAT_R32G32_FLOAT               DxgiFormat = 16
	DXGI_FORMAT_R32G32_UINT                DxgiFormat = 17
	DXGI_FORMAT_R32G32_SINT                DxgiFormat = 18
	DXGI_FORMAT_R32G8X24_TYPELESS          DxgiFormat = 19
	DXGI_FORMAT_D32_FLOAT_S8X24_UINT       DxgiFormat = 20
	DXGI_FORMAT_R32_FLOAT_X8X24_TYPELESS   DxgiFormat = 21
	DXGI_FORMAT_X32_TYPELESS_G8X24_UINT    DxgiFormat = 22
	DXGI_FORMAT_R10G10B10A2_TYPELESS       DxgiFormat = 23
	DXGI_FORMAT_R10G10B10A2_UNORM          DxgiFormat = 24
	DXGI_FORMAT_R10G10B10A2_UINT           DxgiFormat = 25
	DXGI_FORMAT_R11G11B10_FLOAT            DxgiFormat = 26
	DXGI_FORMAT_R8G8B8A8_TYPELESS          DxgiFormat = 27
	DXGI_FORMAT_R8G8B8A8_UNORM             DxgiFormat = 28
	DXGI_FORMAT_R8G8B8A8_UNORM_SRGB        DxgiFormat = 29
	DXGI_FORMAT_R8G8B8A8_UINT              DxgiFormat = 30
	DXGI_FORMAT_R8G8B8A8_SNORM             DxgiFormat = 31
	DXGI_FORMAT_R8G8B8A8_SINT              DxgiFormat = 32
	DXGI_FORMAT_R16G16_TYPELESS            DxgiFormat = 33
	DXGI_FORMAT_R16G16_FLOAT               DxgiFormat = 34
	DXGI_FORMAT_R16G16_UNORM               DxgiFormat = 35
	DXGI_FORMAT_R16G16_UINT                DxgiFormat = 36
	DXGI_FORMAT_R16G16_SNORM               DxgiFormat = 37
	DXGI_FORMAT_R16G16_SINT                DxgiFormat = 38
	DXGI_FORMAT_R32_TYPELESS               DxgiFormat = 39
	DXGI_FORMAT_D32_FLOAT                  DxgiFormat = 40
	DXGI_FORMAT_R32_FLOAT                  DxgiFormat = 41
	DXGI_FORMAT_R32_UINT                   DxgiFormat = 42
	DXGI_FORMAT_R32_SINT                   DxgiFormat = 43
	DXGI_FORMAT_R24G8_TYPELESS             DxgiFormat = 44
	DXGI_FORMAT_D24_UNORM_S8_UINT          DxgiFormat = 45
	DXGI_FORMAT_R24_UNORM_X8_TYPELESS      DxgiFormat = 46
	DXGI_FORMAT_X24_TYPELESS_G8_UINT       DxgiFormat = 47
	DXGI_FORMAT_R8G8_TYPELESS              DxgiFormat = 48
	DXGI_FORMAT_R8G8_UNORM                 DxgiFormat = 49
	DXGI_FORMAT_R8G8_UINT                  DxgiFormat = 50
	DXGI_FORMAT_R8G8_SNORM                 DxgiFormat = 51
	DXGI_FORMAT_R8G8_SINT                  DxgiFormat = 52
	DXGI_FORMAT_R16_TYPELESS               DxgiFormat = 53
	DXGI_FORMAT_R16_FLOAT                  DxgiFormat = 54
	DXGI_FORMAT_D16_UNORM                  DxgiFormat = 55
	DXGI_FORMAT_R16_UNORM                  DxgiFormat = 56
	DXGI_FORMAT_R16_UINT                   DxgiFormat = 57
	DXGI_FORMAT_R16_SNORM                  DxgiFormat = 58
	DXGI_FORMAT_R16_SINT                   DxgiFormat = 59
	DXGI_FORMAT_R8_TYPELESS                DxgiFormat = 60
	DXGI_FORMAT_R8_UNORM                   DxgiFormat = 61
	DXGI_FORMAT_R8_UINT                    DxgiFormat = 62
	DXGI_FORMAT_R8_SNORM                   DxgiFormat = 63
	DXGI_FORMAT_R8_SINT                    DxgiFormat = 64
	DXGI_FORMAT_A8_UNORM                   DxgiFormat = 65
	DXGI_FORMAT_R1_UNORM                   DxgiFormat = 66
	DXGI_FORMAT_R9G9B9E5_SHAREDEXP         DxgiFormat = 67
	DXGI_FORMAT_R8G8_B8G8_UNORM            DxgiFormat = 68
	DXGI_FORMAT_G8R8_G8B8_UNORM            DxgiFormat = 69
	DXGI_FORMAT_BC1_TYPELESS               DxgiFormat = 70
	DXGI_FORMAT_BC1_UNORM                  DxgiFormat = 71
	DXGI_FORMAT_BC1_UNORM_SRGB             DxgiFormat = 72
	DXGI_FORMAT_BC2_TYPELESS               DxgiFormat = 73
	DXGI_FORMAT_BC2_UNORM                  DxgiFormat = 74
	DXGI_FORMAT_BC2_UNORM_SRGB             DxgiFormat = 75
	DXGI_FORMAT_BC3_TYPELESS               DxgiFormat = 76
	DXGI_FORMAT_BC3_UNORM                  DxgiFormat = 77
	DXGI_FORMAT_BC3_UNORM_SRGB             DxgiFormat = 78
	DXGI_FORMAT_BC4_TYPELESS               DxgiFormat = 79
	DXGI_FORMAT_BC4_UNORM                  DxgiFormat = 80
	DXGI_FORMAT_BC4_SNORM                  DxgiFormat = 81
	DXGI_FORMAT_BC5_TYPELESS               DxgiFormat = 82
	DXGI_FORMAT_BC5_UNORM                  DxgiFormat = 83
	DXGI_FORMAT_BC5_SNORM                  DxgiFormat = 84
	DXGI_FORMAT_B5G6R5_UNORM               DxgiFormat = 85
	DXGI_FORMAT_B5G5R5A1_UNORM             DxgiFormat = 86
	DXGI_FORMAT_B8G8R8A8_UNORM             DxgiFormat = 87
	DXGI_FORMAT_B8G8R8X8_UNORM             DxgiFormat = 88
	DXGI_FORMAT_R10G10B10_XR_BIAS_A2_UNORM DxgiFormat = 89
	DXGI_FORMAT_B8G8R8A8_TYPELESS          DxgiFormat = 90
	DXGI_FORMAT_B8G8R8A8_UNORM_SRGB        DxgiFormat = 91
	DXGI_FORMAT_B8G8R8X8_TYPELESS          DxgiFormat = 92
	DXGI_FORMAT_B8G8R8X8_UNORM_SRGB        DxgiFormat = 93
	DXGI_FORMAT_BC6H_TYPELESS              DxgiFormat = 94
	DXGI_FORMAT_BC6H_UF16                  DxgiFormat = 95
	DXGI_FORMAT_BC6H_SF16                  DxgiFormat = 96
	DXGI_FORMAT_BC7_TYPELESS               DxgiFormat = 97
	DXGI_FORMAT_BC7_UNORM                  DxgiFormat = 98
	DXGI_FORMAT_BC7_UNORM_SRGB             DxgiFormat = 99
	DXGI_FORMAT_AYUV                       DxgiFormat = 100
	DXGI_FORMAT_Y410                       DxgiFormat = 101
	DXGI_FORMAT_Y416                       DxgiFormat = 102
	DXGI_FORMAT_NV12                       DxgiFormat = 103
	DXGI_FORMAT_P010                       DxgiFormat = 104
	DXGI_FORMAT_P016                       DxgiFormat = 105
	DXGI_FORMAT_420_OPAQUE                 DxgiFormat = 106
	DXGI_FORMAT_YUY2                       DxgiFormat = 107
	DXGI_FORMAT_Y210                       DxgiFormat = 108
	DXGI_FORMAT_Y216                       DxgiFormat = 109
	DXGI_FORMAT_NV11                       DxgiFormat = 110
	DXGI_FORMAT_AI44                       DxgiFormat = 111
	DXGI_FORMAT_IA44                       DxgiFormat = 112
	DXGI_FORMAT_P8                         DxgiFormat = 113
	DXGI_FORMAT_A8P8                       DxgiFormat = 114
	DXGI_FORMAT_B4G4R4A4_UNORM             DxgiFormat = 115
	DXGI_FORMAT_P208                       DxgiFormat = 130
	DXGI_FORMAT_V208                       DxgiFormat = 131
	DXGI_FORMAT_V408                       DxgiFormat = 132
	DXGI_FORMAT_FORCE_UINT                 DxgiFormat = 0xffffffff
)

type DxgiFormatInfo struct {
	Name           string
	BitsPerPixel   uint
	PixelBlockSize uint
}

var DXGI_FORMAT_INFO_MAP = map[DxgiFormat]DxgiFormatInfo{
	DXGI_FORMAT_BC7_UNORM: {
		Name:           "BC7_UNORM",
		BitsPerPixel:   8,
		PixelBlockSize: 4,
	},
}
