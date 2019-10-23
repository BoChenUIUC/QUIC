package wrapper

import (
	"fmt"
	"io"
	"os"
	"time"
	"strconv"
  "crypto/tls"
	"sync"
	"net"
	"context"
	// "github.com/gonum/stat"
	"github.com/lucas-clemente/quic-go/example/gaming/protocol"
	"github.com/lucas-clemente/quic-go/example/gaming/config"
	"github.com/lucas-clemente/quic-go/example/gaming/toolbox"
  quic "github.com/lucas-clemente/quic-go"
)

type Wrapper struct {
	data_conn protocol.Conn
  ping_conn protocol.Conn
  nack_conn protocol.Conn
	tstp_conn protocol.Conn

	pingLatencyChan chan float64
	pingSentTimeChan chan float64
	frameLatencyChan chan struct {int;float64}
	frameSentTimeChan chan float64
	xackBufChan chan string
	timeChan chan struct {int;time.Time}
	tstpSigChan chan struct {}
	numVideoFiles int
	firstFrameSentTime time.Time
	firstPingSentTime time.Time
}

func NewServerWrapper(addr_data,addr_ping,addr_nack,addr_tstp string,dataquic,pingquic,xackquic bool,numvideo int)Wrapper{
	// write first lines
	fp, _ := os.OpenFile("trace.dat",os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer fp.Close()
	pingInterval,_ := strconv.Atoi(os.Args[3])
	xackInterval,_ := strconv.Atoi(os.Args[4])
	rep,_ := strconv.Atoi(os.Args[5])
	s := fmt.Sprintf("S\t%d\t%d\t%d\n",pingInterval,xackInterval,rep)
	fp.Write([]byte(s))
	w := Wrapper{}
	wg := sync.WaitGroup{}
	wg.Add(4)
	go func(wg *sync.WaitGroup){
		w.tstp_conn = GetTCPServerConn(addr_tstp)
		wg.Done()
	}(&wg)
	go func(wg *sync.WaitGroup){
		if dataquic{
			w.data_conn = GetQUICServerConn(addr_data)
		}else{
			w.data_conn = GetTCPServerConn(addr_data)
		}
		wg.Done()
	}(&wg)
	go func(wg *sync.WaitGroup){
		if pingquic{
			w.ping_conn = GetQUICServerConn(addr_ping)
		}else{
			w.ping_conn = GetTCPServerConn(addr_ping)
		}
		wg.Done()
	}(&wg)
	go func(wg *sync.WaitGroup){
		if xackquic{
			w.nack_conn = GetQUICServerConn(addr_nack)
		}else{
			w.nack_conn = GetTCPServerConn(addr_nack)
		}
		wg.Done()
	}(&wg)
  wg.Wait()
  fmt.Println("server connected")
	w.xackBufChan = make(chan string,1000)
	w.timeChan = make(chan struct {int;time.Time},1000)
	w.tstpSigChan = make(chan struct {},1)
	w.numVideoFiles = numvideo
	// start ping and nack
	go w.RunTstpServer()
	go w.RunPingServer()
	go w.RunXACKServer()
  return w
}

func NewClientWrapper(addr_data,addr_ping,addr_nack,addr_tstp string,dataquic,pingquic,xackquic bool, numvideo int)Wrapper{
	// write initial lines to file
	fp, _ := os.OpenFile("latency.dat",os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer fp.Close()
	pingInterval,_ := strconv.Atoi(os.Args[3])
	xackInterval,_ := strconv.Atoi(os.Args[4])
	rep,_ := strconv.Atoi(os.Args[5])
	s := fmt.Sprintf("S\t%d\t%d\t%d\n",pingInterval,xackInterval,rep)
	fp.Write([]byte(s))
	// construct wrapper
	w := Wrapper{}
	w.tstp_conn = GetTCPClientConn(addr_tstp)
	if dataquic{
		w.data_conn = GetQUICClientConn(addr_data)
	}else{
		w.data_conn = GetTCPClientConn(addr_data)
	}
	if pingquic{
		w.ping_conn = GetQUICClientConn(addr_ping)
	}else{
		w.ping_conn = GetTCPClientConn(addr_ping)
	}
	if xackquic{
		w.nack_conn = GetQUICClientConn(addr_nack)
	}else{
		w.nack_conn = GetTCPClientConn(addr_nack)
	}
	w.pingLatencyChan = make(chan float64,1000)
	w.pingSentTimeChan = make(chan float64,1000)
	w.frameLatencyChan = make(chan struct {int;float64},1000)
	w.frameSentTimeChan = make(chan float64,1000)
	w.timeChan = make(chan struct {int;time.Time},1000)
	w.tstpSigChan = make(chan struct {},1)
	w.numVideoFiles = numvideo
	w.firstFrameSentTime = time.Time{}
	w.firstPingSentTime = time.Time{}

	go w.RunTstpClient()
	go w.RunPingClient()
	go w.RunXACKClient()
  return w
}

func GetQUICServerConn(addr string)protocol.Conn{
	fmt.Println("QUIC server listening at",addr)
	listener, err := quic.ListenAddr(addr, toolbox.GenerateTLSConfig(), nil)
	toolbox.Check(err)
	sess, err := listener.Accept(context.Background())
	toolbox.Check(err)
	conn, err := sess.AcceptStream(context.Background())
	toolbox.Check(err)
	buf := make([]byte, 1)
	_, err = io.ReadFull(conn, buf)
	toolbox.Check(err)
	fmt.Println("QUIC server open at",addr)
	return conn
}

func GetQUICClientConn(addr string)protocol.Conn{
	fmt.Println("QUIC Trying to dial",addr)
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}
	session, err := quic.DialAddr(addr, tlsConf, nil)
	toolbox.Check(err)
	conn, err := session.OpenStreamSync(context.Background())
	toolbox.Check(err)
	conn.Write([]byte("A"))
	fmt.Println("connected to",addr)
	return conn
}

func GetTCPServerConn(addr string)protocol.Conn{
	fmt.Println("Tcp server listening at",addr)
	server, err := net.Listen("tcp", addr)
	toolbox.Check(err)
	conn, err := server.Accept()
	toolbox.Check(err)
	fmt.Println("connected to",addr)
	return conn
}

func GetTCPClientConn(addr string)protocol.Conn{
	d := time.Duration(time.Millisecond*500)
	t := time.NewTicker(d)
  defer t.Stop()

	fmt.Println("TCP Trying to dial",addr)
	for{
		<- t.C
		conn, err := net.Dial("tcp", addr)
		if err!=nil{
			fmt.Println("TCP dial failure, try again.")
		}else{
			fmt.Println("connected to",addr)
			return conn
		}
	}
	return nil
}

func (w Wrapper) Write(p []byte) (int,error){
  n,err := w.data_conn.Write(p)
	if n>10{
		w.timeChan <- struct{int;time.Time}{n,time.Now()}
	}
  return n,err
}

func (w Wrapper) Read(p []byte) (int,error)  {
	// n,err := w.data_conn.Read(p)
	n, err := io.ReadFull(w.data_conn, p)
	if n>10{
		w.timeChan <- struct{int;time.Time}{n,time.Now()}
	}
	return n,err
}

func (w Wrapper) Close() error {
	return w.data_conn.Close()
}

func (w Wrapper) Wait(){
	<-w.tstpSigChan
}

func (w Wrapper) RunTstpClient(){
	fp, _ := os.OpenFile("latency.dat",os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer fp.Close()
	for i:=1;i<=w.numVideoFiles;i++{
		tranStart,err := toolbox.ReadTime(w.tstp_conn)
		if err!=nil{
			panic(err)
		}
		tranStart = tranStart.Add(time.Millisecond*config.ServerTimerAdder)
		if w.firstFrameSentTime.IsZero(){
			w.firstFrameSentTime = tranStart
		}
		tuple := <-w.timeChan
		tranSize := tuple.int
		tranEnd := tuple.Time
		tranEnd,_ = time.Parse(time.StampMicro,tranEnd.Format(time.StampMicro))

		elapsed := tranEnd.Sub(tranStart).Seconds()
		throughput := float64(tranSize)/elapsed/1000000*8
		w.frameLatencyChan <- struct{int;float64}{tranSize,elapsed}
		w.frameSentTimeChan <- tranStart.Sub(w.firstFrameSentTime).Seconds()

		s := fmt.Sprintf("%d\t%dbytes\t%1.3fMbps\t%fs",i,tranSize,throughput,elapsed)
		fmt.Printf("             Analyze frame #%s\n",s)

		s = fmt.Sprintf("%f\n",elapsed)
		fp.Write([]byte(s))
	}
	w.tstp_conn.Close()
	w.tstpSigChan <- struct {}{}
}

func (w Wrapper) RunTstpServer(){
	for i:=1;i<=w.numVideoFiles;i++{
		tuple := <-w.timeChan
		tranStart := tuple.Time
		err := toolbox.WriteTime(w.tstp_conn, tranStart)
    if err!=nil{
      panic(err)
    }
	}
	w.tstp_conn.Close()
	w.tstpSigChan <- struct {}{}
}

func (w Wrapper) RunPingServer(){
	pingInterval,_ := strconv.Atoi(os.Args[3])
	d := time.Millisecond*time.Duration(pingInterval)
	t := time.NewTicker(d)
	defer t.Stop()
	for{
		<-t.C
		tmp := time.Now().Format(time.StampMicro)
		_,err := w.ping_conn.Write([]byte(tmp))
		if err!=nil{break}
	}
}



func (w Wrapper) RunPingClient(){
	for{
		buf := make([]byte, 22)
		_, err := io.ReadFull(w.ping_conn, buf)
		if err!=nil{break}
		sent_time,err := time.Parse(time.StampMicro,string(buf))
		if err!=nil{break}
		sent_time = sent_time.Add(time.Millisecond*config.ServerTimerAdder)
		if w.firstPingSentTime.IsZero(){
			w.firstPingSentTime = sent_time
		}

		echo_time,_ := time.Parse(time.StampMicro,time.Now().Format(time.StampMicro))
		probe_latency := echo_time.Sub(sent_time).Seconds()
		w.pingLatencyChan <- probe_latency
		w.pingSentTimeChan <- sent_time.Sub(w.firstPingSentTime).Seconds()
	}
}

func (w Wrapper) RunXACKServer(){
	for {
		buf := "[XACK]\n"
		l,err := toolbox.ReadInt64(w.nack_conn)
		if err!=nil{return}
		if l>0{
			buf += fmt.Sprintf("[PROBE]%d\n",l)
			for i:=0;i<int(l);i++{
				probeLatency := toolbox.ReadFloat64(w.nack_conn)
				sentSinceEpoch := toolbox.ReadFloat64(w.nack_conn)
				buf = fmt.Sprintf("%s%f\t%f\n",buf,probeLatency,sentSinceEpoch)
			}
		}
		l,err = toolbox.ReadInt64(w.nack_conn)
		if err!=nil{return}
		if l>0{
			buf += fmt.Sprintf("[FRAME]%d\n",l)
			for i:=0;i<int(l);i++{
				size,err := toolbox.ReadInt64(w.nack_conn)
				if err!=nil{return}
				frameLatency := toolbox.ReadFloat64(w.nack_conn)
				sentSinceEpoch := toolbox.ReadFloat64(w.nack_conn)
				buf = fmt.Sprintf("%s%d\t%f\t%f\n",buf,size,frameLatency,sentSinceEpoch)
			}
		}
		w.xackBufChan <- buf
	}
}

func (w Wrapper) RunXACKClient(){
	xackInterval,_ := strconv.Atoi(os.Args[4])
	var nackTicker *time.Ticker
	d := time.Millisecond*time.Duration(xackInterval)
	nackTicker = time.NewTicker(d)
	for{
		<-nackTicker.C
		// probe Information
		l := len(w.pingLatencyChan)
		err := toolbox.WriteInt64(w.nack_conn,int64(l))
		if err!=nil{
			return
		}
		for i:=0;i<l;i++{
			downlink_time := <- w.pingLatencyChan
			sentSinceEpoch := <- w.pingSentTimeChan
			err = toolbox.WriteFloat64(w.nack_conn,downlink_time)
			if err!=nil{return}
			err = toolbox.WriteFloat64(w.nack_conn,sentSinceEpoch)
			if err!=nil{return}
		}
		// frame information
		l = len(w.frameLatencyChan)
		err = toolbox.WriteInt64(w.nack_conn,int64(l))
		if err!=nil{
			return
		}
		for i:=0;i<l;i++{
			tuple := <-w.frameLatencyChan
			frameLatency := tuple.float64
			size := tuple.int
			sentSinceEpoch := <-w.frameSentTimeChan
			err = toolbox.WriteInt64(w.nack_conn,int64(size))
			if err!=nil{return}
			err = toolbox.WriteFloat64(w.nack_conn,frameLatency)
			if err!=nil{return}
			err = toolbox.WriteFloat64(w.nack_conn,sentSinceEpoch)
			if err!=nil{return}
		}
	}
}

func (w Wrapper) GetNetStat()int{
	fp, _ := os.OpenFile("trace.dat",os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer fp.Close()
	l := len(w.xackBufChan)
	for i:=0;i<l;i++{
		s := <-w.xackBufChan
    fp.Write([]byte(s))
	}
	os.Stdout.Sync()
	return 0
}
