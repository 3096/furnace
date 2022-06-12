package test

import (
	"testing"

	"github.com/3096/furnace/commands"
	"github.com/3096/furnace/utils"
)

func TestReplaceTexturesInWismt(t *testing.T) {
	wismtTestFilePath := "formats_testdata/wismt/pc079404.wismt"
	wismtOutFilePath := "commands_testdata/test-out/replace-textures/pc079404.wismt"
	replacementTexturesDir := "commands_testdata/msrd-replaced-textures"

	err := utils.EnsureDirectory(wismtOutFilePath)
	if err != nil {
		t.Fatal(err)
	}

	err = commands.ReplaceTexturesInWismt(wismtTestFilePath, replacementTexturesDir, wismtOutFilePath)
	if err != nil {
		t.Fatal(err)
	}
}
