package dashboard

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"time"
)

var (
	defaultWaitDuration = time.Duration(3 * time.Second)
)

type ListenerWrap struct {
	listener        net.Listener
	maxWaitDuration time.Duration
	cancel          context.Context
	stop            chan struct{}
	connCount       uint64
	shutdown        uint64
}

func (ln *ListenerWrap) Accept() (net.Conn, error) {
	conn, err := ln.listener.Accept()
	if err != nil {
		return nil, err
	}

	atomic.AddUint64(&ln.connCount, 1)

	return &connWrap{Conn: conn, listener: ln}, nil
}

func (ln *ListenerWrap) Addr() net.Addr {
	return ln.listener.Addr()
}

func (ln *ListenerWrap) Close() error {
	if err := ln.listener.Close(); err != nil {
		return err
	}

	return ln.waitForZeroConns()
}

func (ln *ListenerWrap) waitForZeroConns() error {
	var err error

	atomic.AddUint64(&ln.shutdown, 1)

	if atomic.LoadUint64(&ln.connCount) == 0 {
		close(ln.stop)
		return nil
	}

	select {
	case <-ln.stop:
		err = nil
	case <-time.After(ln.maxWaitDuration):
		err = errors.New("close all established connections failed: timeout")
	case <-ln.cancel.Done():
		err = errors.New("close all established connections failed: be canceled")
	}

	return err
}

func (ln *ListenerWrap) tryCloseConnection() {
	cnt := atomic.AddUint64(&ln.connCount, ^uint64(0))

	if atomic.LoadUint64(&ln.shutdown) != 0 && cnt == 0 {
		close(ln.stop)
	}
}

type connWrap struct {
	net.Conn
	listener *ListenerWrap
}

func (c *connWrap) Close() error {
	err := c.Conn.Close()
	if err != nil {
		return err
	}

	c.listener.tryCloseConnection()

	return nil
}

func NewListenWithTryTime(address string, maxWait time.Duration) (net.Listener, error) {
	ln, err := net.Listen("tcp4", address)
	if err != nil {
		return nil, err
	}
	return &ListenerWrap{
		listener:        ln,
		stop:            make(chan struct{}),
		maxWaitDuration: maxWait,
		cancel:          context.Background(),
	}, nil
}

func NewListenWithContext(ctx context.Context, address string) (net.Listener, error) {
	ln, err := net.Listen("tcp4", address)
	if err != nil {
		return nil, err
	}
	return &ListenerWrap{
		listener:        ln,
		stop:            make(chan struct{}),
		maxWaitDuration: defaultWaitDuration,
		cancel:          ctx,
	}, nil
}
