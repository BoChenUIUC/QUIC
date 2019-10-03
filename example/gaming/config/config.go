package config

// time sync
// const ServerTimerAdder = -5*3600*1000
const ServerTimerAdder = 0

// file path
const FilePath = "/var/www/Bigbunny/"

// const SERVIP = "localhost:"
// const SERVIP = "3.14.153.25:"
// const SERVIP = "10.192.55.9:"
const SERVIP = "192.168.101.48:"

// TCP
const TCPBufSize = 1024
const TCPAddr = SERVIP + "8080"
const TCPInstAddr = SERVIP + "8081"
const TCPTimestampAddr = SERVIP + "8084"

// QUIC
const QUICAddr = TCPAddr

// new proto settings
const DataQUIC = false
const PingQUIC = false
const XACKQUIC = false

// ping
const PingSendAddr = SERVIP + "8082"
const PingRecvAddr = SERVIP + "8083"
const PingInterval = 10

// nack
const NACKRecvAddr = SERVIP + "8085"
const XACKInterval = 30

// probe
const ProbeInterval = 11
const ProbeAddr = SERVIP + "8086"

// choice of protocol
const TCP = 2
const QUIC = 1
const NEWPROTO = 0

// application layer mode
const Default = 0
const Drop = 1
const DumbPrefetch = 2
const Prefetch = 3

// FEC
const Reliable = true
const UseFEC = false
const NumDataShards = 3
const NumParityShards = 1
const NumShards = NumDataShards + NumParityShards
