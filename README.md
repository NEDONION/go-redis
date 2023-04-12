# GoRedis - A Redis server implementation with Golang

## Requisites
- Golang
- Network Tools (PacketSender for me)

## Documentations

### 1. TCP connection and Go-redis parser Testing 

Download **PacketSender** in [https://github.com/dannagle/PacketSender](https://github.com/dannagle/PacketSender)

```bash
# test tcp and parser with echo_database which only reply the same content
address: 127.0.0.1
port: 6380
content: *3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n

// output:*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n
```

![](https://raw.githubusercontent.com/NEDONION/my-pics-space/main/20230327181957.png)

### 2. Database commands Testing

```bash
# send a SET command to db
address: 127.0.0.1
port: 6380
content: *3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n
// output: +OK\r\n

# send a GET command to db
send content: *2\r\n$3\r\nGET\r\n$3\r\nkey\r\n
// output: $5\r\nvalue\r\n
```

![](https://raw.githubusercontent.com/NEDONION/my-pics-space/main/20230411230345.png)



## References
- [https://github.com/HDT3213/godis](https://github.com/HDT3213/godis)

