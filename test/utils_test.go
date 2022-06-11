package utils

import (
	"bytes"
	"testing"

	"github.com/3096/furnace/utils"
)

func TestInPlaceWriter(t *testing.T) {
	slice := make([]byte, 16)

	writer := utils.NewInPlaceWriter(slice, 0)
	writer.Write([]byte{1, 2, 3, 4})
	if !bytes.Equal(slice, []byte{1, 2, 3, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}) {
		t.Errorf("Expected slice to be [1, 2, 3, 4], got %v", slice)
	}

	writer = utils.NewInPlaceWriter(slice, 8)
	writer.Write([]byte{5, 6, 7, 8})
	if !bytes.Equal(slice, []byte{1, 2, 3, 4, 0, 0, 0, 0, 5, 6, 7, 8, 0, 0, 0, 0}) {
		t.Errorf("Expected slice to be [1, 2, 3, 4, 0, 0, 0, 0, 5, 6, 7, 8, 0, 0, 0, 0], got %v", slice)
	}
}
