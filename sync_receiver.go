package gfilesyncer

import (
	"log"
	"net"
	"time"
	//"fmt"
	//"io"
	"bufio"
	"path"
	"syscall"
	"os"

)

type ReceiverConfig struct {
	DebugEnable bool
	ListenAddr string	//the port listen for the client-server
	SyncRootFolder string
}


type SyncReceiver struct {
	cfg *ReceiverConfig
	quitCh chan byte
}

func NewSyncReceiver(cfg *ReceiverConfig) *SyncReceiver {
	return new(SyncReceiver).init(cfg)
}

func (s *SyncReceiver) init(cfg *ReceiverConfig) *SyncReceiver {
	s.cfg = cfg
	s.quitCh = make(chan byte)
	return s
}

func (s *SyncReceiver) Start() error {
	laddr, err := net.ResolveTCPAddr("tcp", s.cfg.ListenAddr)
	if err != nil {
		log.Fatalln(err)
		return err
	}
	
	ln, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return err
	}
	defer ln.Close()
	for {
		select {
		case <- s.quitCh:
			log.Println("Receive Quit")
			return nil
		default:
		}
		ln.SetDeadline(time.Now().Add(time.Second*10))	//for the graceful quit
		conn, err := ln.AcceptTCP()
		if err != nil {
			switch e := err.(type) {
			case *net.OpError:
				if e.Timeout() != true {
					log.Fatalln("ln.Accept error:", e)
				}
				continue
			default:
				log.Fatalln("ln.Accept error:", e)
				continue
			}
		}
		go s.HandleConnection(conn)
	}
	return nil
}

func (s *SyncReceiver) HandleConnection(conn net.Conn) {
	var err error
	defer func() {
		log.Println("Receiver: client disconnect with:",err)
		if err != nil {
			conn.Write([]byte{VerifyResultError})
		}
		conn.Close()
	}()
	
	log.Println("New Connection Com")
	br := bufio.NewReader(conn)
	headerLen := syncHeaderLen()
	headerBuf := make([]byte, headerLen)
	_, err = br.Read(headerBuf)
	if err != nil {
		return
	}

	syncHeader := NewSyncHeaderFromBytes(headerBuf)

	log.Println("Read Header Succ", syncHeader.FileLength)
	
	buf_filename, err := br.ReadBytes('\n')
	//log.Println(len(buf_filename), buf_filename)
	if len(buf_filename) <= 1 {
		return
	}
	filename := string(buf_filename[0:len(buf_filename)-1])
	log.Println("filename:",filename)
	filename = path.Join(s.cfg.SyncRootFolder, filename)
	var fi os.FileInfo
	fi, err = os.Stat(filename)
	if os.IsNotExist(err) || fi.IsDir() {
    	log.Printf("no such file: %s\n", filename)
    	conn.Write([]byte{VerifyResultTransferToMe})
    	//NOW Read&Write Entire File
    	
    	rbuf := make([]byte, 512) //readbuf
    	//create temp file
    	tempfilename := filename + ".syncing"
    	tempFile,err := os.Create(tempfilename)
    	if err != nil {
    		log.Println("Create temp file error", err)
    		return
    	}

    	endTransfer := false
    	readedBytes := int64(0)
    	for !endTransfer {
	    	n,err := br.Read(rbuf)
	    	if err != nil {
	    		log.Println("Transfering File Err:", err)
	    		return
	    	}
	    	if n > 0 {
				if nw, err := tempFile.Write(rbuf[0:n]); err != nil || nw != n {
	    			log.Println("Write tempfile error")
	    			return
	    		}
	    	}
	    	readedBytes += int64(n)
	    	if readedBytes > syncHeader.FileLength {
	    		log.Println("Transfer more than syncheader:",readedBytes,syncHeader.FileLength)
	    		return
	    	} else if readedBytes == syncHeader.FileLength {
	    		endTransfer = true
	    	}
	    	log.Println("recv:", n)
    	}
    	if err := tempFile.Close(); err != nil {
    		log.Println("Sync&Close temp file failed!")
    		return
    	} //
    	log.Println("file recv over")
    	//rename file & give a new mod time
    	os.Rename(tempfilename, filename)
    	fi,_ = os.Stat(filename)
    	accessTimeval := new(syscall.Timeval)
    	accessTimeval.Sec = fi.ModTime().Unix()
    	modTimeval := new(syscall.Timeval)
    	modTimeval.Sec = syncHeader.FileModTime.Unix()
    	err = syscall.Utimes(filename, []syscall.Timeval{*accessTimeval,*modTimeval})
    	if err != nil {
    		log.Println("utime error:", err)
    		return
    	}
    	err = nil
    	conn.Write([]byte{VerifyResultDoNext})
    	log.Println("sync over!!!")
	} else {
		if syncHeader.FileLength == fi.Size() && syncHeader.FileModTime.Equal(fi.ModTime()) {
			log.Println("Timestamp & Size Equal. This file is synced")
			conn.Write([]byte{VerifyResultDoNext})
			//TBD continue the loop
		} else {
			log.Println("Modtime not equal. So need file compare")
			//Send need compare flag
			conn.Write([]byte{VerifyResultNeedCompare})
			//Compute & Send chunkchecksum
			// syncHeader.FileLength
			
			f, err := os.Open(filename)
			if err != nil {
				log.Println("open file err:", err)
    			return
			}
			checkSum := NewRollingCheckSum(f, syncHeader.ChunkSize)
			hashes, err := checkSum.SumEveryChunk()
			if err != nil {
				log.Println("error when generate checksum", err)
				return
			}
			
			checksumBytes := CheckSumPackArrayToBytes(hashes)
			conn.Write(checksumBytes)
			
			//adler+md5
			//Recv TransferData Packages
		}
	}

/*
	version := br.ReadByte()
	packageType  := br.ReadByte()
	if !(version == 0x01 && packageType == PackageTypeSyncRequest) {
		
		return
	}
	return
	temp2 := make([]byte, 2)
	br.Read(temp)
	//little endian
	packageLen := uint16(temp2[0])|(uint16(temp2[1])<<8)

	io.WriteString(conn, "Hi")
	*/
}


func (s *SyncReceiver) Stop() {
	//s.quitCh <- 1
	close(s.quitCh)
}