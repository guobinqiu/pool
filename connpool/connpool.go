package connpool

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

type ConnPool struct {
	host         string
	port         int
	maxConns     int
	minIdleConns int
	idleTimeout  time.Duration
	IdleConns    map[string]*Conn
	numConns     int
	mu           *sync.Mutex
	connReqs     chan *Req
	ticker       time.Ticker
}

type Conn struct {
	id        string
	Conn      net.Conn
	createdAt time.Time
}

type Opt struct {
	Host                 string
	Port                 int
	MaxConns             int
	MinIdleConns         int
	IdleTimeout          time.Duration
	IdleTimeoutFrequency time.Duration
}

type Req struct {
	connCh chan *Conn
	errCh  chan error
}

func NewConnPool(opt *Opt) *ConnPool {
	p := &ConnPool{
		host:         opt.Host,
		port:         opt.Port,
		maxConns:     opt.MaxConns,
		minIdleConns: opt.MinIdleConns,
		idleTimeout:  opt.IdleTimeout,
		IdleConns:    make(map[string]*Conn, opt.MaxConns),
		mu:           new(sync.Mutex),
		connReqs:     make(chan *Req, 10000), //10000 concurrency
		ticker:       *time.NewTicker(opt.IdleTimeoutFrequency),
	}
	p.keepMinIdleConns()
	go p.handleConnReqs()
	go p.removeIdleConns()
	return p
}

func (p *ConnPool) Close() {
	close(p.connReqs)
	p.ticker.Stop()
	for _, conn := range p.IdleConns {
		conn.Conn.Close()
	}
}

func (p *ConnPool) keepMinIdleConns() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(p.IdleConns) < p.minIdleConns {
		conn, err := p.NewConn()
		if err == nil {
			p.IdleConns[conn.id] = conn
			p.numConns++
		}
	}
}

func (p *ConnPool) NewConn() (*Conn, error) {
	addr := fmt.Sprintf("%s:%d", p.host, p.port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Conn{
		id:        fmt.Sprintf("%v", time.Now().UnixNano()),
		Conn:      conn,
		createdAt: time.Now(),
	}, nil
}

func (p *ConnPool) Put(conn *Conn) {
	p.mu.Lock()
	p.IdleConns[conn.id] = conn
	p.mu.Unlock()
}

func (p *ConnPool) Get() (*Conn, error) {
	p.mu.Lock()
	if len(p.IdleConns) > 0 {
		for k, v := range p.IdleConns {
			delete(p.IdleConns, k)
			p.mu.Unlock()
			return v, nil
		}
	}

	if p.numConns < p.maxConns {
		conn, err := p.NewConn()
		if err != nil {
			return nil, err
		}
		p.numConns++
		p.mu.Unlock()
		return conn, nil
	}

	//p.numConns == p.maxConns
	// queue a req
	req := &Req{
		connCh: make(chan *Conn, 1),
		errCh:  make(chan error, 1),
	}
	p.connReqs <- req

	p.mu.Unlock()

	// blocked
	select {
	case conn := <-req.connCh:
		return conn, nil
	case err := <-req.errCh:
		return nil, err
	}
}

func (p *ConnPool) handleConnReqs() {
	for req := range p.connReqs {
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
				if len(p.IdleConns) > 0 {
					req.connCh <- p.pop()
					done = true
				}
			}
		}
	}
}

func (p *ConnPool) pop() *Conn {
	p.mu.Lock()
	for k, v := range p.IdleConns {
		delete(p.IdleConns, k)
		p.mu.Unlock()
		return v
	}
	return nil
}

func (p *ConnPool) removeIdleConns() {
	for range p.ticker.C {
		for _, conn := range p.IdleConns {
			if time.Now().After(conn.createdAt.Add(p.idleTimeout)) {
				p.mu.Lock()
				delete(p.IdleConns, conn.id)
				p.numConns--
				p.mu.Unlock()
				conn.Conn.Close()
			}
		}
	}
}

func (p *ConnPool) GetIdleConns() int {
	return len(p.IdleConns)
}
