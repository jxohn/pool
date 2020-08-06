package pool

import (
	"sync"

	"github.com/pkg/errors"
)

type Producer interface {
	Produce() (one Hold, err error)
}

type Pool struct {
	sync.RWMutex
	producer Producer
	holds    chan Hold
}

func NewPool(iniCap int, maxCap int, producer Producer) (pool *Pool, err error) {

	pool = &Pool{
		producer: producer,
		holds:    make(chan Hold, maxCap),
	}

	for i := 0; i < iniCap; i++ {
		one, err := producer.Produce()
		if err != nil {
			pool.Close()
			return nil, errors.Wrap(err, "init error")
		}
		pool.holds <- one
	}
	return pool, nil
}

// Get : fetch one hold from holds, if not, produce one...
func (p *Pool) Get() (one Hold, err error) {
	if p.holds == nil {
		return nil, errors.New("holder holds nothing")
	}

	p.RLock()
	defer p.RUnlock()
	if p.holds == nil {
		return nil, errors.New("holder has been closed")
	}

	select {
	case one = <-p.holds:
		return p.wrapHold(one), nil
	default:
		// if holds don't return one hold, produce one and wrap it...
		// don't worry about producing more than maxCap holds, because Put will recycle or drop it.
		produce, err := p.producer.Produce()
		if err != nil {
			return nil, err
		}
		return p.wrapHold(produce), nil
	}
}

// Put : reuse of hold
func (p *Pool) Put(one Hold) error {
	if p.holds == nil {
		return errors.New("holds is nil")
	}

	// RW Lock to protect holds from closing but Put try to put one hold into holds
	p.RLock()
	defer p.RUnlock()
	if p.holds == nil {
		return nil // holds has been closed
	}

	select {
	case p.holds <- one:
		return nil
	default:
		// once the holds is full, drop the hold...
		_ = one.Close()
		return nil
	}
}

func (p *Pool) wrapHold(one Hold) (proxy Hold) {
	return &ProxyHold{
		pool: p,
		Hold: one,
	}
}

// Close close the holds channel and holds
func (p *Pool) Close() {
	p.Lock()
	defer p.Unlock()
	p.producer = nil
	if p.holds == nil {
		return
	}

	for {
		if len(p.holds) == 0 {
			break
		}
		one := <-p.holds
		_ = one.Close()
	}

	close(p.holds)
}
