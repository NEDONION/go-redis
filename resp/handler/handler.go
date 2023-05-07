package handler

import (
	"context"
	"go-redis/cluster"
	"go-redis/config"
	"go-redis/database"
	databaseface "go-redis/interface/database"
	"go-redis/lib/logger"
	"go-redis/lib/sync/atomic"
	"go-redis/resp/connection"
	"go-redis/resp/parser"
	"go-redis/resp/reply"
	"io"
	"net"
	"strings"
	"sync"
)

var (
	unknownErrReplyBytes = []byte("-ERR unknown\r\n")
)

// RespHandler implements tcp.Handler and serves as a redis handler
type RespHandler struct {
	activeConn sync.Map              // 用于存储当前活跃的客户端连接的同步哈希表
	db         databaseface.Database // 处理Redis命令的数据库接口
	closing    atomic.Boolean        // 记录服务器是否正在关闭，如果正在关闭，将拒绝新客户端和新请求
}

// MakeHandler creates a RespHandler instance
func MakeHandler() *RespHandler {
	var db databaseface.Database
	// 创建一个 EchoDatabase 实例，用于测试
	//db = database.NewEchoDatabase()
	// 创建一个真正的数据库实例
	// 判断并测试 cluster database
	if config.Properties.Self != "" &&
		len(config.Properties.Peers) > 0 {
		db = cluster.MakeClusterDatabase()
	} else {
		db = database.NewStandaloneDatabase()
	}
	return &RespHandler{
		db: db,
	}
}

func (h *RespHandler) closeClient(client *connection.Connection) {
	_ = client.Close()
	h.db.AfterClientClose(client)
	h.activeConn.Delete(client)
}

// Handle receives and executes redis commands
func (h *RespHandler) Handle(ctx context.Context, conn net.Conn) {
	if h.closing.Get() {
		// closing handler refuse new connection
		_ = conn.Close()
	}

	client := connection.NewConn(conn)
	h.activeConn.Store(client, 1)
	ch := parser.ParseStream(conn)
	for payload := range ch {
		if payload.Err != nil {
			// 先判断是否发送 EOF 或者 ErrUnexpectedEOF 信号，或者是否是网络连接关闭的错误，如果是，则关闭客户端连接
			if payload.Err == io.EOF ||
				payload.Err == io.ErrUnexpectedEOF ||
				strings.Contains(payload.Err.Error(), "use of closed network connection") {
				h.closeClient(client)
				logger.Info("connection closed: " + client.RemoteAddr().String())
				return
			}
			// 协议解析错误，返回错误信息
			errReply := reply.MakeErrReply(payload.Err.Error())
			err := client.Write(errReply.ToBytes())
			if err != nil {
				h.closeClient(client)
				logger.Info("connection closed: " + client.RemoteAddr().String())
				return
			}
			continue
		}
		// 如果 payload.Data 为空，则说明没有接收到任何数据，直接跳过
		if payload.Data == nil {
			logger.Error("empty payload")
			continue
		}
		// 尝试转换为 MultiBulkReply 类型，如果转换失败，则跳过这次解析
		// 为什么要转换为 MultiBulkReply 类型呢？因为 Redis 的命令都是以 MultiBulkReply 类型的数组来表示的
		// 比如：*2\r\n$3\r\nSET\r\n$3\r\nkey\r\n
		r, ok := payload.Data.(*reply.MultiBulkReply)
		if !ok {
			logger.Error("require multi bulk reply")
			continue
		}
		result := h.db.Exec(client, r.Args)
		if result != nil {
			_ = client.Write(result.ToBytes())
		} else {
			_ = client.Write(unknownErrReplyBytes)
		}
	}
}

// Close stops handler
func (h *RespHandler) Close() error {
	logger.Info("handler shutting down...")
	h.closing.Set(true)
	// TODO: concurrent wait
	h.activeConn.Range(func(key interface{}, val interface{}) bool {
		client := key.(*connection.Connection)
		_ = client.Close()
		return true
	})
	h.db.Close()
	return nil
}
