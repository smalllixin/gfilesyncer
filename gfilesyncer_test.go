package gfilesyncer
import (
	"code.google.com/p/go.crypto/md4"
	"io"
	"fmt"
	"testing"
	//"path"
	//"time"
	"sync"
	"io/ioutil"
	"os"
	"reflect"
	//"path"
	"crypto/md5"
)

func TestSplit(t *testing.T) {
	s := uint32(0xFFFFFFFF)
	fmt.Println("uint32:", s)
}

func TestRollingsum(t *testing.T) {
	chunkStr := "ABCDEFGasdfj1;u391203819023dmsfla"
	cs := []byte(chunkStr)

	h := NewAdler32()
	fmt.Println("CHECKSUMFOR:", chunkStr)
	h.Write([]byte(chunkStr))
	sumv := h.Sum32()
	fmt.Printf("%x\n", sumv)

	h.Reset()
	beforeRotate := "=" + string(cs[0:len(cs)-1])
	fmt.Println("CHECKSUMFOR:", beforeRotate)
	h.Write([]byte(beforeRotate))
	sumv_mid := h.Sum32()
	fmt.Printf("%x\n", sumv_mid)

	fmt.Println("Rotate:",string(cs[len(cs)-1]))
	h.Rotate('=', cs[len(cs)-1], len(cs))
	sumv_rotate := h.Sum32()
	fmt.Printf("%x\n", sumv_rotate)
	if sumv != sumv_rotate {
		t.Fatal("adler32 roate failed")
	}
}

func TestCompareMd5Value(t *testing.T) {
	h := md5.New()
	io.WriteString(h, "The fog is getting thicker!")
	v1 := h.Sum(nil)
	h.Reset()
	
	v2 := make([]byte, 16)
	copy(v2, v1)

	h.Reset()
	io.WriteString(h, "Something diff")
	v3 := h.Sum(nil)
	if !compareMd5Value(v1, v2) {
		t.Fatal("compareMd5Value", v1, v2)
	}

	if compareMd5Value(v1, v3) {
		t.Fatal("compareMd5Value", v1, v3)
	}
}

func TestChunkChecksum(t *testing.T) {
	f, err := os.Open("./README.md")
	if err != nil {
		t.Fatal(err);return
	}

	rc := NewRollingCheckSum(f, 16)
	hashes, err := rc.SumEveryChunk()
	if err != nil {
		t.Fatal(err);return
	}
	// for _, x := range hashes {
	// 	fmt.Printf("% x | %x\n", x.Adler32,x.Md5)
	// }

	checkbytes := CheckSumPackArrayToBytes(hashes)
	// fmt.Printf("CheckBytes %d:  %x\n", len(checkbytes), checkbytes)
	dataPart := checkbytes[4:]
	decodedHashes := NewCheckSumPackArrayFromBytes(dataPart, int32(len(hashes)))

	// for _, x := range decodedHashes {
	// 	fmt.Printf("% x | %x\n", x.Adler32,x.Md5)
	// }
	if reflect.DeepEqual(hashes, decodedHashes) {

	} else {
		t.Fatal("hash decoded not right")
	}
	
}

func TestRunReceiver(t *testing.T) {
	return
	var wg sync.WaitGroup


	c := md4.New()
	io.WriteString(c, "hello world")
	s := fmt.Sprintf("%x", c.Sum(nil))
	fmt.Println(s)
	
	wg.Add(1)

	var srvCfg ReceiverConfig
	srvCfg.ListenAddr = "0.0.0.0:9999"
	srvCfg.SyncRootFolder = "/Users/smalllixin/Documents/lightchaser/temp"
	syncReceiver := NewSyncReceiver(&srvCfg)
	go func() {
		syncReceiver.Start()
	}()
	
	go func() {
		defer func() {
			fmt.Println("client defer run")
			syncReceiver.Stop()
			wg.Done()
		}()

		var clientCfg SenderConfig
		clientCfg.SrvAddr = "127.0.0.1:9999"
		clientCfg.SyncRootFolder = "/Users/smalllixin/Documents/pyspace"
		client := NewSyncSender(&clientCfg)
		
		files, err := ioutil.ReadDir(clientCfg.SyncRootFolder)
		if err != nil {
			t.Fatal("ReadDir err:", err)
			return
		}
		for _, f := range files {
			if !f.IsDir() {
				fileToSync := f.Name()//path.Join(clientCfg.SyncRootFolder, )
				client.SyncAFile(fileToSync)
			}
		}

		//client.SyncAFile("udpclient.py")
	}()

	wg.Wait()
	//time.Sleep()
}