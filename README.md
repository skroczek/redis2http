# redis2http

This is a weekend project of mine that makes a list stored in redis available as a text file over http.
That can be useful if you want to store a [rspamd](https://github.com/rspamd/rspamd) regex multimap in redis.
The main reason for the project is to learn about Go, especially in conjunction with Docker.

#Build
Prerequisite: Working Docker with experimental features enabled

```
export COMPOSE_DOCKER_CLI_BUILD=1
export DOCKER_BUILDKIT=1
```
then
```
> git clone https://github.com/skroczek/redis2http.git
> make PLATFORM=linux/amd64
> bin/redis2http
```

or

```
go build -o bin/redis2http .
```

# Configuration

```yaml
server:
  host: 0.0.0.0
  port: 8080
  timeout:
    server: "30s"
    read: "15s"
    write: "10s"
    idle: "5s"
redis:
  addr: localhost:6379
  password: ""
  db: 0
  keys:
    - wv_high_spammy_tlds
logging:
  prefix: "redis2http "
  flag: 19
```
For the flags you can use the integer representation of the following:

	Ldate         = 1 << iota     //  1 the date in the local time zone: 2009/01/23
	Ltime                         //  2 the time in the local time zone: 01:23:23
	Lmicroseconds                 //  4 microsecond resolution: 01:23:23.123123.  assumes Ltime.
	Llongfile                     //  8 full file name and line number: /a/b/c/d.go:23
	Lshortfile                    // 16 final file name element and line number: d.go:23. overrides Llongfile
	LUTC                          // 32 if Ldate or Ltime is set, use UTC rather than the local time zone
	Lmsgprefix                    // 64 move the "prefix" from the beginning of the line to before the message
	LstdFlags     = Ldate | Ltime //  3 initial values for the standard logger

The used used flag `19` is `Ldate | Lmicroseconds | Lshortfile` or `LstdFlags | Lshortfile` and will produce output like this:
```
redis2http 2021/03/14 20:30:27 redis2http.go:164: Server is starting on 0.0.0.0:8080 with PID 139339
```

## Thanks to

https://www.docker.com/blog/containerize-your-go-developer-environment-part-1/

https://gist.github.com/enricofoltran/10b4a980cd07cb02836f70a4ab3e72d7

https://github.com/enricofoltran/simple-go-server/blob/master/main.go

https://fabianlee.org/2017/05/21/golang-running-a-go-binary-as-a-systemd-service-on-ubuntu-16-04/

https://dev.to/koddr/let-s-write-config-for-your-golang-web-app-on-right-way-yaml-5ggp