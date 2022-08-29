package nctst

import "sync/atomic"

type Duplicater struct {
	Output chan *BufItem

	input chan *BufItem
	num   int32
}

func NewDuplicater(num int, input chan *BufItem) *Duplicater {
	h := &Duplicater{}

	h.Output = make(chan *BufItem, 8)

	h.input = input
	h.num = int32(num)
	go h.daemon()

	return h
}

func (h *Duplicater) SetNum(num int) {
	atomic.StoreInt32(&h.num, int32(num))
}

func (h *Duplicater) GetNum() int {
	return int(atomic.LoadInt32(&h.num))
}

func (h *Duplicater) daemon() {
	for item := range h.input {
		num := int(atomic.LoadInt32(&h.num))
		for i := 1; i < num; i++ {
			h.Output <- item.Copy()
		}
		h.Output <- item
	}
}
