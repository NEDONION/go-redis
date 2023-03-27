package parser

import (
	"bufio"
	"errors"
	"go-redis/interface/resp"
	"go-redis/lib/logger"
	"go-redis/resp/reply"
	"io"
	"runtime/debug"
	"strconv"
	"strings"
)

// Payload stores redis.Reply or error
type Payload struct {
	Data resp.Reply
	Err  error
}

// ParseStream reads data from io.Reader and send payloads through channel
// 从io.Reader中读取数据，并通过通道发送负载
func ParseStream(reader io.Reader) <-chan *Payload {
	ch := make(chan *Payload)
	go parse0(reader, ch)
	return ch
}

type readState struct {
	readingMultiLine  bool     // 是否在读取多行
	expectedArgsCount int      // 期望的参数个数
	msgType           byte     // 消息类型
	args              [][]byte // 已经解析的参数
	bulkLen           int64    // 字节长度
}

func (s *readState) finished() bool {
	return s.expectedArgsCount > 0 && len(s.args) == s.expectedArgsCount
}

func parse0(reader io.Reader, ch chan<- *Payload) {
	// 延迟函数会在函数执行完毕后执行
	// 该延迟函数首先检查是否有发生了panic，如果有，就将调用logger.Error函数并将堆栈信息输出到日志中。
	defer func() {
		if err := recover(); err != nil {
			logger.Error(string(debug.Stack()))
		}
	}()
	bufReader := bufio.NewReader(reader)
	var state readState
	var err error
	var msg []byte
	for {
		// read line
		var ioErr bool
		msg, ioErr, err = readLine(bufReader, &state)
		if err != nil {
			if ioErr { // 先判断是不是 IO error，如果是，就关闭 channel 并返回
				ch <- &Payload{
					Err: err,
				}
				close(ch)
				return
			}
			/// 不是 IO error，是协议错误，就将错误信息发送到 channel 中
			ch <- &Payload{
				Err: err,
			}
			continue
		}

		// 判断不是在读取多行
		if !state.readingMultiLine {
			// 判断是不是以 * 开头，如果是，就是多行
			if msg[0] == '*' {
				err := parseMultiBulkHeader(msg, &state)
				if err != nil {
					ch <- &Payload{
						Err: errors.New("protocol error: " + string(msg)),
					}
					state = readState{}
					continue
				}
				// 检查多行回复消息中预期的参数数量，如果为 0，则发送一个空的回复消息给 Redis 客户端
				if state.expectedArgsCount == 0 {
					ch <- &Payload{
						Data: &reply.EmptyMultiBulkReply{},
					}
					state = readState{}
					continue
				}
			} else if msg[0] == '$' { // 判断是不是以 $ 开头，如果是，就是单行
				err := parseBulkHeader(msg, &state)
				if err != nil {
					ch <- &Payload{
						Err: errors.New("protocol error: " + string(msg)),
					}
					state = readState{}
					continue
				}
				// 字节长度为 -1，表示空的 bulk reply
				if state.bulkLen == -1 {
					ch <- &Payload{
						Data: &reply.NullBulkReply{},
					}
					state = readState{}
					continue
				}
			} else {
				result, err := parseSingleLineReply(msg)
				ch <- &Payload{
					Data: result,
					Err:  err,
				}
				state = readState{}
				continue
			}
		} else {
			// 如果是在读取多行，就调用 readBody 函数来解析多行消息
			err = readBody(msg, &state)
			if err != nil {
				ch <- &Payload{
					Err: errors.New("protocol error: " + string(msg)),
				}
				state = readState{}
				continue
			}
			if state.finished() {
				var result resp.Reply
				if state.msgType == '*' {
					result = reply.MakeMultiBulkReply(state.args)
				} else if state.msgType == '$' {
					result = reply.MakeBulkReply(state.args[0])
				}
				ch <- &Payload{
					Data: result,
					Err:  err,
				}
				state = readState{}
			}
		}
	}
}

func readLine(bufReader *bufio.Reader, state *readState) ([]byte, bool, error) {
	var msg []byte
	var err error
	if state.bulkLen == 0 { // read normal line
		msg, err = bufReader.ReadBytes('\n')
		if err != nil {
			return nil, true, err
		}
		if len(msg) == 0 || msg[len(msg)-2] != '\r' {
			return nil, false, errors.New("protocol error: " + string(msg))
		}
	} else { // read bulk line (binary safe)
		msg = make([]byte, state.bulkLen+2) // 创建一个切片
		_, err = io.ReadFull(bufReader, msg)
		if err != nil {
			return nil, true, err
		}
		if len(msg) == 0 ||
			msg[len(msg)-2] != '\r' ||
			msg[len(msg)-1] != '\n' {
			// 检查读取的行是否以"\r\n"结尾，如果不是，就返回错误
			return nil, false, errors.New("protocol error: " + string(msg))
		}
		state.bulkLen = 0
	}
	return msg, false, nil
}

// 用于解析多条消息的 header
func parseMultiBulkHeader(msg []byte, state *readState) error {
	var err error
	var expectedLine uint64
	expectedLine, err = strconv.ParseUint(string(msg[1:len(msg)-2]), 10, 32) // 从msg中解析出期望的参数个数
	if err != nil {
		return errors.New("protocol error: " + string(msg))
	}
	if expectedLine == 0 {
		state.expectedArgsCount = 0
		return nil
	} else if expectedLine > 0 {
		// first line of multi bulk reply
		state.msgType = msg[0]
		state.readingMultiLine = true
		state.expectedArgsCount = int(expectedLine)
		state.args = make([][]byte, 0, expectedLine)
		return nil
	} else {
		return errors.New("protocol error: " + string(msg))
	}
}

func parseBulkHeader(msg []byte, state *readState) error {
	var err error
	state.bulkLen, err = strconv.ParseInt(string(msg[1:len(msg)-2]), 10, 64)
	if err != nil {
		return errors.New("protocol error: " + string(msg))
	}
	if state.bulkLen == -1 { // null bulk
		return nil
	} else if state.bulkLen > 0 {
		state.msgType = msg[0]
		state.readingMultiLine = true
		state.expectedArgsCount = 1
		state.args = make([][]byte, 0, 1)
		return nil
	} else {
		return errors.New("protocol error: " + string(msg))
	}
}

// 实现了解析 Redis 协议中的单行回复的功能
func parseSingleLineReply(msg []byte) (resp.Reply, error) {
	str := strings.TrimSuffix(string(msg), "\r\n")
	var result resp.Reply
	switch msg[0] {
	case '+': // status reply
		result = reply.MakeStatusReply(str[1:])
	case '-': // err reply
		result = reply.MakeErrReply(str[1:])
	case ':': // int reply
		val, err := strconv.ParseInt(str[1:], 10, 64)
		if err != nil {
			return nil, errors.New("protocol error: " + string(msg))
		}
		result = reply.MakeIntReply(val)
	}
	return result, nil
}

// read the non-first lines of multi bulk reply or bulk reply
func readBody(msg []byte, state *readState) error {
	line := msg[0 : len(msg)-2]
	var err error
	if line[0] == '$' {
		// bulk reply
		state.bulkLen, err = strconv.ParseInt(string(line[1:]), 10, 64)
		if err != nil {
			return errors.New("protocol error: " + string(msg))
		}
		if state.bulkLen <= 0 { // null bulk in multi bulks
			state.args = append(state.args, []byte{})
			state.bulkLen = 0
		}
	} else {
		state.args = append(state.args, line)
	}
	return nil
}
