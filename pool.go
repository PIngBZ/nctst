package nctst

import (
	"errors"
	"io"
	"log"
	"sync"
)

type Pool struct {
	size int
	pool sync.Pool
}

type BufItem struct {
	pool  *Pool
	data  []byte
	start int
	size  int
	meta  interface{}
}

func NewBufItem(pool *Pool, cap int) *BufItem {
	return &BufItem{pool, make([]byte, cap), 0, 0, nil}
}

func (h *BufItem) Cap() int {
	return len(h.data)
}

func (h *BufItem) FreeSpace() int {
	return h.Cap() - h.start - h.size
}

func (h *BufItem) Size() int {
	return h.size
}

func (h *BufItem) AddSize(n int) *BufItem {
	h.size += n
	if h.start+h.size > h.Cap() {
		h.size = h.Cap() - h.start
		log.Println("BufItem AddSize out of size")
	}
	return h
}

func (h *BufItem) OriginBuf() []byte {
	return h.data
}

func (h *BufItem) Data() []byte {
	return h.data[h.start : h.start+h.size]
}

func (h *BufItem) SetMetaData(meta interface{}) {
	h.meta = meta
}

func (h *BufItem) MetaData() interface{} {
	return h.meta
}

func (h *BufItem) Read(buf []byte) (int, error) {
	n := copy(buf, h.Data())
	h.start += n
	h.size -= n
	return n, nil
}

func (h *BufItem) ReadFromReader(src io.Reader) (int, error) {
	h.start = 0
	n, err := src.Read(h.data)
	h.size = n
	return n, err
}

func (h *BufItem) ReadNFromReader(src io.Reader, n int) (int, error) {
	if n > h.FreeSpace() {
		log.Println("BufItem ReadNFromReader no enough space", n, h.FreeSpace())
	}
	h.start = 0
	_, err := io.ReadFull(src, h.data[:n])
	h.size = n
	return n, err
}

func (h *BufItem) AppendData(data []byte) *BufItem {
	if len(data) > h.FreeSpace() {
		log.Println("BufItem AppendData no enough space", len(data), h.FreeSpace())
	}
	h.size += copy(h.data[h.start+h.size:], data)
	return h
}

func (h *BufItem) AppendFromReader(src io.Reader) error {
	n, err := src.Read(h.data[h.start+h.size:])
	h.size += n
	return err
}

func (h *BufItem) AppendNFromReader(src io.Reader, n int) error {
	if n > h.FreeSpace() {
		log.Println("BufItem AppendNFromReader no enough space", n, h.FreeSpace())
	}
	_, err := io.ReadFull(src, h.data[h.start+h.size:h.start+h.size+n])
	if err != nil {
		return err
	}
	h.size += n
	return nil
}

func (h *BufItem) Copy() *BufItem {
	n := h.pool.Get()
	copy(n.data[:h.size], h.data[:h.size])
	n.size = h.size
	return n
}

func (h *BufItem) Reset() *BufItem {
	h.start = 0
	h.size = 0
	h.meta = nil
	return h
}

func (h *BufItem) ResetWithData(data []byte) *BufItem {
	h.start = 0
	h.size = copy(h.data, data)
	return h
}

func (h *BufItem) Release() {
	h.pool.Put(h)
}

var poolMap sync.Map

func NewPool(size int) *Pool {
	if pool, loaded := poolMap.Load(size); loaded {
		return pool.(*Pool)
	}

	c := &Pool{}
	c.size = size

	if pool, loaded := poolMap.LoadOrStore(size, c); loaded {
		return pool.(*Pool)
	}

	return c
}

func (c *Pool) Get() *BufItem {
	x := c.pool.Get()
	if x == nil {
		return NewBufItem(c, c.size)
	}
	return x.(*BufItem)
}

func (c *Pool) Put(item *BufItem) {
	if len(item.data) != c.size {
		return
	}
	if item.pool != c {
		CheckError(errors.New("pool type error"))
	}
	item.Reset()
	c.pool.Put(item)
}
