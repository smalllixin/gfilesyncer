package gfilesyncer

import (
	"os"
	"io"
	"crypto/md5"
	"math"
	"log"
	"bytes"
	"encoding/binary"
)

type CheckSumPack struct{
	Adler32 uint32
	Md5 []byte
	Idx int
}

const CheckSumPackSize=(4+16)

func CheckSumPackArrayToBytes(hashes []CheckSumPack) []byte {
	checksumNumber := int32(len(hashes))
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, checksumNumber)
	for _, x := range hashes {
		binary.Write(buf, binary.LittleEndian, x.Adler32)
		binary.Write(buf, binary.LittleEndian, x.Md5)
	}
	return buf.Bytes()
}

//b: pure data, size of array is not encoded
func NewCheckSumPackArrayFromBytes(b []byte, arrayLen int32) []CheckSumPack{
	buf := bytes.NewReader(b)
	hashesResult := make([]CheckSumPack, 0, arrayLen)
	var tempAdler32 uint32
	for i := int32(0); i < arrayLen; i ++ {
		binary.Read(buf, binary.LittleEndian, &tempAdler32)
		tempMd5 := make([]byte, 16)
		buf.Read(tempMd5)
		h := CheckSumPack{tempAdler32,tempMd5, int(i)}
		hashesResult = append(hashesResult, h)
	}

	return hashesResult
}

type RollingChecksum struct {
	file *os.File
	offset int64
	chunkSize int16
	adlerHash *AdlerDigest
}

func NewRollingCheckSum(f *os.File, chunkSize int16) *RollingChecksum {
	s := new(RollingChecksum)
	s.file = f
	s.chunkSize = chunkSize
	s.offset = 0
	s.adlerHash = NewAdler32()
	return s
}

func (s *RollingChecksum) setOffset(readOffset int64) {
	s.offset = readOffset
}

func (s *RollingChecksum) moveToNextByte() {
	s.offset ++
}

func (s *RollingChecksum) moveToNextChunk() {
	s.offset += int64(s.chunkSize)
}

func (s *RollingChecksum) RollingCheck(checkSumMap map[uint32]CheckSumPack) error {

	chunkSize := s.chunkSize
	readBuf := make([]byte, chunkSize)
	var nr int
	nr = int(1)
	pPosLeftNotSame := uint32(0)
	pLeft := uint32(0)
	// pRight := uint32(0)

	md5Hash := md5.New()
	s.adlerHash.Reset()
	var err error
	for nr > 0 {
		nr, err = s.file.Read(readBuf)
		if err != nil && err != io.EOF {
			log.Println("Compute reading :", err)
			return err
		}
		if nr > 0 {
			buf := readBuf[0:nr]
			s.adlerHash.Write(buf)
			localAdlerChecksum := s.adlerHash.Sum32()

			checkSumPack, ok := checkSumMap[localAdlerChecksum]
			if ok {
				//weak checksum(alder32) bingo
				//compare strong checksum(md5)
				md5Hash.Reset()
				md5Hash.Write(buf)
				localMd5Checksum := md5Hash.Sum(nil)
				if compareMd5Value(localMd5Checksum, checkSumPack.Md5) {
					//strong checksum hit
					//
				} else {
					//this not same yet
					pLeft ++
				}

			} else {
				//this not same data chunk
				pLeft ++
				123
			}
			_ = checkSumPack
		}
	}
	return nil
}

func compareMd5Value(v1 []byte, v2 []byte) bool {
	for i, x := range v1 {
		if x != v2[i] {
			return false
		}
	}
	return true
}

func (s *RollingChecksum) SumEveryChunk() ([]CheckSumPack, error) {
	var err error
	
	fi, err := s.file.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := fi.Size()

	s.adlerHash.Reset()
	
	chunkNumber := int(math.Ceil(float64(fileSize)/float64(s.chunkSize)))
	
	chunkHashes := make([]CheckSumPack, 0, chunkNumber)
	chunkBuf := make([]byte, s.chunkSize)

	md5Hash := md5.New()
	nr := int(1)
	i := 0
	for nr > 0 {
		nr, err = s.file.Read(chunkBuf)
		if err != nil && err != io.EOF {
			log.Println("Compute reading :", err)
			return nil, err
		}
		if nr > 0 {
			readedBuf := chunkBuf[0:nr]
			s.adlerHash.Write(readedBuf)
			adlerSum := s.adlerHash.Sum32()
			s.adlerHash.Reset()

			md5Hash.Write(readedBuf)
			md5Sum := md5Hash.Sum(nil)
			csh := CheckSumPack{adlerSum, md5Sum, i}
			chunkHashes = append(chunkHashes, csh)
		}
		i ++
	}
	return chunkHashes,nil
}