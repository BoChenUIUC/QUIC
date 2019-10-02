package main

import (
  "fmt"
  // "net"
  "io"
  "context"
  "time"
	"crypto/tls"
	// "github.com/lucas-clemente/quic-go/example/gaming/config"
	"github.com/lucas-clemente/quic-go/example/gaming/toolbox"
  quic "github.com/lucas-clemente/quic-go"
)

func main(){
  RunProbeClient()
}

func RunProbeClient(){
  tlsConf := &tls.Config{
    InsecureSkipVerify: true,
    NextProtos:         []string{"quic-echo-example"},
  }
  // session, err := quic.DialAddr("10.192.55.9:8086", tlsConf, nil)
  session, err := quic.DialAddr("18.219.44.207:8086", tlsConf, nil)
  toolbox.Check(err)
  conn, err := session.OpenStreamSync(context.Background())
  toolbox.Check(err)

  conn.Write([]byte(" "))

  prevTime := time.Time{}
  for{
		buf := make([]byte, 22)
		_, err := io.ReadFull(conn, buf)
		toolbox.Check(err)

		echo_time,_ := time.Parse(time.StampMicro,time.Now().Format(time.StampMicro))
    if prevTime.IsZero()==false{
      time_diff := echo_time.Sub(prevTime).Seconds()
      // should print time diff and run another goroutine to send timestamps
      fmt.Println(time_diff)
    }
    prevTime = echo_time
	}
}
