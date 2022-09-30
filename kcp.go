package nctst

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"sync"
	"time"

	kcpgo "github.com/xtaci/kcp-go"
)

type Kcp struct {
	ID uint

	session  *kcpgo.UDPSession
	fakeAddr *net.UDPAddr

	InputChan  chan *BufItem
	OutputChan chan *BufItem

	die     chan struct{}
	dieOnce sync.Once

	currentBuf *BufItem

	nextPackageID       uint32
	receivedPackages1   map[uint32]bool
	receivedPackages2   map[uint32]bool
	receivePackageTimes int
}

func NewKcp(connID uint) *Kcp {
	log.Println("Kcp create")

	h := &Kcp{}

	h.ID = connID
	h.fakeAddr, _ = net.ResolveUDPAddr("udp", "127.0.0.1:1234")
	h.session, _ = kcpgo.NewConn3(uint32(connID), h.fakeAddr, nil, 0, 0, h)

	h.session.SetStreamMode(true)
	h.session.SetWriteDelay(false)
	h.session.SetNoDelay(1, 10, 0, 1)
	h.session.SetWindowSize(64, 64)
	h.session.SetMtu(1024 * 8)
	h.session.SetACKNoDelay(true)

	h.InputChan = make(chan *BufItem, KCP_UDP_RECEIVE_BUF_NUM)
	h.OutputChan = make(chan *BufItem, KCP_UDP_SEND_BUF_NUM)

	h.die = make(chan struct{})

	h.receivedPackages1 = make(map[uint32]bool)
	h.receivedPackages2 = make(map[uint32]bool)
	return h
}

func (h *Kcp) Close() error {
	var once bool
	h.dieOnce.Do(func() {
		once = true
	})
	if !once {
		return io.ErrClosedPipe
	}

	close(h.die)

	go func() {
		time.Sleep(time.Second * 60)
		h.session.Close()
	}()

	log.Printf("Kcp %d closed", h.ID)
	return nil
}

func (h *Kcp) Read(buf []byte) (int, error) {
	return h.session.Read(buf)
}

func (h *Kcp) Write(buf []byte) (int, error) {
	return h.session.Write(buf)
}

func (h *Kcp) WriteBuffers(v [][]byte) (n int, err error) {
	return h.session.WriteBuffers(v)
}

// fake udp
func (h *Kcp) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	for h.currentBuf == nil {
		buf := <-h.InputChan
		idx, _ := ReadUInt(buf)
		if _, ok := h.receivedPackages1[idx]; ok {
			buf.Release()
			continue
		}
		if _, ok := h.receivedPackages2[idx]; ok {
			buf.Release()
			continue
		}

		h.receivedPackages1[idx] = true
		h.receivedPackages2[idx] = true
		h.receivePackageTimes++
		if h.receivePackageTimes == 100 {
			h.receivedPackages1 = make(map[uint32]bool)
		} else if h.receivePackageTimes == 200 {
			h.receivedPackages2 = make(map[uint32]bool)
			h.receivePackageTimes = 0
		}
		h.currentBuf = buf
	}

	n, _ = h.currentBuf.Read(p)

	if h.currentBuf.Size() == 0 {
		h.currentBuf.Release()
		h.currentBuf = nil
	}

	return n, h.fakeAddr, nil
}

func (h *Kcp) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	buf := DataBufPool.Get()
	var ubuf [4]byte
	binary.BigEndian.PutUint32(ubuf[:], h.nextPackageID)
	h.nextPackageID++
	buf.AppendData(ubuf[:])
	buf.AppendData(p)
	h.OutputChan <- buf
	return len(p), nil
}

func (h *Kcp) LocalAddr() net.Addr {
	return h.fakeAddr
}

func (h *Kcp) RemoteAddr() net.Addr {
	return h.fakeAddr
}

func (h *Kcp) SetDeadline(t time.Time) error {
	return nil
}

func (h *Kcp) SetReadDeadline(t time.Time) error {
	return nil
}

func (h *Kcp) SetWriteDeadline(t time.Time) error {
	return nil
}
