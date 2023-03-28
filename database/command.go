package database

import "strings"

var cmdTable = make(map[string]*command)

type command struct {
	// 执行命令的函数
	executor ExecFunc
	// allow number of args, arity < 0 means len(args) >= -arity
	// 表示参数的个数，如果是正数，表示参数的个数必须等于这个数，如果是负数，表示参数的个数必须大于等于这个数的绝对值
	arity int
}

// RegisterCommand registers a new command. 用于注册一个新的命令
// arity means allowed number of cmdArgs, arity < 0 means len(args) >= -arity.
// for example: the arity of `get` is 2, `mget` is -2
func RegisterCommand(name string, executor ExecFunc, arity int) {
	name = strings.ToLower(name)
	cmdTable[name] = &command{
		executor: executor,
		arity:    arity,
	}
}
