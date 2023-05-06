TCP Connection Pool
---

### Configuration

name|type|desc
---|---|---
Host|string|remote server ip, default to 127.0.0.1
Port|int|remote server port
PoolSize|int|maximum of connections, default to 10 * CPU cores
MinIdleConns|int|minimum of connections
IdleTimeout|int|default to 5 minutes
IdleCheckFrequency|int|default to 1 minutes
DialTimeout|int|default to 5 seconds
ReadTimeout|int|default to never
WriteTimeout|int|default to never
QueueSize|int|default to 10000
QueueConnTimeout|int|default to 3 seconds

### Example

```
func main() {
	p := connpool.NewTcpConnPool(&connpool.Opt{
		Host:               "127.0.0.1",
		Port:               7000,
		PoolSize:           10,
		MinIdleConns:       5,
		IdleTimeout:        5 * time.Minute,
		IdleCheckFrequency: 1 * time.Minute,
		DialTimeout:        5 * time.Second,
		ReadTimeout:        0,
		WriteTimeout:       0,
		QueueSize:          10000,
		QueueConnTimeout:   3 * time.Second,
	})

	c, err := p.GetConn()
	if err != nil {
		c.Write([]byte("something"))
	}
	c.Close()

    c, err := p.GetConn()
    c.ReleaseConn()

	p.Close()
}
```
