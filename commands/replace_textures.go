package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
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
		inTextureIndex, found := wismt.TextureIdToIndexMap[uint16(inTextureId)]
		if !found {
			fmt.Printf("Skipping %s: index out of range\n", inTextureFileInfo.Name())
			continue
		}

		msrdFileIndex := formats.MSRD_FILE_INDEX_TEXTURE_START + inTextureIndex
		xbc1Header, err := formats.ReadXBC1Header(bytes.NewReader(wismt.Files[msrdFileIndex]))
		if err != nil {
			fmt.Printf("Skipping %s: %s\n", inTextureFileInfo.Name(), err)
			continue
		}

		inTexturePath := filepath.Join(inTextureDir, inTextureFileInfo.Name())
		go ReadTexture(inTexturePath, msrdFileIndex, xbc1Header.Name, fileReadChan)
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
		if inRawReplaceIndex < 0 || inRawReplaceIndex >= len(wismt.Files) {
			fmt.Printf("Skipping %s: index out of range\n", inRawReplaceFileInfo.Name())
			continue
		}
		xbc1Header, err := formats.ReadXBC1Header(bytes.NewReader(wismt.Files[inRawReplaceIndex]))
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

		if result.MipsMIBL != nil {
			mipsMIBLs[result.FileIndex-formats.MSRD_FILE_INDEX_TEXTURE_START] = result.MipsMIBL
		}
		wismt.SetFileData(result.FileIndex, result.CompressedData)
		totalFilesReplaced++
		fmt.Printf("Successfully placed %s\n", result.Path)
	}

	if totalFilesReplaced == 0 {
		return errors.New("No files replaced")
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

type FileReadResult struct {
	Err            error
	Path           string
	FileIndex      int
	CompressedData formats.XBC1
	MipsMIBL       formats.MIBL
}

func ReadTexture(texturePath string, index int, xbc1Name [0x1C]byte, channel chan *FileReadResult) {
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
		MipsMIBL:       mipsMIBL,
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
