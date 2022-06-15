package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math/bits"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/3096/furnace/dds"
	"github.com/3096/furnace/furnace"
	"github.com/3096/furnace/furnace/formats"
)

const INDEX_SEPARATOR = rune('.')
const RAW_REPLACE_DIR = "raw"
const FILE_INDEX_NO_ENTRY = -1

func ReplaceTexturesInWismt(inWismtPath, inTextureDir, outWismtPath string) error {
	fmt.Printf("Reading wismt file: %s...\n", inWismtPath)
	inWismtFile, err := os.Open(inWismtPath)
	defer inWismtFile.Close()
	if err != nil {
		return err
	}
	wismt, err := formats.ReadMSRD(inWismtFile)
	if err != nil {
		return err
	}
	wismtCachedTextures, err := wismt.GetCachedTextures()
	if err != nil {
		return err
	}

	fmt.Printf("Dispatching file read routines...\n")

	inTextureDirFileInfos, err := ioutil.ReadDir(inTextureDir)
	if err != nil {
		return err
	}

	rawReplaceDir := filepath.Join(inTextureDir, RAW_REPLACE_DIR)
	inRawReplaceDirFileInfos, _ := ioutil.ReadDir(rawReplaceDir)

	fileReadChan := make(chan *FileReadResult, len(inTextureDirFileInfos)+len(inRawReplaceDirFileInfos))
	routinesRunning := 0

	for _, inTextureFileInfo := range inTextureDirFileInfos {
		if inTextureFileInfo.IsDir() {
			continue
		}

		inTextureId, err := strconv.Atoi(inTextureFileInfo.Name()[:strings.IndexRune(inTextureFileInfo.Name(), INDEX_SEPARATOR)])
		if err != nil {
			fmt.Printf("Skipping %s: no id number found in filename, please use <id.name.dds> naming format\n", inTextureFileInfo.Name())
			continue
		}
		if inTextureId >= int(wismt.TextureInfoHeader.TextureCount) {
			fmt.Printf("Skipping %s: id number is out of range\n", inTextureFileInfo.Name())
			continue
		}

		origCachedTexture := wismtCachedTextures[inTextureId]
		inTexturePath := filepath.Join(inTextureDir, inTextureFileInfo.Name())
		inTextureIndex, hasFileEntry := wismt.TextureIdToIndexMap[formats.MSRDTextureId(inTextureId)]
		if hasFileEntry {
			msrdFileIndex := formats.MSRD_FILE_INDEX_TEXTURE_START + inTextureIndex
			xbc1Header, err := formats.ReadXBC1Header(bytes.NewReader(wismt.CompressedFiles[msrdFileIndex]))
			if err != nil {
				fmt.Printf("Skipping %s: %s\n", inTextureFileInfo.Name(), err)
				continue
			}

			go ReadTexture(inTexturePath, msrdFileIndex, formats.MSRDTextureId(inTextureId), origCachedTexture, xbc1Header.Name, fileReadChan)

		} else {
			go ReadTexture(inTexturePath, FILE_INDEX_NO_ENTRY, formats.MSRDTextureId(inTextureId), origCachedTexture, [0x1C]byte{}, fileReadChan)
		}

		routinesRunning++
	}

	for _, inRawReplaceFileInfo := range inRawReplaceDirFileInfos {
		if inRawReplaceFileInfo.IsDir() {
			continue
		}

		inRawReplaceIndex, err := strconv.Atoi(
			inRawReplaceFileInfo.Name()[:strings.IndexRune(inRawReplaceFileInfo.Name(), INDEX_SEPARATOR)])
		if err != nil {
			fmt.Printf("Skipping %s: no index found in filename\n", inRawReplaceFileInfo.Name())
			continue
		}
		if inRawReplaceIndex < 0 || inRawReplaceIndex >= len(wismt.CompressedFiles) {
			fmt.Printf("Skipping %s: index out of range\n", inRawReplaceFileInfo.Name())
			continue
		}
		xbc1Header, err := formats.ReadXBC1Header(bytes.NewReader(wismt.CompressedFiles[inRawReplaceIndex]))
		if err != nil {
			fmt.Printf("Skipping %s: %s\n", inRawReplaceFileInfo.Name(), err)
			continue
		}

		inRawReplacePath := filepath.Join(rawReplaceDir, inRawReplaceFileInfo.Name())
		go ReadRaw(inRawReplacePath, inRawReplaceIndex, xbc1Header.Name, fileReadChan)
		routinesRunning++
	}

	fmt.Printf("Reading wimdo file: %s...\n", outWismtPath)
	inWimdoFile, err := os.Open(strings.TrimSuffix(inWismtPath, filepath.Ext(inWismtPath)) + ".wimdo")
	defer inWimdoFile.Close()
	if err != nil {
		return errors.New(err.Error() + "\nMake sure you place it in the same directory as the wismt file")
	}
	wimdo, err := formats.ReadMXMD(inWimdoFile)
	if err != nil {
		return errors.New("Could not read wimdo file: " + err.Error())
	}
	wimdoHeader, err := wimdo.GetHeader()
	if err != nil {
		return errors.New("Could not read wimdo header: " + err.Error())
	}
	if wimdoHeader.UncachedTexturesOffset == 0 {
		return errors.New("Could not find uncached textures offset in wimdo file")
	}

	fmt.Printf("Handling file reads...\n")
	mipsMIBLs, err := wismt.GetSplitMips()
	if err != nil {
		return err
	}
	totalFilesReplaced := 0
	for routinesRunning > 0 {
		result := <-fileReadChan
		routinesRunning--
		if result.Err != nil {
			fmt.Printf("Skipped due to error - %s: %s\n", result.Err, result.Path)
			continue
		}

		if result.CompressedData != nil {
			wismt.SetCompressedFileData(result.FileIndex, result.CompressedData)
		}
		if result.TextureReadResult.MipsMIBL != nil {
			mipsMIBLs[result.FileIndex-formats.MSRD_FILE_INDEX_TEXTURE_START] = result.TextureReadResult.MipsMIBL
		}
		if result.TextureReadResult.CacheMIBL != nil {
			wismtCachedTextures[result.TextureReadResult.TextureId] = result.TextureReadResult.CacheMIBL
		}
		totalFilesReplaced++
		fmt.Printf("Successfully placed %s\n", result.Path)
	}

	if totalFilesReplaced == 0 {
		return errors.New("No files replaced")
	}

	fmt.Printf("Saving cached textures to file%d...\n", formats.MSRD_FILE_INDEX_0)
	err = wismt.SetCachedTextures(wismtCachedTextures)
	if err != nil {
		return errors.New("Could not save cached textures: " + err.Error())
	}

	fmt.Printf("Saving mipmaps to file%d...\n", formats.MSRD_FILE_INDEX_MIPS)
	err = wismt.SetMips(mipsMIBLs)
	if err != nil {
		return err
	}

	fmt.Printf("Saving wismt file: %s...\n", outWismtPath)
	outWismtFile, err := os.Create(outWismtPath)
	defer outWismtFile.Close()
	if err != nil {
		return err
	}
	err = formats.WriteMSRD(outWismtFile, wismt)
	if err != nil {
		return err
	}

	outWimdoPath := strings.TrimSuffix(outWismtPath, filepath.Ext(outWismtPath)) + ".wimdo"
	fmt.Printf("Saving wimdo file: %s...\n", outWimdoPath)
	copy(wimdo[wimdoHeader.UncachedTexturesOffset:], formats.MXMD(wismt.MetaData))
	outWimdoFile, err := os.Create(outWimdoPath)
	defer outWimdoFile.Close()
	if err != nil {
		return err
	}
	err = formats.WriteMXMD(outWimdoFile, wimdo)
	if err != nil {
		return err
	}

	fmt.Printf("Done: replaced %d files, output: %s\n", totalFilesReplaced, outWismtPath)
	return nil
}

type TextureReadResult struct {
	TextureId formats.MSRDTextureId
	MipsMIBL  formats.MIBL
	CacheMIBL formats.MIBL
}

type FileReadResult struct {
	Err               error
	Path              string
	FileIndex         int
	CompressedData    formats.XBC1
	TextureReadResult TextureReadResult
}

func ReadTexture(texturePath string, index int, textureId formats.MSRDTextureId,
	origCacheMIBL formats.MIBL, xbc1Name [0x1C]byte, channel chan *FileReadResult) {

	textureFile, err := os.Open(texturePath)
	defer textureFile.Close()
	if err != nil {
		channel <- &FileReadResult{Err: err, Path: texturePath}
		return
	}

	ddsHeader, ddsHeaderDXT10, mips, err := dds.LoadDDS(textureFile)
	if err != nil {
		channel <- &FileReadResult{Err: err, Path: texturePath}
		return
	}
	if len(mips[0]) <= 1 {
		channel <- &FileReadResult{Err: errors.New("missing mipmaps"), Path: texturePath}
		return
	}

	origCacheMIBLFooter, err := origCacheMIBL.GetFooter()
	if err != nil {
		channel <- &FileReadResult{Err: err, Path: texturePath}
		return
	}

	if origCacheMIBLFooter.Width > ddsHeader.Width || origCacheMIBLFooter.Height > ddsHeader.Height {
		channel <- &FileReadResult{Err: errors.New("texture size mismatch"), Path: texturePath}
		return
	}

	cachedMipLevel := bits.Len32(ddsHeader.Width/origCacheMIBLFooter.Width) - 1
	if ddsHeader.Height>>cachedMipLevel != origCacheMIBLFooter.Height {
		channel <- &FileReadResult{Err: errors.New("texture ratio mismatch"), Path: texturePath}
		return
	}

	cacheMIBL, err := formats.NewMIBL(mips[0][cachedMipLevel:cachedMipLevel+int(origCacheMIBLFooter.MipCount)],
		origCacheMIBLFooter.Width, origCacheMIBLFooter.Height, ddsHeaderDXT10.DxgiFormat, 0)

	if index == FILE_INDEX_NO_ENTRY {
		channel <- &FileReadResult{
			Path:      texturePath,
			FileIndex: index,
			TextureReadResult: TextureReadResult{
				TextureId: textureId,
				CacheMIBL: cacheMIBL,
			},
		}
		return
	}

	compressedTextureData, err := formats.CompressToXBC1(xbc1Name,
		furnace.GetSwizzled(mips[0][0], ddsHeader.Width, ddsHeader.Height, ddsHeaderDXT10.DxgiFormat))
	if err != nil {
		channel <- &FileReadResult{Err: err, Path: texturePath}
		return
	}

	mipsMIBL, err := formats.NewMIBL(mips[0], ddsHeader.Width, ddsHeader.Height, ddsHeaderDXT10.DxgiFormat, 1)
	if err != nil {
		channel <- &FileReadResult{Err: err, Path: texturePath}
		return
	}

	channel <- &FileReadResult{
		Path:           texturePath,
		FileIndex:      index,
		CompressedData: compressedTextureData,
		TextureReadResult: TextureReadResult{
			TextureId: textureId,
			MipsMIBL:  mipsMIBL,
			CacheMIBL: cacheMIBL,
		},
	}
}

func ReadRaw(rawPath string, index int, xbc1Name [0x1C]byte, channel chan *FileReadResult) {
	data, err := ioutil.ReadFile(rawPath)
	if err != nil {
		channel <- &FileReadResult{Err: err, Path: rawPath}
		return
	}

	compressedData, err := formats.CompressToXBC1(xbc1Name, data)
	if err != nil {
		channel <- &FileReadResult{Err: err, Path: rawPath}
		return
	}

	channel <- &FileReadResult{
		Path:           rawPath,
		FileIndex:      index,
		CompressedData: compressedData,
	}
}
