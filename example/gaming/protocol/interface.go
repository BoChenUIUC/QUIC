package protocol

type Conn interface {
	Write([]byte) (int,error)
	Read([]byte) (int,error)
	Close() error
}
