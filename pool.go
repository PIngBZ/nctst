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

type IMetaCopy interface {
	MetaCopy() interface{}
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

func (h *BufItem) SetFromReader(src io.Reader) (int, error) {
	h.start = 0
	n, err := src.Read(h.data)
	h.size = n
	return n, err
}

func (h *BufItem) SetNFromReader(src io.Reader, n int) (int, error) {
	if n > h.FreeSpace() {
		log.Println("BufItem ReadNFromReader no enough space", n, h.FreeSpace())
		return 0, io.ErrShortBuffer
	}
	h.start = 0
	_, err := io.ReadFull(src, h.data[:n])
	h.size = n
	return n, err
}

func (h *BufItem) SetAllFromReader(src io.Reader) (int, error) {
	h.Reset()
	buf, err := io.ReadAll(src)
	if err != nil {
		return 0, err
	}
	return h.AppendBytes(buf)
}

func (h *BufItem) AppendItem(data *BufItem) (int, error) {
	if h.FreeSpace() < data.Size() {
		log.Println("BufItem Append no enough space", data.Size(), h.FreeSpace())
		return 0, io.ErrShortBuffer
	}
	h.AppendBytes(data.Data())
	return data.Size(), nil
}

func (h *BufItem) AppendBytes(data []byte) (int, error) {
	if h.FreeSpace() < len(data) {
		log.Println("BufItem AppendData no enough space", len(data), h.FreeSpace())
		return 0, io.ErrShortBuffer
	}
	h.size += copy(h.data[h.start+h.size:], data)
	return len(data), nil
}

func (h *BufItem) AppendFromReader(src io.Reader) (n int, err error) {
	n, err = src.Read(h.data[h.start+h.size:])
	h.size += n
	return
}

func (h *BufItem) AppendNFromReader(src io.Reader, n int) (int, error) {
	if n > h.FreeSpace() {
		log.Println("BufItem AppendNFromReader no enough space", n, h.FreeSpace())
		return 0, io.ErrShortBuffer
	}
	_, err := io.ReadFull(src, h.data[h.start+h.size:h.start+h.size+n])
	if err != nil {
		return 0, err
	}
	h.size += n
	return n, nil
}

func (h *BufItem) AppendAllFromReader(src io.Reader) (int, error) {
	buf, err := io.ReadAll(src)
	if err != nil {
		return 0, err
	}
	return h.AppendBytes(buf)
}

func (h *BufItem) Write(p []byte) (n int, err error) {
	if h.FreeSpace() < len(p) {
		return 0, io.ErrShortBuffer
	}
	h.AppendBytes(p)
	return len(p), nil
}

func (h *BufItem) Copy() *BufItem {
	n := h.pool.Get()
	copy(n.data[:h.size], h.data[:h.size])
	n.size = h.size
	if c, ok := h.meta.(IMetaCopy); ok {
		n.meta = c.MetaCopy()
	} else {
		n.meta = h.meta
	}
	return n
}

func (h *BufItem) Reset() *BufItem {
	h.start = 0
	h.size = 0
	h.meta = nil
	return h
}

func (h *BufItem) ResetWithData(data []byte) error {
	if h.Cap() < len(data) {
		return io.ErrShortBuffer
	}
	h.start = 0
	h.size = copy(h.data, data)
	return nil
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
