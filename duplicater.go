package nctst

import (
	"log"
	"sync"
	"sync/atomic"
)

type Duplicater struct {
	Output chan *BufItem

	tunnelsListCallback func(uint32) (uint32, []*OuterTunnel)
	input               chan *BufItem
	num                 int32

	tunnelsListVer uint32
	tunnels        []*OuterTunnel

	die     chan struct{}
	dieOnce sync.Once
}

func NewDuplicater(num int, input chan *BufItem, tunnelsListCallback func(uint32) (uint32, []*OuterTunnel)) *Duplicater {
	h := &Duplicater{}
	h.die = make(chan struct{})

	h.Output = make(chan *BufItem, 4)

	h.tunnelsListCallback = tunnelsListCallback

	h.input = input
	h.num = int32(num)
	go h.daemon()

	log.Println("Duplicater.New")
	return h
}

func (h *Duplicater) Close() {
	var once bool
	h.dieOnce.Do(func() {
		close(h.die)
		once = true
	})

	if !once {
		return
	}

	log.Println("Duplicater.Close")
}

func (h *Duplicater) SetNum(num int) {
	atomic.StoreInt32(&h.num, int32(num))
	log.Printf("Duplicater SetNum: %d\n", num)
}

func (h *Duplicater) GetNum() int {
	return int(atomic.LoadInt32(&h.num))
}

func (h *Duplicater) daemon() {
	for {
		select {
		case <-h.die:
			return
		case item := <-h.input:
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
				select {
				case <-h.die:
					cp.Release()
					item.Release()
					return
				case h.Output <- item:
				}

				item = nil
			}
			if cp != nil {
				cp.Release()
			}
			if item != nil {
				item.Release()
			}
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
