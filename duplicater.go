package nctst

import (
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Duplicater struct {
	sendChan chan *BufItem
	tunnels  *sync.Map

	num int32
}

func NewDuplicater(num int, sendChan chan *BufItem, tunnels *sync.Map) *Duplicater {
	h := &Duplicater{}
	h.sendChan = sendChan
	h.tunnels = tunnels
	h.num = int32(num)

	go h.daemon()

	return h
}

func (h *Duplicater) SetNum(num int) {
	atomic.StoreInt32(&h.num, int32(num))
	log.Printf("Duplicater SetNum: %d", num)
}

func (h *Duplicater) GetNum() int {
	return int(atomic.LoadInt32(&h.num))
}

func (h *Duplicater) daemon() {
	var conns []*OuterTunnel
	var aliveNum int
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case buf := <-h.sendChan:
			num := Max(Min(int(atomic.LoadInt32(&h.num)), aliveNum), 1)
			sent := 0

			var cp *BufItem
			if num == 1 {
				cp = buf
				buf = nil
			} else {
				cp = buf.Copy()
			}

			for _, tunnel := range conns {
				if tunnel.TrySend(cp) {
					cp = nil
					sent++
					if sent == num {
						break
					}

					if sent == num-1 {
						cp = buf
						buf = nil
					} else {
						cp = buf.Copy()
					}
				}
			}

			if sent == 0 {
				// wait send
				cp = nil
			}

			if buf != nil {
				buf.Release()
			}
			if cp != nil {
				cp.Release()
			}

		case <-ticker.C:
			conns = make([]*OuterTunnel, 0)
			aliveNum = 0
			h.tunnels.Range(func(key, value any) bool {
				tunnel := value.(*OuterTunnel)
				if !tunnel.IsAlive() {
					tunnel.Ping = 1000000
				} else {
					aliveNum++
				}
				conns = append(conns, tunnel)
				return true
			})

			sort.SliceStable(conns, func(i, j int) bool {
				return conns[i].Ping < conns[j].Ping
			})
		}
	}

}
