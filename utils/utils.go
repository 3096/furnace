package utils

import (
	"encoding/binary"
	"os"
	"path"
	"unsafe"
)

func NativeByteOrder() binary.ByteOrder {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)

	switch buf {
	case [2]byte{0xCD, 0xAB}:
		return binary.LittleEndian
	case [2]byte{0xAB, 0xCD}:
		return binary.BigEndian
	default:
		panic("Could not determine native endianness.")
	}
}

func Align(size, alignment uint32) uint32 {
	return (size + alignment - 1) &^ (alignment - 1)
}

type InPlaceWriter struct {
	buf []byte
	off int
}

func NewInPlaceWriter(buf []byte, off int) *InPlaceWriter {
	return &InPlaceWriter{buf, off}
}

func (w *InPlaceWriter) Write(p []byte) (n int, err error) {
	copy(w.buf[w.off:], p)
	w.off += len(p)
	return len(p), nil
}

func EnsureDirectory(targetPath string) error {
	dir := path.Dir(targetPath)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}
