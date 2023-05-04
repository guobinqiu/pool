package connpool

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

	p := NewTcpConnPool(&Opt{
		Host:              "127.0.0.1",
		Port:              7000,
		MaxConns:          10,
		MinIdleConns:      5,
		IdleTimeout:       60 * time.Second,
		IdleScanFrequency: time.Second,
	})

	assert.Equal(t, 5, p.GetIdleConns())

	c, _ := p.Get()
	c.conn.Write([]byte(fmt.Sprintf("client %d", 0)))
	assert.Equal(t, 4, p.GetIdleConns())

	p.Put(c)
	assert.Equal(t, 5, p.GetIdleConns())

	p.Close()
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

	p := NewTcpConnPool(&Opt{
		Host:              "127.0.0.1",
		Port:              7000,
		MaxConns:          10,
		MinIdleConns:      5,
		IdleTimeout:       100 * time.Second,
		IdleScanFrequency: 100 * time.Microsecond,
	})

	assert.Equal(t, 5, p.GetIdleConns())

	fibers := 10
	for i := 0; i < fibers; i++ {
		go func(i int) {
			c, _ := p.Get()
			c.conn.Write([]byte(fmt.Sprintf("client %d", i)))
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

	p := NewTcpConnPool(&Opt{
		Host:              "127.0.0.1",
		Port:              7000,
		MaxConns:          10,
		MinIdleConns:      5,
		IdleTimeout:       100 * time.Second,
		IdleScanFrequency: 100 * time.Microsecond,
	})

	assert.Equal(t, 5, p.GetIdleConns())

	fibers := 4
	for i := 0; i < fibers; i++ {
		go func(i int) {
			c, _ := p.Get()
			c.conn.Write([]byte(fmt.Sprintf("client %d", i)))
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

	p := NewTcpConnPool(&Opt{
		Host:              "127.0.0.1",
		Port:              7000,
		MaxConns:          10,
		MinIdleConns:      5,
		IdleTimeout:       60 * time.Second,
		IdleScanFrequency: time.Second,
	})

	assert.Equal(t, 5, p.GetIdleConns())

	errTimes := 0
	for i := 0; i < 12; i++ {
		c, err := p.Get() //last two wait 6s
		if err != nil {
			errTimes++
		} else {
			c.conn.Write([]byte(fmt.Sprintf("client %d", i)))
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

	p := NewTcpConnPool(&Opt{
		Host:              "127.0.0.1",
		Port:              7000,
		MaxConns:          10,
		MinIdleConns:      5,
		IdleTimeout:       60 * time.Second,
		IdleScanFrequency: time.Second,
	})

	assert.Equal(t, p.GetIdleConns(), 5)

	errTimes := 0
	for i := 0; i < 12; i++ {
		c, err := p.Get()
		if err != nil {
			errTimes++
		} else {
			c.conn.Write([]byte(fmt.Sprintf("client %d", i)))
			p.Put(c)
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

	p := NewTcpConnPool(&Opt{
		Host:              "127.0.0.1",
		Port:              7000,
		MaxConns:          10,
		MinIdleConns:      5,
		IdleTimeout:       60 * time.Second,
		IdleScanFrequency: time.Second,
	})

	assert.Equal(t, 5, p.GetIdleConns())

	fibers := 12
	for i := 0; i < fibers; i++ {
		go func(i int) {
			c, _ := p.Get()
			if c != nil {
				c.conn.Write([]byte(fmt.Sprintf("client %d", i)))
			}
		}(i)
	}

	time.Sleep(time.Second)

	assert.Equal(t, 0, p.GetIdleConns())
	assert.Equal(t, fibers-p.maxConns-1, len(p.queue))

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

	p := NewTcpConnPool(&Opt{
		Host:              "127.0.0.1",
		Port:              7000,
		MaxConns:          10,
		MinIdleConns:      5,
		IdleTimeout:       1 * time.Second,
		IdleScanFrequency: 1 * time.Second,
	})
	assert.Equal(t, 5, p.GetIdleConns())

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

	p := NewTcpConnPool(&Opt{
		Host:              "127.0.0.1",
		Port:              7000,
		MaxConns:          10,
		MinIdleConns:      5,
		IdleTimeout:       2 * time.Second,
		IdleScanFrequency: 1 * time.Second,
	})
	assert.Equal(t, 5, p.GetIdleConns())

	time.Sleep(time.Second)
	assert.Equal(t, 5, p.GetIdleConns())

	p.Close()
	l.Close()
}
