package pool

import (
	"errors"
	"net"
	"runtime"
	"sync"
	"time"
)

type connPool struct {
	factory            ConnFactory
	initCap            int
	maxCap             int
	idleTimeout        time.Duration
	idleCheckFrequency time.Duration
	idleConns          map[int64]*Conn
	numConns           int
	mu                 *sync.Mutex
	reqQueue           chan *ConnReq
	ticker             *time.Ticker
	quit               chan struct{}
}

type Option struct {
	Factory            ConnFactory
	InitCap            int
	MaxCap             int
	IdleTimeout        time.Duration
	IdleCheckFrequency time.Duration
}

func (opt *Option) init() {
	if opt.MaxCap == 0 {
		opt.MaxCap = 10 * runtime.NumCPU()
	}
	if opt.IdleTimeout == 0 {
		opt.IdleTimeout = 5 * time.Minute
	}
	if opt.IdleCheckFrequency == 0 {
		opt.IdleCheckFrequency = time.Minute
	}
	if opt.Factory == nil || opt.InitCap < 0 || opt.MaxCap < 0 || opt.InitCap > opt.MaxCap {
		panic("invalid capacity settings")
	}
}

type ConnFactory func() (net.Conn, error)

type ConnReq struct {
	connCh chan *Conn
}

func NewConnPool(opt *Option) *connPool {
	opt.init()
	p := &connPool{
		factory:            opt.Factory,
		maxCap:             opt.MaxCap,
		initCap:            opt.InitCap,
		idleTimeout:        opt.IdleTimeout,
		idleCheckFrequency: opt.IdleCheckFrequency,
		idleConns:          make(map[int64]*Conn, opt.MaxCap),
		mu:                 new(sync.Mutex),
		reqQueue:           make(chan *ConnReq, 10000),
		ticker:             time.NewTicker(opt.IdleCheckFrequency),
		quit:               make(chan struct{}, 1),
	}

	for i := 0; i < p.initCap; i++ {
		c, err := p.newConn()
		if err != nil {
			panic(err)
		}
		p.idleConns[c.id] = c
		p.numConns++
	}

	go p.handleReqQueue()

	go p.release()

	return p
}

func (p *connPool) Close() {
	p.quit <- struct{}{}
	close(p.reqQueue)
	p.ticker.Stop()
	for _, c := range p.idleConns {
		p.ReleaseConn(c)
	}
}

func (p *connPool) newConn() (*Conn, error) {
	conn, err := p.factory()
	if err != nil {
		return nil, err
	}
	return &Conn{
		id:   time.Now().UnixNano(),
		conn: conn,
		p:    p,
	}, nil
}

func (p *connPool) GetConn() (*Conn, error) {
	p.mu.Lock()
	if len(p.idleConns) > 0 {
		for _, c := range p.idleConns {
			delete(p.idleConns, c.id)
			p.mu.Unlock()
			return c, nil
		}
	}

	if p.numConns < p.maxCap {
		c, err := p.newConn()
		if err != nil {
			return nil, err
		}
		p.numConns++
		p.mu.Unlock()
		return c, nil
	}

	//p.openConns == p.MaxCap
	if len(p.reqQueue) >= 10000 {
		p.mu.Unlock()
		return nil, errors.New("request abandoned")
	}

	req := &ConnReq{
		connCh: make(chan *Conn, 1),
	}
	p.reqQueue <- req

	p.mu.Unlock()

	// blocked
	c := <-req.connCh
	return c, nil
}

func (p *connPool) handleReqQueue() {
	for req := range p.reqQueue {
		p.mu.Lock()
		if len(p.idleConns) > 0 {
			p.mu.Unlock()
			req.connCh <- p.popConn()
			continue
		}

		//code here is a little bit tricky
		//quit twice in case of writing to a closed channel
		select {
		case <-p.quit:
			return
		case <-time.After(10 * time.Microsecond):
			return
		default:
			p.reqQueue <- req
			p.mu.Unlock()
			select {
			case <-p.quit:
				return
			case <-time.After(10 * time.Microsecond):
				return
			}
		}
	}
}

func (p *connPool) popConn() *Conn {
	p.mu.Lock()
	for _, c := range p.idleConns {
		delete(p.idleConns, c.id)
		p.mu.Unlock()
		return c
	}
	return nil
}

func (p *connPool) release() {
	for range p.ticker.C {
		for _, c := range p.idleConns {
			if time.Now().After(c.homedAt.Add(p.idleTimeout)) {
				p.ReleaseConn(c)
			}
		}
	}
}

func (p *connPool) ReleaseConn(c *Conn) {
	p.mu.Lock()
	delete(p.idleConns, c.id)
	p.numConns--
	c.conn.Close()
	p.mu.Unlock()
}

func (p *connPool) GetIdleConns() int {
	return len(p.idleConns)
}
