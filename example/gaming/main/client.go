package main

import (
  "fmt"
  "net"
  "io"
  "os"
  "time"
  "sync"
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

var recvtimeChan chan time.Time
var instChan chan struct{}
var totalProbeLatency float64
var numProbes int64
var numAbnormalProbes10 int64
var numAbnormalProbes05 int64
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
    connection = wrapper.NewClientWrapper(config.QUICAddr,config.PingSendAddr,config.NACKRecvAddr,
                                          config.DataQUIC,config.PingQUIC,config.XACKQUIC)
  }

  wg := sync.WaitGroup{}
	wg.Add(1)

  recvtimeChan = make(chan time.Time,1000)
  instChan = make(chan struct{},1000)
  go TimestampClient(&wg)
  // go InstructionClient()
  // go RunProbeClient()

  // for evaluation purpose
  var initTime time.Time
  recvtimeArr := make([]float64,numVideoFiles)
  var totalSize int64

  frame_sent_time_f, err := os.OpenFile("frame_sent_time.dat",os.O_CREATE|os.O_WRONLY, 0644)
  cnt := 0
  for {
    // store sent time
    sTime := toolbox.ReadTime(connection)
    sTime = sTime.Add(time.Millisecond*config.ServerTimerAdder)
    rTime,_ := time.Parse(time.StampMicro,time.Now().Format(time.StampMicro))
    zTime,_ := time.Parse(time.StampMicro,time.Time{}.Format(time.StampMicro))
    s := fmt.Sprintf("%f\t%f\n",rTime.Sub(zTime).Seconds(),sTime.Sub(zTime).Seconds())
    frame_sent_time_f.Write([]byte(s))
      // receive a frame
		fileSize,frameIndex,err := recvSingleFile(connection)

    if int(frameIndex) > numVideoFiles{
      continue
    }
    // send receiving time back
    recvtime := time.Now()
    recvtimeChan <- recvtime
    // write
    fmt.Printf("\nRender%d\t%s\n",frameIndex,recvtime.Format(time.StampMicro))
    // store received frame
    if initTime.IsZero(){
      initTime = recvtime
    }
    recvtimeArr[int(frameIndex)-1] = recvtime.Sub(initTime).Seconds()
    totalSize += fileSize
    if err!=nil{
      fmt.Println("ERR!")
      break
    }
    // check if got the final frame
    if frameIndex == int64(numVideoFiles){
      break
    }
    // put some instruction
    r := rand.Intn(10)
    if r==0{
      instChan <- struct{}{}
    }
    cnt += 1
  }
  recvtimeChan <- time.Time{}
  wg.Wait()

  // summarize
  fmt.Println("Total transmission size:",totalSize)
}

func AnalyzeRecvtime(arr []float64,m string,tolerance float64)(int,float64){
  f, err := os.OpenFile(m+".txt",os.O_CREATE|os.O_WRONLY, 0644)
  if err!=nil{
    panic(err)
  }
  numMiss := 0
  numArrive := 0
  var totalDelay float64
  for i:=0;i<len(arr);i++{
    d := arr[i]-tolerance-0.033*float64(i)
    s := fmt.Sprintf("%d\t%.3f\t%.3f\n",i,arr[i],d)
    f.Write([]byte(s))
    if (arr[i]==0&&i>0) || d>0{
      numMiss += 1
      if d>0{
        totalDelay += d
      }
    }
    if i==0||(i>0&&arr[i]>0){
      numArrive += 1
    }
  }
  avgDelay := totalDelay/float64(numArrive)
  fmt.Println("Num misses:",numMiss,"avg delay",avgDelay,totalDelay)
  return numMiss,avgDelay
}

func recvSingleFile(connection protocol.Conn) (int64,int64,error){
	fileSize := toolbox.ReadInt64(connection)
	// if size < 0, then nothing to receive
	if fileSize < 0{
		return int64(fileSize),int64(0),nil
	}
	// read frame index
	frameIndex := toolbox.ReadInt64(connection)

	buf := make([]byte, fileSize)
	_, err := io.ReadFull(connection, buf)

	// go func(){
	// 	// open output file
	// 	tmpName := fmt.Sprintf("%04d.h264", frameIndex)
	//   fo, err := os.Create(tmpName)
	//   toolbox.Check(err)
  //
	//   _, err = fo.Write(buf)
	// 	toolbox.Check(err)
  //
	// 	fileName := fmt.Sprintf(config.FilePath+"%04d.h264", frameIndex)
	// 	cmd := exec.Command("diff", tmpName, fileName)
	// 	_, err = cmd.CombinedOutput()
	// 	if err!=nil{
	// 		fmt.Println("FILE ERROR!!!")
	// 	}
	// 	toolbox.Check(fo.Close())
	// 	toolbox.Check(os.Remove(tmpName))
	// }()
	return fileSize,frameIndex,err
}

func TimestampClient(wg *sync.WaitGroup){
	connection, err := net.Dial("tcp", config.TCPTimestampAddr)
	toolbox.Check(err)
	fmt.Println("Tstp sender created")
	for{
		recv_time:= <-recvtimeChan
		err = toolbox.WriteTime(connection, recv_time)
    if err!=nil{
      break
    }
		if recv_time.IsZero(){
			break
		}
	}
	wg.Done()
}

func InstructionClient(){
	connection, err := net.Dial("tcp", config.TCPInstAddr)
	toolbox.Check(err)
	fmt.Println("Inst sender created")
	for{
		<-instChan
    _,err := connection.Write([]byte("O"))
		if err!=nil{
      break
    }
	}
}

func RunProbeClient(){
  tlsConf := &tls.Config{
    InsecureSkipVerify: true,
    NextProtos:         []string{"quic-echo-example"},
  }
  session, err := quic.DialAddr(config.ProbeAddr, tlsConf, nil)
	toolbox.Check(err)
	conn, err := session.OpenStreamSync(context.Background())
	toolbox.Check(err)
  conn.Write([]byte("A"))

	for{
		buf := make([]byte, 22)
		_, err := io.ReadFull(conn, buf)
		toolbox.Check(err)
		sent_time,err := time.Parse(time.StampMicro,string(buf))
		if err!=nil{
			continue
		}

		echo_time,_ := time.Parse(time.StampMicro,time.Now().Format(time.StampMicro))
		downlink_time := echo_time.Sub(sent_time.Add(time.Millisecond*config.ServerTimerAdder)).Seconds()
    fmt.Println(downlink_time)
    numProbes += 1
    totalProbeLatency += downlink_time
    if downlink_time > 0.1{
      numAbnormalProbes10 += 1
    }
    if downlink_time > 0.05{
      numAbnormalProbes05 += 1
    }
	}
}
