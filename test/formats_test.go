package test

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/3096/furnace/furnace/formats"
	"github.com/3096/furnace/utils"
)

func TestMSRDReadWrite(t *testing.T) {
	msrdTestFilePath := "formats_testdata/wismt/pc079404.wismt"
	msrdOutFilePath := "formats_testdata/test-out/msrd-rw/pc079404.wismt"
	err := utils.EnsureDirectory(msrdOutFilePath)
	if err != nil {
		t.Fatal(err)
	}

	msrdTestFile, err := ioutil.ReadFile(msrdTestFilePath)
	if err != nil {
		t.Fatal(err)
	}

	msrdFileIn, err := os.Open(msrdTestFilePath)
	defer msrdFileIn.Close()
	if err != nil {
		t.Fatal(err)
	}
	msrd, err := formats.ReadMSRD(msrdFileIn)
	if err != nil {
		t.Fatal(err)
	}

	msrdFileOut, err := os.Create(msrdOutFilePath)
	defer msrdFileOut.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = formats.WriteMSRD(msrdFileOut, msrd)
	if err != nil {
		t.Fatal(err)
	}

	msrdOutFile, err := ioutil.ReadFile(msrdOutFilePath)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(msrdTestFile, msrdOutFile) {
		t.Errorf("Expected %s to be %s", msrdOutFilePath, msrdTestFilePath)
	}
}

func TestMSRDMips(t *testing.T) {
	msrdTestFilePath := "formats_testdata/wismt/pc079404.wismt"
	msrdOutFilePath := "formats_testdata/test-out/msrd-mips/pc079404.wismt"
	err := utils.EnsureDirectory(msrdOutFilePath)
	if err != nil {
		t.Fatal(err)
	}

	msrdFileIn, err := os.Open(msrdTestFilePath)
	defer msrdFileIn.Close()
	if err != nil {
		t.Fatal(err)
	}
	msrd, err := formats.ReadMSRD(msrdFileIn)
	if err != nil {
		t.Fatal(err)
	}

	mips, err := msrd.GetSplitMips()
	if err != nil {
		t.Fatal(err)
	}
	err = msrd.SetMips(mips)
	if err != nil {
		t.Fatal(err)
	}

	msrdFileOut, err := os.Create(msrdOutFilePath)
	defer msrdFileOut.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = formats.WriteMSRD(msrdFileOut, msrd)
	if err != nil {
		t.Fatal(err)
	}
}
