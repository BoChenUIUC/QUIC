package main

import (
  "fmt"
  "net"
  // "io"
  "os"
  // "time"
  // "sync"
  "strconv"
  // "os/exec"
	"crypto/tls"
  "math/rand"
  "context"
	"github.com/lucas-clemente/quic-go/example/gaming/wrapper"
	"github.com/lucas-clemente/quic-go/example/gaming/config"
	"github.com/lucas-clemente/quic-go/example/gaming/protocol"
	"github.com/lucas-clemente/quic-go/example/gaming/toolbox"
  quic "github.com/lucas-clemente/quic-go"
)

var numProbes int64
var app int
var numVideoFiles int

func main(){
  app,_ = strconv.Atoi(os.Args[1])
  numVideoFiles,_ = strconv.Atoi(os.Args[2])
  rand.Seed(1)
  var connection protocol.Conn
	var err error
  proto := config.NEWPROTO
  if app==config.Default||app==config.DumbPrefetch{
    proto = config.QUIC
  }
	if proto == config.TCP{
		fmt.Println("TCP Trying to dial",config.TCPAddr)
		connection, err = net.Dial("tcp", config.TCPAddr)
		toolbox.Check(err)
		fmt.Println("TCP connection established")
	}else if proto == config.QUIC{
		fmt.Println("QUIC Trying to dial",config.QUICAddr)
    tlsConf := &tls.Config{
      InsecureSkipVerify: true,
      NextProtos:         []string{"quic-echo-example"},
    }
		session, err := quic.DialAddr(config.QUICAddr, tlsConf, nil)
		toolbox.Check(err)
		connection, err = session.OpenStreamSync(context.Background())
		toolbox.Check(err)
    connection.Write([]byte("A"))
		fmt.Println("QUIC connection established")
	}else if proto == config.NEWPROTO{
    connection = wrapper.NewClientWrapper(config.QUICAddr,config.PingSendAddr,config.NACKRecvAddr,config.TimestampAddr,
                                          config.DataQUIC,config.PingQUIC,config.XACKQUIC,numVideoFiles)
  }

  // for evaluation purpose
  var totalSize int64

  for frameIndex:=1;frameIndex<=numVideoFiles;frameIndex++{
    // receive a frame
		fileSize,err := recvSingleFile(connection)
    if err!=nil{
      fmt.Println("ERR!")
      break
    }
    // write
    fmt.Printf("\nRender%d\n",frameIndex)
    totalSize += fileSize
  }
  c, ok := connection.(wrapper.Wrapper)
  if ok{
    c.Wait()
  }

  // summarize
  fmt.Println("Total transmission size:",totalSize)
}

func recvSingleFile(connection protocol.Conn) (int64,error){
	fileSize,err := toolbox.ReadInt64(connection)
  if err!=nil{panic(err)}
	buf := make([]byte, fileSize)
  _, err = connection.Read(buf)

	return fileSize,err
}
