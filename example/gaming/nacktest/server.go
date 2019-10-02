package main

import (
  "time"
  "fmt"
  // "io"
	"github.com/lucas-clemente/quic-go/example/gaming/config"
	"github.com/lucas-clemente/quic-go/example/gaming/toolbox"
  "context"
  quic "github.com/lucas-clemente/quic-go"
)

func main(){
  RunProbeServer()

}

func RunProbeServer(){
  fmt.Println("Probe server listening at",config.ProbeAddr)
	listener, err := quic.ListenAddr("10.192.55.9:8086", toolbox.GenerateTLSConfig(), nil)
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
	for i:=0;i<100;i++{
		<-t.C
		tmp := time.Now().Format(time.StampMicro)
		_,err := conn.Write([]byte(tmp))
    fmt.Println("Sent"+tmp)
		toolbox.Check(err)
	}

}
