package wrapper

import (
	"fmt"
	"io"
	"os"
	"time"
  "crypto/tls"
	"sync"
	"net"
	"context"
	"github.com/gonum/stat"
	"github.com/lucas-clemente/quic-go/example/gaming/protocol"
	"github.com/lucas-clemente/quic-go/example/gaming/config"
	"github.com/lucas-clemente/quic-go/example/gaming/toolbox"
  quic "github.com/lucas-clemente/quic-go"
)

type Wrapper struct {
	data_conn protocol.Conn
  ping_conn protocol.Conn
  nack_conn protocol.Conn

	pingLatencyChan chan float64

	fp *os.File
}

var meanPingLatency float64
var pingSentTime time.Time
var rframeIndex int64

var wIndex int64
var wcnt int
var frameSentTime time.Time
var serverPingIndex int64

var lastPingIndex int64
var indexDup int64

var rIndex int64
var rCnt int
var clientPingIndex int64

func NewServerWrapper(addr_data,addr_ping,addr_nack string,dataquic,pingquic,xackquic bool)Wrapper{
  w := Wrapper{}
	wg := sync.WaitGroup{}
	wg.Add(3)
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
	// create file pointer
	w.fp, _ = os.OpenFile("xack.dat",os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	// start ping and nack
	go w.RunPingServer()
	go w.RunNACKServer()
  return w
}

func NewClientWrapper(addr_data,addr_ping,addr_nack string,dataquic,pingquic,xackquic bool)Wrapper{
	w := Wrapper{}
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
	// create file pointer
	w.fp, _ = os.OpenFile("probe.dat",os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	go w.RunPingClient()
	go w.RunNACKClient()
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
	fmt.Println("TCP Trying to dial",addr)
	conn, err := net.Dial("tcp", addr)
	toolbox.Check(err)
	fmt.Println("connected to",addr)
	return conn
}

func (w Wrapper) Write(p []byte) (int,error){
  n,err := w.data_conn.Write(p)
  if n == 10{
  	wcnt += 1
  }else if wcnt >= 2{
  	wIndex += 1
  	wcnt = 2
  	frameSentTime = time.Now()
  }
  return n,err
}

func (w Wrapper) Read(p []byte) (int,error)  {
	n,err := w.data_conn.Read(p)
	if n==10{
		rCnt += 1
	}else if n!=10 && rCnt==2{
		rIndex += 1
		rCnt = 0
	}
	return n,err
}

func (w Wrapper) Close() error {
	return w.data_conn.Close()
}

func (w Wrapper) RunPingServer(){
	d := time.Duration(time.Millisecond*config.PingInterval)
	t := time.NewTicker(d)
	defer t.Stop()
	for{
		<-t.C
		tmp := time.Now().Format(time.StampMicro)
		_,err := w.ping_conn.Write([]byte(tmp))
		toolbox.Check(err)
	}
}

func (w Wrapper) RunPingClient(){
	for{
		buf := make([]byte, 22)
		_, err := io.ReadFull(w.ping_conn, buf)
		if err!=nil{break}
		sent_time,err := time.Parse(time.StampMicro,string(buf))
		if err!=nil{
			fmt.Println("Ping LOST")
			continue
		}
		sent_time = sent_time.Add(time.Millisecond*config.ServerTimerAdder)

		echo_time,_ := time.Parse(time.StampMicro,time.Now().Format(time.StampMicro))
		probe_latency := echo_time.Sub(sent_time).Seconds()
		w.pingLatencyChan <- probe_latency
		fmt.Printf("\nGot probe, downlink time %.7f, sent time %s\n",probe_latency,sent_time.Format(time.StampMicro))
		zero_time,_ := time.Parse(time.StampMicro,time.Time{}.Format(time.StampMicro))
		milli := sent_time.Sub(zero_time).Seconds()
		s := fmt.Sprintf("%f\t%f\n",milli,probe_latency)
    w.fp.Write([]byte(s))
	}
}

func (w Wrapper) RunNACKServer(){
	for {
		buf := make([]byte, 8)
		_, err := io.ReadFull(w.nack_conn, buf)
		toolbox.Check(err)
		meanPingLatency = toolbox.ByteToFloat64(buf)
		rframeIndex = toolbox.ReadInt64(w.nack_conn)
		serverPingIndex = toolbox.ReadInt64(w.nack_conn)
		fmt.Printf("latency updated to %.5f, recent frame %d, recent ping #%d\n",meanPingLatency,rframeIndex,serverPingIndex)
		zero_time,_ := time.Parse(time.StampMicro,time.Time{}.Format(time.StampMicro))
		infoUpdateTime,_ := time.Parse(time.StampMicro,time.Now().Add(time.Millisecond*config.ServerTimerAdder).Format(time.StampMicro))
		milli := infoUpdateTime.Sub(zero_time).Seconds()
		s := fmt.Sprintf("%f\t%f\n",milli,meanPingLatency)
    w.fp.Write([]byte(s))
	}
}


func (w Wrapper) RunNACKClient(){
	var nackTicker *time.Ticker
	d := time.Duration(time.Millisecond*config.XACKInterval)

	nackTicker = time.NewTicker(d)
	for{
		<-nackTicker.C
		// analyze ping latency
		existAbnormal := false
		var time_slice []float64
		l := len(w.pingLatencyChan)
		clientPingIndex += int64(l)
		for i:=0;i<l;i++{
			downlink_time := <-w.pingLatencyChan
			if downlink_time > float64(0.1){
				fmt.Println("==========Abnormal Ping",downlink_time)
				existAbnormal = true
			}
			time_slice = append(time_slice,downlink_time)
		}
		var meanStr []byte
		if existAbnormal||len(time_slice)==0{
			meanStr = toolbox.Float64ToByte(100)
		}else{
			mean,_ := stat.MeanStdDev(time_slice,nil)
			meanStr = toolbox.Float64ToByte(mean)
		}
		fmt.Println(time_slice)
		fmt.Printf("Avg probe latency %.5f at %s\n",toolbox.ByteToFloat64(meanStr),time.Now().Format(time.StampMicro))
		_,err := w.nack_conn.Write(meanStr)
		if err!=nil{break}
		toolbox.WriteInt64(w.nack_conn,rIndex)
		toolbox.WriteInt64(w.nack_conn,clientPingIndex)
	}
}


func (w Wrapper) IsDelay()bool{
	fmt.Printf("Judge from mean latency %.5f, ping #%d, latest sent frame %d, received frame %d\n",meanPingLatency,serverPingIndex,wIndex,rframeIndex)
	if serverPingIndex == lastPingIndex{
		indexDup += 1
	}else{
		indexDup = 1
	}
	if indexDup >= 3{
		return true
	}
	lastPingIndex = serverPingIndex

	if meanPingLatency > 0.1{
		return true
	}
	return false
}
