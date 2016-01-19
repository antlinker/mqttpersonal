# MQTT针对单人聊天的压力测试

## 获取

``` bash
$ go get github.com/antlinker/mqttpersonal
```

## 使用

``` bash
$ cd github.com/antlinker/mqttpersonal
$ go build -o mqttpersonal
```

## Help

``` bash
$ ./mqttpersonal help
```

```
NAME:
   mqttpersonal - MQTT单人聊天测试

USAGE:
   ./mqttpersonal [global options] command [command options] [arguments...]

VERSION:
   0.1.0

AUTHOR(S):
   Lyric

COMMANDS:
   generate, gen        生成用户并分配好友
   publish, pub         发布消息
   clear, c             清除数据
   help, h              Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h           show help
   --version, -v        print the version
```