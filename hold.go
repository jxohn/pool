package pool

type Hold interface {
	Close() error
}

type ProxyHold struct {
	pool *Pool
	Hold
}

func (n *ProxyHold) Close() error {
	return n.pool.Put(n.Hold)
}
