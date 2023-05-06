package connpool

import (
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TcpConnPool struct {
	opt       *Opt
	idleConns map[string]*TcpConn
	numConns  int
	mu        *sync.Mutex
	queue     chan *ConnReq
	ticker    *time.Ticker
}

type Opt struct {
	Host               string
	Port               int
	PoolSize           int
	MinIdleConns       int
	IdleTimeout        time.Duration
	IdleCheckFrequency time.Duration
	DialTimeout        time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	QueueSize          int
	QueueConnTimeout   time.Duration
}

type ConnReq struct {
	connCh chan *TcpConn
	errCh  chan error
}

func (opt *Opt) init() {
	if opt.Host == "" {
		opt.Host = "127.0.0.1"
	}
	if opt.PoolSize == 0 {
		opt.PoolSize = 10 * runtime.NumCPU()
	}
	if opt.IdleTimeout == 0 {
		opt.IdleTimeout = 5 * time.Minute
	}
	if opt.IdleCheckFrequency == 0 {
		opt.IdleCheckFrequency = time.Minute
	}
	if opt.DialTimeout == 0 {
		opt.DialTimeout = 5 * time.Second
	}
	if opt.QueueSize == 0 {
		opt.QueueSize = 10000
	}
	if opt.QueueConnTimeout == 0 {
		opt.QueueConnTimeout = 3 * time.Second
	}
}

func NewTcpConnPool(opt *Opt) *TcpConnPool {
	opt.init()
	p := &TcpConnPool{
		opt:       opt,
		idleConns: make(map[string]*TcpConn, opt.PoolSize),
		mu:        new(sync.Mutex),
		queue:     make(chan *ConnReq, opt.QueueSize),
		ticker:    time.NewTicker(opt.IdleCheckFrequency),
	}
	for len(p.idleConns) < opt.MinIdleConns {
		c, err := p.newConn()
		if err != nil {
			panic(err)
		}
		p.idleConns[c.id] = c
		p.numConns++
	}
	go p.handleQueue()
	go p.release()
	return p
}

func (p *TcpConnPool) Close() {
	close(p.queue)
	p.ticker.Stop()
	for _, c := range p.idleConns {
		c.conn.Close()
		p.ReleaseConn(c)
	}
}

func (p *TcpConnPool) newConn() (*TcpConn, error) {
	addr := fmt.Sprintf("%s:%d", p.opt.Host, p.opt.Port)
	conn, err := net.DialTimeout("tcp", addr, p.opt.DialTimeout) //only accept tcp for now
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if p.opt.ReadTimeout > 0 {
		conn.SetReadDeadline(now.Add(p.opt.ReadTimeout))
	}
	if p.opt.WriteTimeout > 0 {
		conn.SetWriteDeadline(now.Add(p.opt.WriteTimeout))
	}
	return &TcpConn{
		id:   uuid.New().String(),
		conn: conn,
		p:    p,
	}, nil
}

func (p *TcpConnPool) GetConn() (*TcpConn, error) {
	p.mu.Lock()
	if len(p.idleConns) > 0 {
		for k, v := range p.idleConns {
			delete(p.idleConns, k)
			p.mu.Unlock()
			return v, nil
		}
	}

	if p.numConns < p.opt.PoolSize {
		c, err := p.newConn()
		if err != nil {
			return nil, err
		}
		p.numConns++
		p.mu.Unlock()
		return c, nil
	}

	//p.openConns == p.PoolSize
	// come to queue
	req := &ConnReq{
		connCh: make(chan *TcpConn, 1),
		errCh:  make(chan error, 1),
	}
	p.queue <- req

	p.mu.Unlock()

	// blocked
	select {
	case conn := <-req.connCh:
		return conn, nil
	case err := <-req.errCh:
		return nil, err
	}
}

func (p *TcpConnPool) handleQueue() {
	for req := range p.queue {
		t := time.After(p.opt.QueueConnTimeout)
		var timeout bool
		var done bool
		for {
			if timeout || done {
				break
			}
			select {
			case <-t:
				req.errCh <- errors.New("request timeout")
				timeout = true
			default:
				if len(p.idleConns) > 0 {
					req.connCh <- p.popConn()
					done = true
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (p *TcpConnPool) popConn() *TcpConn {
	p.mu.Lock()
	for _, c := range p.idleConns {
		delete(p.idleConns, c.id)
		p.mu.Unlock()
		return c
	}
	return nil
}

func (p *TcpConnPool) release() {
	for range p.ticker.C {
		for _, c := range p.idleConns {
			if time.Now().After(c.homedAt.Add(p.opt.IdleTimeout)) {
				p.ReleaseConn(c)
			}
		}
	}
}

func (p *TcpConnPool) ReleaseConn(c *TcpConn) {
	p.mu.Lock()
	delete(p.idleConns, c.id)
	p.numConns--
	c.conn.Close()
	p.mu.Unlock()
}

func (p *TcpConnPool) GetIdleConns() int {
	return len(p.idleConns)
}
