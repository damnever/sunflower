package birpc

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/damnever/sunflower/msg"
	"github.com/damnever/sunflower/pkg/util"
)

const (
	inChSize  = 16
	outChSize = 16
)

type Conn struct {
	conn      net.Conn
	rdTimeout time.Duration
	wrTimeout time.Duration
	closeFlag int32
	closed    chan struct{}
	errCh     chan error
	in        chan interface{}
	out       chan interface{}
}

func NewConn(conn net.Conn, rdTimeout time.Duration, wrTimeout time.Duration) *Conn {
	c := &Conn{
		conn:      conn,
		rdTimeout: rdTimeout,
		wrTimeout: wrTimeout,
		closeFlag: 0,
		closed:    make(chan struct{}),
		errCh:     make(chan error, 2),
		in:        make(chan interface{}, inChSize),
		out:       make(chan interface{}, outChSize),
	}
	return c
}

func (c *Conn) In() <-chan interface{} {
	return c.in
}

func (c *Conn) Out() chan<- interface{} {
	return c.out
}

func (c *Conn) Err() <-chan error {
	return c.errCh
}

func (c *Conn) Go() {
	go c.push()
	go c.pull()
}

func (c *Conn) Close() error {
	if !atomic.CompareAndSwapInt32(&c.closeFlag, 0, 1) {
		return nil
	}
	close(c.closed)
	return c.conn.Close()
}

func (c *Conn) push() {
	defer c.recoverPanic()
PUSH_LOOP:
	for {
		select {
		case <-c.closed:
			break PUSH_LOOP
		case v := <-c.out:
			c.conn.SetWriteDeadline(time.Now().Add(c.wrTimeout))
			util.Must(msg.Write(c.conn, v))
		}
	}
}

func (c *Conn) pull() {
	defer c.recoverPanic()
PULL_LOOP:
	for {
		c.conn.SetReadDeadline(time.Now().Add(c.rdTimeout))
		m, err := msg.Read(c.conn)
		util.Must(err)

		select {
		case <-c.closed:
			break PULL_LOOP
		case c.in <- m:
		}
	}
}

func (c *Conn) recoverPanic() {
	if e := recover(); e != nil {
		c.Close()
		err, ok := e.(error)
		if !ok {
			err = fmt.Errorf("panic: %+v\n", e)
		}
		c.errCh <- err
	}
}
