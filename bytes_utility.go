package ojnet

import (
	"bytes"
	"encoding/binary"
)

func WriteUInt16(buffer *bytes.Buffer, val uint16) {
	temp := make([]byte, 2)
	binary.BigEndian.PutUint16(temp, val)
	buffer.Write(temp)
}

func WriteUInt32(buffer *bytes.Buffer, val uint32) {
	temp := make([]byte, 4)
	binary.BigEndian.PutUint32(temp, val)
	buffer.Write(temp)
}

func WriteUInt64(buffer *bytes.Buffer, val uint64) {
	temp := make([]byte, 8)
	binary.BigEndian.PutUint64(temp, val)
	buffer.Write(temp)
}
