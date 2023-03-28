# GoRedis - A Redis server implementation with Golang

## Requisites
- Golang
- Network Tools (PacketSender for me)

## Documentations

### 1. TCP and Go-redis parser Testing 

Download **PacketSender** in [https://github.com/dannagle/PacketSender](https://github.com/dannagle/PacketSender)

```bash
# test tcp and parser with echo_database which only reply the same content
address: 127.0.0.1
port: 6380
content: *3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n

// output:*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n
```

![](https://raw.githubusercontent.com/NEDONION/my-pics-space/main/20230327181957.png)


## References
- [https://github.com/HDT3213/godis](https://github.com/HDT3213/godis)

