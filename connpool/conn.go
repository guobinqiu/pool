package connpool

import (
	"net"
	"time"
)

type TcpConn struct {
	id      string
	conn    net.Conn
	homedAt time.Time
	p       *TcpConnPool
}

func (c *TcpConn) Read(b []byte) (n int, err error) {
	return c.conn.Read(b)
}

func (c *TcpConn) Write(b []byte) (n int, err error) {
	return c.conn.Write(b)
}

// put back to pool
func (c *TcpConn) Close() {
	c.p.mu.Lock()
	c.p.idleConns[c.id] = c
	c.homedAt = time.Now()
	c.p.mu.Unlock()
}

func (c *TcpConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *TcpConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *TcpConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *TcpConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *TcpConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
