TCP Connection Pool
---

### Configuration

name|type|desc
---|---|---
Host|string|remote server ip, default to 127.0.0.1
Port|int|remote server port
PoolSize|int|default to 10 * CPU cores
InitCap|int|default to PoolSize/2
IdleTimeout|int|default to 5 minutes
IdleCheckFrequency|int|default to 1 minutes
DialTimeout|int|default to 5 seconds
ReadTimeout|int|0 to never
WriteTimeout|int|0 to never

### Example

```
func main() {
	p := pool.NewTcpConnPool(&connpool.Opt{
		Host:               "127.0.0.1",
		Port:               7000,
		PoolSize:           10,
		MinIdleConns:       5,
		IdleTimeout:        5 * time.Minute,
		IdleCheckFrequency: 1 * time.Minute,
		DialTimeout:        5 * time.Second,
		ReadTimeout:        0,
		WriteTimeout:       0,
	})

	// get a connection from pool
	c, err := p.GetConn()
	
	// put connection back to pool
	c.Close() 
	
	// remove connection from pool
	c.ReleaseConn()

	p.Close()
}
```

### Run Test

```
go clean -testcache && go test -v .
```
