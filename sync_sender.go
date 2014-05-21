package gfilesyncer
import (
	"fmt"
	"path"
	"os"
	"net"
	"bufio"
	"bytes"
	"encoding/binary"
)

type SenderConfig struct {
	SrvAddr string
	SyncRootFolder string // sync root path
}

type SyncSender struct {
	cfg *SenderConfig
}

func NewSyncSender(cfg *SenderConfig) *SyncSender {
	return new(SyncSender).Init(cfg)
}

func (s *SyncSender) Init(cfg *SenderConfig) *SyncSender {
	s.cfg = cfg
	return s
}

func (s *SyncSender) Start() error {
	conn, err := net.Dial("tcp", s.cfg.SrvAddr)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Sender:%v\n", err.Error())
		return err
	}
	defer conn.Close()

	return nil
}


func (s *SyncSender) SyncAFile(filePath string) error {
	var err error
	conn, err := net.Dial("tcp", s.cfg.SrvAddr)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Sender:%v\n", err.Error())
		return err
	}
	defer conn.Close()

	fmt.Println("Start sync ",filePath)

	filename := path.Join(s.cfg.SyncRootFolder, filePath)
	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Sender:%v\n", err.Error())
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		fmt.Fprintf(os.Stdout, "Sender:%v\n", err.Error())
		return err
	}

	syncHeader := new(SyncHeader)

	syncHeader.Version = 0x01
	syncHeader.PackageType = PackageTypeSyncRequest
	syncHeader.ChunkSize = DefaultChunkSize
	syncHeader.FileLength = fi.Size()
	syncHeader.FileModTime = fi.ModTime()
	syncHeader.FileName = filePath

	b := syncHeader.toBytes()

	_, err = conn.Write(b)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Sender:%v\n", err.Error())
		return err
	}

	r := bufio.NewReader(conn)
	//recv server reply
	replyCode,_ := r.ReadByte()
	fmt.Printf("server reply:%x\n",replyCode)
	if replyCode == VerifyResultTransferToMe {
		fmt.Println("Sender Recv:VerifyResultTransferToMe")
		fmt.Println("Sender start send file!")
		rbuf := make([]byte, 512)
		var n int
		n = 1
		for n > 0 {
			n, _ = f.Read(rbuf)
			if n > 0 {
				if _, err := conn.Write(rbuf[0:n]); err != nil {
					fmt.Println("Write to server meet err", err)
					return nil
				}
			}
			fmt.Println("client write write:",n)
		}
		replyCode,err = r.ReadByte()
		if err != nil {
			fmt.Println("after transfer got error", err)
			return nil
		}
		fmt.Println("Sender recv ",replyCode)
	} else if replyCode == VerifyResultNeedCompare {
		s.protoCompare(conn, f, syncHeader)
	} else {
		fmt.Printf("Sender not implement replyCode yet:%x\n", replyCode)
	}
	return nil
}

func (s *SyncSender) protoCompare(r net.Conn, f *os.File, syncHeader *SyncHeader) error {
	buf32 := make([]byte, 4)
	if _, err := r.Read(buf32); err != nil {
		return err
	}
	numBufReader := bytes.NewReader(buf32)
	var checksumBlockNum int32
	binary.Read(numBufReader, binary.LittleEndian, &checksumBlockNum)
	checkSumBuf := make([]byte, len(buf32) + int(checksumBlockNum)*(CheckSumPackSize)) //4 alder32, 16 md5

	if _, err := r.Read(checkSumBuf); err != nil {
		return err
	}

	checkSumHashes := NewCheckSumPackArrayFromBytes(checkSumBuf, checksumBlockNum)
	checkSumMap := make(map[uint32]CheckSumPack)
	for _, h := range checkSumHashes {
		checkSumMap[h.Adler32] = h
	}

	//now rolling !
	if _,err := f.Seek(0, 0); err != nil {
		return err
	}

	checker := NewRollingCheckSum(f, syncHeader.ChunkSize)
	err := checker.RollingCheck(checkSumMap)
	if err != nil {
		return err
	}

	return nil
	/*
	csBufReader := bytes.NewReader(checkSumBuf)
	binary.Read()
	csBufReader
	*/
}
