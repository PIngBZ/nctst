package nctst

import (
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/snappy"
	"github.com/pkg/errors"
)

type CompStream struct {
	conn io.ReadWriteCloser
	w    *snappy.Writer
	r    *snappy.Reader
	l    sync.Mutex
	n    atomic.Int32

	die     chan struct{}
	dieOnce sync.Once
}

func NewCompStream(conn io.ReadWriteCloser) *CompStream {
	c := new(CompStream)
	c.conn = conn
	c.w = snappy.NewBufferedWriter(conn)
	c.r = snappy.NewReader(conn)
	c.die = make(chan struct{})
	go c.daemon()
	return c
}

func (c *CompStream) Read(p []byte) (n int, err error) {
	if c.IsClosed() {
		return 0, io.ErrClosedPipe
	}
	return c.r.Read(p)
}

func (c *CompStream) Write(p []byte) (n int, err error) {
	return c.WriteBuffers([][]byte{p})
}

func (c *CompStream) WriteBuffers(v [][]byte) (n int, err error) {
	if c.IsClosed() {
		return 0, io.ErrClosedPipe
	}

	c.l.Lock()
	defer c.l.Unlock()

	var total int
	for _, vv := range v {
		if _, err := c.w.Write(vv); err != nil {
			return total, errors.WithStack(err)
		}
		n := len(vv)
		total += n
		c.n.Add(int32(n))
	}
	return total, err
}

func (c *CompStream) Close() error {
	var once bool
	c.dieOnce.Do(func() {
		close(c.die)
		once = true
	})

	if once {
		return c.conn.Close()
	} else {
		return io.ErrClosedPipe
	}
}

func (c *CompStream) IsClosed() bool {
	select {
	case <-c.die:
		return true
	default:
		return false
	}
}

func (c *CompStream) daemon() {
	ticker := time.NewTicker(time.Millisecond * 10)
	for {
		select {
		case <-c.die:
			return
		case <-ticker.C:
			if c.n.Load() > 0 {
				c.l.Lock()
				c.w.Flush()
				c.n.Store(0)
				c.l.Unlock()
			}
		}
	}
}
