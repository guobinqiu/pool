package pool

import (
	"net"
	"time"
)

type Conn struct {
	id      int64
	conn    net.Conn
	homedAt time.Time
	p       *connPool
}

func (c *Conn) Read(b []byte) (n int, err error) {
	return c.conn.Read(b)
}

func (c *Conn) Write(b []byte) (n int, err error) {
	return c.conn.Write(b)
}

// put back to pool
func (c *Conn) Close() error {
	c.p.mu.Lock()
	defer c.p.mu.Unlock()
	if len(c.p.idleConns) == c.p.maxCap {
		return c.conn.Close()
	}
	c.p.idleConns[c.id] = c
	c.homedAt = time.Now()
	return nil
}

func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Conn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
