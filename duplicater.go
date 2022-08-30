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
	total := 0
	h.tunnels.Range(func(key, value any) bool {
		if value.(*OuterTunnel).IsAlive() {
			total += 1
		}
		return true
	})

	num = Max(Min(total, num), 1)
	atomic.StoreInt32(&h.num, int32(num))
	log.Printf("Duplicater SetNum: %d", num)
}

func (h *Duplicater) GetNum() int {
	return int(atomic.LoadInt32(&h.num))
}

func (h *Duplicater) daemon() {
	var alives []*OuterTunnel
	for {
		ticker := time.NewTicker(time.Second)

		select {
		case buf := <-h.sendChan:
			if len(alives) == 0 {
				buf.Release()
				continue
			}

			num := Max(Min(int(atomic.LoadInt32(&h.num)), len(alives)), 1)

			var cp *BufItem
			if num == 1 {
				cp = buf
				buf = nil
			} else {
				cp = buf.Copy()
			}
			for _, tunnel := range alives {
				if tunnel.TrySend(cp) {
					cp = nil
					num -= 1

					if num == 0 {
						break
					}

					if num == 1 {
						cp = buf
						buf = nil
					} else {
						cp = buf.Copy()
					}
				}
			}

			if buf != nil {
				buf.Release()
			}
			if cp != nil {
				cp.Release()
			}
		case <-ticker.C:
			alives = make([]*OuterTunnel, 0)
			h.tunnels.Range(func(key, value any) bool {
				tunnel := value.(*OuterTunnel)
				if tunnel.IsAlive() {
					alives = append(alives, tunnel)
				}
				return true
			})

			sort.SliceStable(alives, func(i, j int) bool {
				return alives[i].Ping < alives[j].Ping
			})
		}
	}

}
