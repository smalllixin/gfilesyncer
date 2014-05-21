package gfilesyncer
import (
	"time"
	"encoding/binary"
	"bytes"
	// "log"
)
const (
	PackageTypeSyncRequest = 0x01 //Sync a file

)

const DefaultChunkSize = 512

type SyncHeader struct {
	Version byte
	PackageType byte
	ChunkSize int16
	FileLength int64
	FileModTime time.Time
	FileName string
}

func NewSyncHeaderFromBytes(buf []byte) *SyncHeader {
	h := new(SyncHeader)
	headerReader := bytes.NewReader(buf)
	binary.Read(headerReader, binary.LittleEndian, &h.Version)
	binary.Read(headerReader, binary.LittleEndian, &h.PackageType)
	binary.Read(headerReader, binary.LittleEndian, &h.ChunkSize)
	binary.Read(headerReader, binary.LittleEndian, &h.FileLength)
	var timestamp int64
	binary.Read(headerReader, binary.LittleEndian, &timestamp)
	h.FileModTime = time.Unix(timestamp, 0)
	return h
}

func (s *SyncHeader) toBytes() []byte {
	packageLen := uint16(syncHeaderLen())+uint16(len(s.FileName)+1)
	b := make([]byte, 0, packageLen)
	buf := bytes.NewBuffer(b)
	binary.Write(buf, binary.LittleEndian, s.Version)
	binary.Write(buf, binary.LittleEndian, s.PackageType)
	binary.Write(buf, binary.LittleEndian, s.ChunkSize)
	binary.Write(buf, binary.LittleEndian, s.FileLength)
	binary.Write(buf, binary.LittleEndian, s.FileModTime.Unix())
	buf.WriteString(s.FileName+"\n")
	bbytes := buf.Bytes()
	// log.Printf("bbytes:% x", bbytes)
	// log.Println("version:", bbytes[0])
	return bbytes
}

func syncHeaderLen() int {
	return 1+1+2+8+8
}

const (
	VerifyResultError = 0xff
	VerifyResultTransferToMe = 0x01
	VerifyResultDoNext = 0x02
	VerifyResultNeedCompare = 0x03
)

type SyncReply struct {
	VerifyResult byte
}