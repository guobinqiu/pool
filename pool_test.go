package pool

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetPut(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:7000")
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				bs := make([]byte, 1024)
				n, _ := conn.Read(bs)
				t.Log(string(bs[:n]))
				conn.Close()
			}(conn)
		}
	}()

	p := NewTcpConnPool(&Option{
		Host:     "127.0.0.1",
		Port:     7000,
		PoolSize: 10,
		InitCap:  5,
	})

	assert.Equal(t, 5, p.GetIdleConns())

	c, _ := p.GetConn()
	c.Write([]byte(fmt.Sprintf("client %d", 0)))
	assert.Equal(t, 4, p.GetIdleConns())

	c.Close()
	assert.Equal(t, 5, p.GetIdleConns())

	p.Close()
	l.Close()
}

func TestGetPut2(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:7000")
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {

			}(conn)
		}
	}()

	p := NewTcpConnPool(&Option{
		Host:     "127.0.0.1",
		Port:     7000,
		PoolSize: 10,
		InitCap:  0,
	})

	assert.Equal(t, 0, p.GetIdleConns())
	assert.Equal(t, 0, p.numConns)

	c, _ := p.GetConn()
	assert.Equal(t, 0, p.GetIdleConns())
	assert.Equal(t, 1, p.numConns)

	c.Close()
	assert.Equal(t, 1, p.GetIdleConns())
	assert.Equal(t, 1, p.numConns)

	p.Close()
	assert.Equal(t, 0, p.GetIdleConns())
	assert.Equal(t, 0, p.numConns)

	l.Close()
}

func TestGetPut3(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:7000")
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {

			}(conn)
		}
	}()

	p := NewTcpConnPool(&Option{
		Host:     "127.0.0.1",
		Port:     7000,
		PoolSize: 10,
		InitCap:  0,
	})

	assert.Equal(t, 0, p.GetIdleConns())
	assert.Equal(t, 0, p.numConns)

	c, _ := p.GetConn()
	assert.Equal(t, 0, p.GetIdleConns())
	assert.Equal(t, 1, p.numConns)

	p.ReleaseConn(c)
	assert.Equal(t, 0, p.GetIdleConns())
	assert.Equal(t, 0, p.numConns)

	p.Close()
	assert.Equal(t, 0, p.GetIdleConns())
	assert.Equal(t, 0, p.numConns)

	l.Close()
}

func TestWithinMaxConc(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:7000")
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				bs := make([]byte, 1024)
				n, _ := conn.Read(bs)
				t.Log(string(bs[:n]))
				conn.Close()
			}(conn)
		}
	}()

	p := NewTcpConnPool(&Option{
		Host:     "127.0.0.1",
		Port:     7000,
		PoolSize: 10,
		InitCap:  5,
	})

	assert.Equal(t, 5, p.GetIdleConns())

	fibers := 10
	for i := 0; i < fibers; i++ {
		go func(i int) {
			c, _ := p.GetConn()
			c.Write([]byte(fmt.Sprintf("client %d", i)))
		}(i)
	}

	time.Sleep(time.Second)

	assert.Equal(t, 0, p.GetIdleConns())

	p.Close()
	l.Close()
}

func TestWithinMaxConc2(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:7000")
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				bs := make([]byte, 1024)
				n, _ := conn.Read(bs)
				t.Log(string(bs[:n]))
				conn.Close()
			}(conn)
		}
	}()

	p := NewTcpConnPool(&Option{
		Host:     "127.0.0.1",
		Port:     7000,
		PoolSize: 10,
		InitCap:  5,
	})

	assert.Equal(t, 5, p.GetIdleConns())

	fibers := 4
	for i := 0; i < fibers; i++ {
		go func(i int) {
			c, _ := p.GetConn()
			c.Write([]byte(fmt.Sprintf("client %d", i)))
		}(i)
	}

	time.Sleep(time.Second)

	assert.Equal(t, 1, p.GetIdleConns())

	p.Close()
	l.Close()
}

func TestOverMax1(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:7000")
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				bs := make([]byte, 1024)
				n, _ := conn.Read(bs)
				t.Log(string(bs[:n]))
				conn.Close()
			}(conn)
		}
	}()

	p := NewTcpConnPool(&Option{
		Host:     "127.0.0.1",
		Port:     7000,
		PoolSize: 10,
		InitCap:  5,
	})

	assert.Equal(t, 5, p.GetIdleConns())

	errTimes := 0
	for i := 0; i < 12; i++ {
		c, err := p.GetConn() //last two wait 6s
		if err != nil {
			errTimes++
		} else {
			c.Write([]byte(fmt.Sprintf("client %d", i)))
		}
	}

	assert.Equal(t, 0, p.GetIdleConns())
	assert.Equal(t, 2, errTimes)

	p.Close()
	l.Close()
}

func TestOverMax2(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:7000")
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				bs := make([]byte, 1024)
				n, _ := conn.Read(bs)
				t.Log(string(bs[:n]))
				conn.Close()
			}(conn)
		}
	}()

	p := NewTcpConnPool(&Option{
		Host:     "127.0.0.1",
		Port:     7000,
		PoolSize: 10,
		InitCap:  5,
	})

	assert.Equal(t, p.GetIdleConns(), 5)

	errTimes := 0
	for i := 0; i < 12; i++ {
		c, err := p.GetConn()
		if err != nil {
			errTimes++
		} else {
			c.Write([]byte(fmt.Sprintf("client %d", i)))
			c.Close()
		}
	}

	assert.Equal(t, 5, p.GetIdleConns())
	assert.Equal(t, 0, errTimes)

	p.Close()
	l.Close()
}

func TestOverMax3(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:7000")
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				bs := make([]byte, 1024)
				n, _ := conn.Read(bs)
				t.Log(string(bs[:n]))
				conn.Close()
			}(conn)
		}
	}()

	p := NewTcpConnPool(&Option{
		Host:     "127.0.0.1",
		Port:     7000,
		PoolSize: 10,
		InitCap:  5,
	})

	assert.Equal(t, 5, p.GetIdleConns())

	fibers := 12
	for i := 0; i < fibers; i++ {
		go func(i int) {
			c, _ := p.GetConn()
			if c != nil {
				c.Write([]byte(fmt.Sprintf("client %d", i)))
			}
		}(i)
	}

	time.Sleep(time.Second)

	assert.Equal(t, 0, p.GetIdleConns())
	assert.Equal(t, fibers-p.opt.PoolSize-1, len(p.queue))

	p.Close()
	l.Close()
}

func TestRemoveIdleConns(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:7000")
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				conn.Close()
			}(conn)
		}
	}()

	p := NewTcpConnPool(&Option{
		Host:               "127.0.0.1",
		Port:               7000,
		PoolSize:           10,
		InitCap:            0,
		IdleTimeout:        1 * time.Second,
		IdleCheckFrequency: 1 * time.Second,
	})

	c, _ := p.GetConn()
	c.Close()

	time.Sleep(2 * time.Second)
	assert.Equal(t, 0, p.GetIdleConns())

	p.Close()
	l.Close()
}

func TestRemoveIdleConns2(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:7000")
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				conn.Close()
			}(conn)
		}
	}()

	p := NewTcpConnPool(&Option{
		Host:               "127.0.0.1",
		Port:               7000,
		PoolSize:           10,
		InitCap:            0,
		IdleTimeout:        2 * time.Second,
		IdleCheckFrequency: 1 * time.Second,
	})

	c, _ := p.GetConn()
	c.Close()

	time.Sleep(time.Second)
	assert.Equal(t, 1, p.GetIdleConns())

	p.Close()
	l.Close()
}
