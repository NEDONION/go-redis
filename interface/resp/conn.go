package resp

// Connection represents a connection with redis client
type Connection interface {
	Write([]byte) error
	GetDBIndex() int // used for multi database
	SelectDB(int)
}
