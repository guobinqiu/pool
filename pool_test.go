package pool

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var l net.Listener

func init() {
	l, _ = net.Listen("tcp", "127.0.0.1:0")
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {}(conn)
		}
	}()
}

var factory = func() (net.Conn, error) {
	return net.Dial("tcp", l.Addr().String())
}

func TestGetAndClose1(t *testing.T) {
	p := NewConnPool(&Option{
		Factory: factory,
		InitCap: 5,
	})
	assert.Equal(t, 5, p.GetIdleConns())

	c, _ := p.GetConn()
	assert.Equal(t, 4, p.GetIdleConns())

	c.Close()
	assert.Equal(t, 5, p.GetIdleConns())

	p.Close()
}

func TestGetAndClose2(t *testing.T) {
	p := NewConnPool(&Option{
		Factory: factory,
		InitCap: 0,
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
}

func TestRelease(t *testing.T) {
	p := NewConnPool(&Option{
		Factory: factory,
		InitCap: 0,
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
}

func TestLessThanInitCap(t *testing.T) {
	p := NewConnPool(&Option{
		Factory: factory,
		InitCap: 5,
	})

	wg := new(sync.WaitGroup)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.GetConn()
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, p.GetIdleConns())
	assert.Equal(t, 5, p.numConns)

	p.Close()
}

func TestLessThanMaxCap(t *testing.T) {
	p := NewConnPool(&Option{
		Factory: factory,
		MaxCap:  10,
	})

	wg := new(sync.WaitGroup)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.GetConn()
		}()
	}
	wg.Wait()

	assert.Equal(t, 0, p.GetIdleConns())
	assert.Equal(t, 10, p.numConns)

	p.Close()
}

func TestGreatThanMaxCap(t *testing.T) {
	p := NewConnPool(&Option{
		Factory: factory,
		MaxCap:  10,
	})

	for i := 0; i < 20; i++ {
		go func() {
			p.GetConn()
		}()
	}

	time.Sleep(time.Second)

	assert.Equal(t, 0, p.GetIdleConns())
	assert.True(t, len(p.reqQueue) > 0)
	assert.Equal(t, 10, p.numConns)

	p.Close()
}

func TestReleaseTimeout(t *testing.T) {
	p := NewConnPool(&Option{
		Factory:            factory,
		IdleTimeout:        time.Second,
		IdleCheckFrequency: time.Second,
	})
	c, _ := p.GetConn()
	c.Close()

	time.Sleep(2 * time.Second)
	assert.Equal(t, 0, p.GetIdleConns())

	p.Close()
}

func TestReleaseUntimeout(t *testing.T) {
	p := NewConnPool(&Option{
		Factory:            factory,
		IdleTimeout:        2 * time.Second,
		IdleCheckFrequency: time.Second,
	})

	c, _ := p.GetConn()
	c.Close()

	time.Sleep(time.Second)
	assert.Equal(t, 1, p.GetIdleConns())

	p.Close()
}
