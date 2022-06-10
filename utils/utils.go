package utils

import (
	"encoding/binary"
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
