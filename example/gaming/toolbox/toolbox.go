package toolbox

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"encoding/binary"
  "math"
	"time"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
  "os"
	"io/ioutil"
	"github.com/lucas-clemente/quic-go/example/gaming/protocol"
	"github.com/lucas-clemente/quic-go/example/gaming/config"
	reedsolomon "github.com/klauspost/reedsolomon"
)

func SendFEC(buf []byte, connection protocol.Conn) int{
	if config.NumParityShards == 0{
		// prepare the shards
		enc, err := reedsolomon.New(config.NumDataShards,1)
		Check(err)
		data, err := enc.Split(buf)
		Check(err)
		// append shards
		var size int
		for _,buf := range data[:config.NumDataShards]{
			n,err := connection.Write(buf)
			Check(err)
			size += n
		}
		return size
	}else{
		// prepare the shards
		enc, err := reedsolomon.New(config.NumDataShards,config.NumParityShards)
		Check(err)
		data, err := enc.Split(buf)
		Check(err)
		err = enc.Encode(data)
		Check(err)
		// append shards
		var size int
		for _,buf := range data{
			n,err := connection.Write(buf)
			Check(err)
			size += n
		}
		return size
	}
}

func RecvFEC(connection protocol.Conn, fileSize int64)[]byte{
	// compute size considering padding and FEC
	var shard_size int64
	if fileSize%config.NumDataShards != 0{
		shard_size = fileSize/config.NumDataShards+1
	}else{
		shard_size = fileSize/config.NumDataShards
	}

	// receive shards
	data := make([][]byte,config.NumShards)
	for i:=0;i<config.NumShards;i++{
		data[i] = make([]byte,shard_size)
		_, err := io.ReadFull(connection, data[i])
		Check(err)
	}

	// deal with 0 parity case
	if config.NumParityShards == 0{
		buf := []byte{}
		for i:=0;i<config.NumDataShards;i++{
			buf = append(buf,data[i]...)
		}
		return buf[:fileSize]
	}else{
		// reconstruct shards
		enc, err := reedsolomon.New(config.NumDataShards,config.NumParityShards)
		Check(err)
		err = enc.Reconstruct(data)
		Check(err)

		// convert shards to buffer
		buf := []byte{}
		for i:=0;i<config.NumDataShards;i++{
			buf = append(buf,data[i]...)
		}
		return buf[:fileSize]
	}
}


func ReadLen(connection protocol.Conn, bufLen int)[]byte{
	buf := make([]byte, bufLen)
	//Get the filesize
	n,_ :=connection.Read(buf)
	for {
		if n==bufLen{
			break
		}
		tmp := make([]byte,10-n)
		n1, err := connection.Read(tmp)
		buf = append(buf[:n],tmp[:n1]...)
		n += n1
		fmt.Println("Something not right",buf)
		if err!=nil{
			panic(err)
		}
	}
	return buf
}

func WriteFloat64(connection protocol.Conn, val float64){
	_, err := connection.Write(Float64ToByte(val))
	Check(err)
	return
}

func ReadFloat64(connection protocol.Conn)float64{
	float64_buf := make([]byte, 8)
	_,err := connection.Read(float64_buf)
	Check(err)
	return ByteToFloat64(float64_buf)
}

func WriteInt64(connection protocol.Conn,val int64){
	valStr := FillString(strconv.FormatInt(val, 10), 10)
	_,err := connection.Write([]byte(valStr))
	Check(err)
	return
}


func ReadInt64(connection protocol.Conn) (int64){
	//Create buffer to read in the size of the file
	buf := ReadLen(connection,10)
	//Strip the ':' from the received size, convert it to a int64
	val,err := strconv.ParseInt(strings.Trim(string(buf), ":"), 10, 64)
	Check(err)
	return val
}

func ReadTime(connection protocol.Conn) time.Time{
	buf := ReadLen(connection,22)
	t,err := time.Parse(time.StampMicro,string(buf))
	Check(err)
	return t
}

func WriteTime(connection protocol.Conn, t time.Time)error{
	_,err := connection.Write([]byte(t.Format(time.StampMicro)))
	return err
}

func Float64ArrayToFile(floats []float64,fileName string){
	f, err := os.OpenFile(fileName,os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	Check(err)
	for _,flt := range floats{
		_, err := f.Write(Float64ToByte(flt))
		Check(err)
	}
}

func FileToFloat64Array(fileName string)([]float64){
	buf, err := ioutil.ReadFile(fileName)
	Check(err)
	arrlen := len(buf)/8
	floatarr := make([]float64,arrlen)
	for i:=0;i<arrlen;i++{
		floatarr[i] = ByteToFloat64(buf[i*8:(i+1)*8])
	}
	return floatarr
}

func Float64ToByte(float float64) []byte {
    bits := math.Float64bits(float)
    bytes := make([]byte, 8)
    binary.LittleEndian.PutUint64(bytes, bits)

    return bytes
}

func ByteToFloat64(bytes []byte) float64 {
    bits := binary.LittleEndian.Uint64(bytes)

    return math.Float64frombits(bits)
}

func FillString(retunString string, toLength int) string {
	for {
		lengtString := len(retunString)
		if lengtString < toLength {
			retunString = retunString + ":"
			continue
		}
		break
	}
	return retunString
}

func Check(e error) {
    if e != nil {
			fmt.Println(e)
        panic(e)
    }
}

func GenerateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-echo-example"},
	}
}
