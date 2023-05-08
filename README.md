A High Performance Connection Pool
---

### Params

name|required|desc
---|---|---
Factory|Y|net.Conn generator
InitCap|N|default to 0
MaxCap|N|default to 10 * CPU cores
IdleTimeout|N|default to 5 minutes
IdleCheckFrequency|N|default to 1 minutes

### Usage

```
package main

import (
	"net"
	"time"

	"github.com/guobinqiu/pool"
)

func main() {
	pool := pool.NewConnPool(&pool.Option{
		Factory: func() (net.Conn, error) {
			return net.Dial("tcp", "host:port")
		},
		InitCap:            5,
		MaxCap:             10,
		IdleTimeout:        5 * time.Minute,
		IdleCheckFrequency: time.Minute,
	})
	
	// get a connection from pool
	c, _ := pool.GetConn()

	// put connection back to pool
	c.Close()

	// remove connection from pool
	pool.ReleaseConn(c)

	pool.Close()
}
```

### Test

```
go clean -testcache && go test -v .
```
