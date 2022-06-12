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

func ReplaceTexturesInWismt(inWismtPath, inTextureDir, outWismtPath string) error {
	inWismtFile, err := os.Open(inWismtPath)
	defer inWismtFile.Close()
	if err != nil {
		return err
	}
	wismt, err := formats.ReadMSRD(inWismtFile)
	if err != nil {
		return err
	}

	inTextureDirFileInfos, err := ioutil.ReadDir(inTextureDir)
	if err != nil {
		return err
	}

	textureReadChan := make(chan *TextureReadResult, len(inTextureDirFileInfos))
	routinesRunning := 0
	for _, inTextureDirFileInfo := range inTextureDirFileInfos {
		if inTextureDirFileInfo.IsDir() {
			continue
		}

		inTextureIndex, err := strconv.Atoi(
			inTextureDirFileInfo.Name()[:strings.IndexRune(inTextureDirFileInfo.Name(), INDEX_SEPARATOR)])
		if err != nil {
			fmt.Printf("Skipping %s: no index found in filename\n", inTextureDirFileInfo.Name())
			continue
		}
		if inTextureIndex < 0 || inTextureIndex >= len(wismt.Files)-formats.MSRD_FILE_INDEX_TEXTURE_START {
			fmt.Printf("Skipping %s: index out of range\n", inTextureDirFileInfo.Name())
			continue
		}
		xbc1Header, err := formats.ReadXBC1Header(bytes.NewReader(wismt.Files[formats.MSRD_FILE_INDEX_TEXTURE_START+inTextureIndex]))
		if err != nil {
			fmt.Printf("Skipping %s: %s\n", inTextureDirFileInfo.Name(), err)
			continue
		}

		inTexturePath := filepath.Join(inTextureDir, inTextureDirFileInfo.Name())
		go ReadTexture(inTexturePath, inTextureIndex, xbc1Header.Name, textureReadChan)
		routinesRunning++
	}

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

	mipsMIBLs, err := wismt.GetSplitMips()
	if err != nil {
		return err
	}

	totalTexturesReplaced := 0
	for routinesRunning > 0 {
		result := <-textureReadChan
		routinesRunning--
		if result.Err != nil {
			fmt.Printf("Skipped due to error - %s: %s\n", result.Err, result.Path)
			continue
		}

		mipsMIBLs[result.TextureIndex] = result.MipsMIBL
		wismt.SetFileData(formats.MSRD_FILE_INDEX_TEXTURE_START+result.TextureIndex, result.TextureData)
		totalTexturesReplaced++
	}

	if totalTexturesReplaced == 0 {
		return errors.New("No textures replaced")
	}

	err = wismt.SetMips(mipsMIBLs)
	if err != nil {
		return err
	}

	outWismtFile, err := os.Create(outWismtPath)
	defer outWismtFile.Close()
	if err != nil {
		return err
	}
	err = formats.WriteMSRD(outWismtFile, wismt)
	if err != nil {
		return err
	}

	copy(wimdo[wimdoHeader.UncachedTexturesOffset:], formats.MXMD(wismt.MetaData))
	outWimdoFile, err := os.Create(strings.TrimSuffix(outWismtPath, filepath.Ext(outWismtPath)) + ".wimdo")
	defer outWimdoFile.Close()
	if err != nil {
		return err
	}
	err = formats.WriteMXMD(outWimdoFile, wimdo)
	if err != nil {
		return err
	}

	fmt.Printf("Done: replaced %d textures, output: %s\n", totalTexturesReplaced, outWismtPath)
	return nil
}

type TextureReadResult struct {
	Err          error
	Path         string
	TextureIndex int
	TextureData  formats.XBC1
	MipsMIBL     formats.MIBL
}

func ReadTexture(texturePath string, index int, xbc1Name [0x1C]byte, channel chan *TextureReadResult) {
	textureFile, err := os.Open(texturePath)
	defer textureFile.Close()
	if err != nil {
		channel <- &TextureReadResult{Err: err, Path: texturePath}
		return
	}

	ddsHeader, ddsHeaderDXT10, mips, err := dds.LoadDDS(textureFile)
	if err != nil {
		channel <- &TextureReadResult{Err: err, Path: texturePath}
		return
	}
	if len(mips[0]) <= 1 {
		channel <- &TextureReadResult{Err: errors.New("missing mipmaps"), Path: texturePath}
		return
	}

	compressedTextureData, err := formats.CompressToXBC1(xbc1Name,
		furnace.GetSwizzled(mips[0][0], ddsHeader.Width, ddsHeader.Height, ddsHeaderDXT10.DxgiFormat))
	if err != nil {
		channel <- &TextureReadResult{Err: err, Path: texturePath}
		return
	}

	mipsMIBL, err := formats.GetMIBL(mips[0], ddsHeader.Width, ddsHeader.Height, ddsHeaderDXT10.DxgiFormat)

	channel <- &TextureReadResult{
		Path:         texturePath,
		TextureIndex: index,
		TextureData:  compressedTextureData,
		MipsMIBL:     mipsMIBL,
	}
}
