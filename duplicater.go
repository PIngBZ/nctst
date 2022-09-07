package nctst

import (
	"log"
	"sync/atomic"
)

type Duplicater struct {
	Output chan *BufItem

	tunnelsListCallback func(uint32) (uint32, []*OuterTunnel)
	input               chan *BufItem
	num                 int32
}

func NewDuplicater(num int, input chan *BufItem, tunnelsListCallback func(uint32) (uint32, []*OuterTunnel)) *Duplicater {
	h := &Duplicater{}

	h.Output = make(chan *BufItem, 4)

	h.tunnelsListCallback = tunnelsListCallback

	h.input = input
	h.num = int32(num)
	go h.daemon()

	return h
}

func (h *Duplicater) SetNum(num int) {
	atomic.StoreInt32(&h.num, int32(num))
	log.Printf("Duplicater SetNum: %d\n", num)
}

func (h *Duplicater) GetNum() int {
	return int(atomic.LoadInt32(&h.num))
}

func (h *Duplicater) daemon() {
	var tunnelsListVer uint32
	var tunnels []*OuterTunnel

	for item := range h.input {
		if item.Size() < 128 {
			sent := false
			cp := item.Copy()
			v, t := h.tunnelsListCallback(tunnelsListVer)
			if t != nil {
				tunnels = t
			}
			tunnelsListVer = v
			for _, tunnel := range tunnels {
				select {
				case tunnel.DirectChan <- cp:
					cp = item.Copy()
					sent = true
				default:
				}
			}
			cp.Release()
			if !sent {
				h.Output <- item
			} else {
				item.Release()
			}
		} else {
			if item.Size() < 256 {
				num := int(atomic.LoadInt32(&h.num))
				for i := 1; i < num; i++ {
					h.Output <- item.Copy()
				}
			}
			h.Output <- item
		}
	}
}
