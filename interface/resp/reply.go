package resp

// Reply is the interface of redis serialization protocol message
// redis 序列化协议的接口
type Reply interface {
	ToBytes() []byte
}
