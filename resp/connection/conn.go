package connection

import (
	"go-redis/lib/sync/wait"
	"net"
	"sync"
	"time"
)

// Connection represents a connection with a redis-cli
type Connection struct {
	conn         net.Conn   // 客户端连接
	waitingReply wait.Wait  // 等待回复完成
	mu           sync.Mutex // 处理发送回复时的互斥锁
	selectedDB   int        // 选择的数据库
}

func NewConn(conn net.Conn) *Connection {
	// 返回指向Connection的指针
	return &Connection{
		conn: conn,
	}
}

// RemoteAddr returns the remote network address
func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Close disconnect with the client
func (c *Connection) Close() error {
	c.waitingReply.WaitWithTimeout(10 * time.Second)
	_ = c.conn.Close()
	return nil
}

// Write sends response to client over tcp connection
// 发送回复给客户端
func (c *Connection) Write(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	c.mu.Lock()
	c.waitingReply.Add(1)
	defer func() {
		c.waitingReply.Done()
		c.mu.Unlock()
	}()

	_, err := c.conn.Write(b)
	return err
}

// GetDBIndex returns selected db
func (c *Connection) GetDBIndex() int {
	return c.selectedDB
}

// SelectDB selects a database
func (c *Connection) SelectDB(dbNum int) {
	c.selectedDB = dbNum
}
