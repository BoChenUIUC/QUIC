package main

import (
  "time"
  "fmt"
  "net"
  "io"
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
  go TimestampServer()
  // go InstructionServer()
  // go RunProbeServerS()

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
    connection = wrapper.NewServerWrapper(config.QUICAddr,config.PingSendAddr,config.NACKRecvAddr,
                                          config.DataQUIC,config.PingQUIC,config.XACKQUIC)
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

  var sentIndex int
	for i:=1;i<=numVideoFiles;i++{
		<- t.C
    // if there is an instruction, the sentIndex should
    // be set to i-1
    select{
    case <- instChan:
      sentIndex = i-1
      fmt.Println("Instruction")
    default:
    }
    // check the type
    c, ok := conn.(wrapper.Wrapper)
    if ok==false{
      if mode == config.DumbPrefetch{
        if sentIndex >= numVideoFiles{
          continue
        }
        sendFile(sentIndex+1)
        sentIndex += 1
        if sentIndex<i+prefetchBufSize && sentIndex+1<=numVideoFiles{
          sendFile(sentIndex+1)
          sentIndex += 1
        }
        continue
      }else{
        sendFile(i)
      }
      continue
    }
    // if there is delay, prefetch should stop sending new frames
    if c.IsDelay()==false{
      // send one frame
      // if sent the last frame, continue
      if sentIndex >= numVideoFiles{
        continue
      }
  		sendFile(sentIndex+1)
      sentIndex += 1
      // check whether do prefetching
      if mode == config.Prefetch && sentIndex<i+prefetchBufSize && sentIndex+1<=numVideoFiles{
        sendFile(sentIndex+1)
        sentIndex += 1
      }
    }else{
      // if use drop
      if mode == config.Drop{
        sentIndex = i
        if i==numVideoFiles{
        	sendFile(sentIndex)
        }else{
	        fmt.Println("Drop",sentIndex)
        }
      }
    }
  }
}

func sendFile(frameIndex int) {
  startTime := time.Now()
	fileName := fmt.Sprintf(config.FilePath + "%04d.h264", frameIndex)
  filenameChan <- struct{int;string;time.Time}{frameIndex,fileName,startTime}

  s := fmt.Sprintf("\nSent %d at %s\n",frameIndex,startTime.Format(time.StampMicro))
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
    toolbox.WriteTime(conn, time.Now())
  	toolbox.WriteInt64(conn, int64(len(buf)))
  	toolbox.WriteInt64(conn, int64(frameIndex))
  	n,err := conn.Write(buf)
  	toolbox.Check(err)
    frameInfoChan <- struct{int;int64;time.Time}{frameIndex,int64(n),startTime}
    if frameIndex == maxNum{
      break
    }
  }
  fmt.Println("File sender stops.")
  wg.Done()
}

func InstructionServer(){
	fmt.Println("Inst server listening at",config.TCPInstAddr)
	server, err := net.Listen("tcp", config.TCPInstAddr)
	toolbox.Check(err)
	defer server.Close()

	connection, err := server.Accept()
	toolbox.Check(err)
	defer connection.Close()
	fmt.Println("Inst client connected")

  buf := make([]byte,1)
  for{
    _, err := io.ReadFull(connection, buf)
    if err!=nil{
      break
    }
    instChan <- struct{}{}
  }
}

func TimestampServer(){
	fmt.Println("Tstp server listening at",config.TCPTimestampAddr)
	server, err := net.Listen("tcp", config.TCPTimestampAddr)
	toolbox.Check(err)
	defer server.Close()

	connection, err := server.Accept()
	toolbox.Check(err)
	defer connection.Close()
	fmt.Println("Tstp client connected")

  // create a file to store transmission trace
  f, err := os.OpenFile("frame.dat",os.O_CREATE|os.O_WRONLY, 0644)

	for{
		tranEnd := toolbox.ReadTime(connection)
		tzero,_ := time.Parse(time.StampMicro,time.Time{}.Format(time.StampMicro))
		if tranEnd.Sub(tzero)==0{
			break
		}

		tuple := <-frameInfoChan
    frameIndex := tuple.int
		tranStart := tuple.Time
		tranSize := tuple.int64

		tranStart,_ = time.Parse(time.StampMicro,tranStart.Format(time.StampMicro))
		tranStart = tranStart.Add(time.Millisecond*config.ServerTimerAdder)

		elapsed := tranEnd.Sub(tranStart).Seconds()
		throughput := float64(tranSize)/elapsed/1000000*8

		tran_ack,_ := time.Parse(time.StampMicro,time.Now().Add(time.Millisecond*config.ServerTimerAdder).Format(time.StampMicro))
		uplink_time := tran_ack.Sub(tranEnd).Seconds()

		s := fmt.Sprintf("%d\t%dbytes\t%1.3fMbps\t%fs\t%fs",frameIndex,tranSize,throughput,elapsed,uplink_time)
		fmt.Printf("\n                                          Analyze frame #%s\n",s)

    zero_time,_ := time.Parse(time.StampMicro,time.Time{}.Format(time.StampMicro))
    milli := tranStart.Sub(zero_time).Seconds()
		s = fmt.Sprintf("%f\t%f\n",milli,elapsed)
    f.Write([]byte(s))
	}
}

func RunProbeServer(){
  fmt.Println("Probe server listening at",config.ProbeAddr)
	listener, err := quic.ListenAddr(config.ProbeAddr, toolbox.GenerateTLSConfig(), nil)
	toolbox.Check(err)
	defer listener.Close()
	sess, err := listener.Accept(context.Background())
	toolbox.Check(err)
	defer sess.Close()
	conn, err := sess.AcceptStream(context.Background())
	toolbox.Check(err)
	defer conn.Close()
	fmt.Println("Probe server is up")

	d := time.Duration(time.Millisecond*config.ProbeInterval)
	t := time.NewTicker(d)
	defer t.Stop()
	for{
		<-t.C
		tmp := time.Now().Format(time.StampMicro)
		_,err := conn.Write([]byte(tmp))
		toolbox.Check(err)
	}
}
