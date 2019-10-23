package main

import (
  "time"
  "fmt"
  "net"
  // "io"
  "os"
  "sync"
  "strconv"
  "context"
	"io/ioutil"
	"github.com/lucas-clemente/quic-go/example/gaming/wrapper"
	"github.com/lucas-clemente/quic-go/example/gaming/config"
	"github.com/lucas-clemente/quic-go/example/gaming/protocol"
	"github.com/lucas-clemente/quic-go/example/gaming/toolbox"
  quic "github.com/lucas-clemente/quic-go"
)

var frameInfoChan chan struct {int;int64;time.Time}
var instChan chan struct {}
var app int
var numVideoFiles int
var prefetchBufSize int
var filenameChan chan struct {int;string;time.Time}

func main(){
  app,_ = strconv.Atoi(os.Args[1])
  numVideoFiles,_ = strconv.Atoi(os.Args[2])
  // choose protocol and application layer mode
  proto := config.QUIC
  mode := app
  if app == config.Drop || app == config.Prefetch{
    proto = config.NEWPROTO
  }
  if app == config.Prefetch || app == config.DumbPrefetch{
    prefetchBufSize,_ = strconv.Atoi(os.Args[3])
  }

  frameInfoChan = make(chan struct {int;int64;time.Time},1000)
  instChan = make(chan struct{}, 1000)
  filenameChan = make(chan struct {int;string;time.Time},1000)

  // create connection
  var connection protocol.Conn
	if proto == config.TCP{
		fmt.Println("Tcp server listening at",config.TCPAddr)
		server, err := net.Listen("tcp", config.TCPAddr)
		toolbox.Check(err)
		defer server.Close()
		connection, err = server.Accept()
		toolbox.Check(err)
		defer connection.Close()
		fmt.Println("Tcp open")
	}else if proto == config.QUIC{
		fmt.Println("QUIC server listening at",config.QUICAddr)
		listener, err := quic.ListenAddr(config.QUICAddr, toolbox.GenerateTLSConfig(), nil)
		toolbox.Check(err)
		defer listener.Close()
		sess, err := listener.Accept(context.Background())
		toolbox.Check(err)
		defer sess.Close()
		connection, err = sess.AcceptStream(context.Background())
		toolbox.Check(err)
		defer connection.Close()
		fmt.Println("QUIC open")
	}else if proto == config.NEWPROTO{
    connection = wrapper.NewServerWrapper(config.QUICAddr,config.PingSendAddr,config.NACKRecvAddr,config.TimestampAddr,
                                          config.DataQUIC,config.PingQUIC,config.XACKQUIC,numVideoFiles)
  }
  wg := sync.WaitGroup{}
	wg.Add(1)
  go fileSender(connection,numVideoFiles,&wg)
  ServerStreaming(connection,mode)
  wg.Wait()
}

func ServerStreaming(conn protocol.Conn,mode int){
  // set timer
	d := time.Duration(time.Millisecond*33)
	t := time.NewTicker(d)
  defer t.Stop()

	for i:=1;i<=numVideoFiles;i++{
		<- t.C
    // check the type
    c, ok := conn.(wrapper.Wrapper)
    if ok==false{
      sendFile(i)
      continue
    }
    // flush all received xack information
    // make some decisions based on xack info
    c.GetNetStat()
    // print send index
    fp, _ := os.OpenFile("trace.dat",os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    s := "[SEND]\n"
    fp.Write([]byte(s))
    fp.Close()
    // if there is delay
    sendFile(i)
  }
  c, ok := conn.(wrapper.Wrapper)
  if ok{
    c.Wait()
  }
}

func sendFile(frameIndex int) {
  startTime := time.Now()
	fileName := fmt.Sprintf(config.FilePath + "%04d.h264", frameIndex)
  filenameChan <- struct{int;string;time.Time}{frameIndex,fileName,startTime}

  s := fmt.Sprintf("Sent %d at %s\n",frameIndex,startTime.Format(time.StampMicro))
  fmt.Printf(s)
}

func fileSender(conn protocol.Conn, maxNum int, wg *sync.WaitGroup){
  fmt.Println("File sender started")
  for{
    tuple := <-filenameChan
    frameIndex := tuple.int
    filename := tuple.string
    startTime := tuple.Time
    buf, err := ioutil.ReadFile(filename)
  	toolbox.Check(err)
    // toolbox.WriteTime(conn, time.Now())
  	toolbox.WriteInt64(conn, int64(len(buf)))
  	conn.Write(buf)
    frameInfoChan <- struct{int;int64;time.Time}{frameIndex,int64(len(buf)),startTime}
    if frameIndex == maxNum{
      break
    }
  }
  fmt.Println("File sender stops.")
  wg.Done()
}
