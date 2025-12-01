package core

import (
	"bytes"
	"encoding/binary"
)

func IntToHex(num int64) []byte {
	buf := new(bytes.Buffer)
	// Ignore error because bytes.Buffer writes do not fail.
	_ = binary.Write(buf, binary.BigEndian, num)
	return buf.Bytes()
}
