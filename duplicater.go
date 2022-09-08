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

	tunnelsListVer uint32
	tunnels        []*OuterTunnel
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

	for item := range h.input {
		if item.Size() < 128 {
			h.updateTunnelsList()
			sent := false
			cp := item.Copy()
			for i, tunnel := range h.tunnels {
				select {
				case tunnel.DirectChan <- cp:
					if i == len(h.tunnels)-1 {
						cp = nil
					} else if i == len(h.tunnels)-2 {
						cp = item
						item = nil
					} else {
						cp = item.Copy()
					}
					sent = true
				default:
				}
			}
			if !sent {
				h.Output <- item
				item = nil
			}
			if cp != nil {
				cp.Release()
			}
			if item != nil {
				item.Release()
			}
		} else {
			if item.Size() < 512 {
				num := int(atomic.LoadInt32(&h.num))
				for i := 1; i < num; i++ {
					h.Output <- item.Copy()
				}
			}
			h.Output <- item
		}
	}
}

func (h *Duplicater) updateTunnelsList() {
	v, t := h.tunnelsListCallback(h.tunnelsListVer)
	if t != nil {
		h.tunnels = t
	}
	h.tunnelsListVer = v
}
