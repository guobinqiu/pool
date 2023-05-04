package connpool

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TcpConnPool struct {
	host         string
	port         int
	maxConns     int
	minIdleConns int
	idleTimeout  time.Duration
	idleConns    map[string]*TcpConn
	numConns     int
	mu           *sync.Mutex
	queue        chan *Req
	ticker       time.Ticker
}

type TcpConn struct {
	id        string
	conn      net.Conn
	createdAt time.Time
}

type Opt struct {
	Host              string
	Port              int
	MaxConns          int
	MinIdleConns      int
	IdleTimeout       time.Duration
	IdleScanFrequency time.Duration
}

type Req struct {
	connCh chan *TcpConn
	errCh  chan error
}

func NewTcpConnPool(opt *Opt) *TcpConnPool {
	p := &TcpConnPool{
		host:         opt.Host,
		port:         opt.Port,
		maxConns:     opt.MaxConns,
		minIdleConns: opt.MinIdleConns,
		idleTimeout:  opt.IdleTimeout,
		idleConns:    make(map[string]*TcpConn, opt.MaxConns),
		mu:           new(sync.Mutex),
		queue:        make(chan *Req, 10000), //10000 concurrency
		ticker:       *time.NewTicker(opt.IdleScanFrequency),
	}
	for len(p.idleConns) < p.minIdleConns {
		c, err := p.NewConn()
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
	}
}

func (p *TcpConnPool) NewConn() (*TcpConn, error) {
	addr := fmt.Sprintf("%s:%d", p.host, p.port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &TcpConn{
		id:        uuid.New().String(),
		conn:      conn,
		createdAt: time.Now(),
	}, nil
}

func (p *TcpConnPool) Put(c *TcpConn) {
	p.mu.Lock()
	p.idleConns[c.id] = c
	p.mu.Unlock()
}

func (p *TcpConnPool) Get() (*TcpConn, error) {
	p.mu.Lock()
	if len(p.idleConns) > 0 {
		for k, v := range p.idleConns {
			delete(p.idleConns, k)
			p.mu.Unlock()
			return v, nil
		}
	}

	if p.numConns < p.maxConns {
		c, err := p.NewConn()
		if err != nil {
			return nil, err
		}
		p.numConns++
		p.mu.Unlock()
		return c, nil
	}

	//p.numConns == p.maxConns
	// queue a req
	req := &Req{
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
		t := time.After(3 * time.Second)
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
					req.connCh <- p.pop()
					done = true
				}
			}
		}
	}
}

func (p *TcpConnPool) pop() *TcpConn {
	p.mu.Lock()
	for k, v := range p.idleConns {
		delete(p.idleConns, k)
		p.mu.Unlock()
		return v
	}
	return nil
}

func (p *TcpConnPool) release() {
	for range p.ticker.C {
		for _, c := range p.idleConns {
			if time.Now().After(c.createdAt.Add(p.idleTimeout)) {
				p.mu.Lock()
				delete(p.idleConns, c.id)
				p.numConns--
				c.conn.Close()
				p.mu.Unlock()
			}
		}
	}
}

func (p *TcpConnPool) GetIdleConns() int {
	return len(p.idleConns)
}
