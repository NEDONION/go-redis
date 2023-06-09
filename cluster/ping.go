package cluster

import "go-redis/interface/resp"

func ping(cluster *ClusterDatabase, c resp.Connection, cmdAndArgs [][]byte) resp.Reply {
	return cluster.db.Exec(c, cmdAndArgs) // 在集群上执行命令
}
