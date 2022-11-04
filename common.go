package nctst

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/xtaci/smux"
)

var (
	DataBufPool          = NewPool(DATA_BUF_SIZE)
	DelayCloseNum uint32 = 0
)

type ContextKey struct {
	Key string
}

type Pair[T, U any] struct {
	First  T
	Second U
}

func CheckError(err error) {
	if err != nil {
		log.Printf("%+v\n%s\n", err, debug.Stack())
		os.Exit(-1)
	}
}

func ToJson(item interface{}) (string, error) {
	ret, err := json.Marshal(item)
	return string(ret), err
}

func OpenLog() {
	f, err := os.OpenFile("flog.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	CheckError(err)
	log.SetOutput(io.MultiWriter(f, os.Stdout))
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func SmuxConfig() *smux.Config {
	smuxConfig := smux.DefaultConfig()
	smuxConfig.Version = 1
	smuxConfig.MaxFrameSize = 1024 * 4
	smuxConfig.MaxReceiveBuffer = 1024 * 1024
	smuxConfig.KeepAliveInterval = time.Second * 30
	smuxConfig.KeepAliveTimeout = time.Hour * 24 * 30

	err := smux.VerifyConfig(smuxConfig)
	CheckError(err)

	return smuxConfig
}

var _copy_buf_pool = NewPool(1024 * 512)

func Transfer(p1, p2 io.ReadWriteCloser) {
	TransferWithCounter(p1, p2, nil, nil)
}

func TransferWithCounter(p1, p2 io.ReadWriteCloser, wl1, wl2 *atomic.Int64) {
	streamCopy := func(to, from io.ReadWriteCloser, l *atomic.Int64) {
		defer to.Close()
		defer from.Close()

		buf := _copy_buf_pool.Get()
		defer buf.Release()

		CopyBufferWithCounter(to, from, buf.data, l)
	}

	go streamCopy(p1, p2, wl1)
	streamCopy(p2, p1, wl2)
}

func CopyBufferWithCounter(dst io.Writer, src io.Reader, buf []byte, wl *atomic.Int64) (written int64, err error) {
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = io.ErrShortWrite
				}
			}
			written += int64(nw)
			if wl != nil {
				wl.Add(int64(nw))
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func Xor(data []byte, key []byte) {
	if len(key) == 0 {
		return
	}
	kn := 0
	for i, v := range data {
		data[i] = v ^ key[kn]
		kn = (kn + 1) % len(key)
	}
}

func ToUint(data []byte) uint32 {
	return binary.BigEndian.Uint32(data)
}

func WriteData(writer io.Writer, data []byte) (int, error) {
	var err error
	var written int

	nw, ew := writer.Write(data)
	written += nw

	if ew != nil {
		err = ew
	} else if len(data) != nw {
		err = io.ErrShortWrite
	}
	return written, err
}

func WriteLData(writer io.Writer, data []byte) error {
	if err := WriteUInt(writer, uint32(len(data))); err != nil {
		return err
	}
	if n, err := writer.Write(data); err != nil {
		return err
	} else if n != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

func ReadUInt(reader io.Reader) (uint32, error) {
	var buf [4]byte
	_, err := io.ReadFull(reader, buf[:])
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(buf[:]), nil
}

func ReadUInt64(reader io.Reader) (uint64, error) {
	var buf [8]byte
	_, err := io.ReadFull(reader, buf[:])
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(buf[:]), nil
}

func WriteUInt(writer io.Writer, n uint32) error {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], n)
	if n, err := writer.Write(buf[:]); err != nil {
		return err
	} else if n != 4 {
		return io.ErrShortWrite
	}
	return nil
}

func WriteString(writer io.Writer, s string) error {
	if n, err := writer.Write([]byte(s)); err != nil {
		return err
	} else if n != len(s) {
		return io.ErrShortWrite
	}
	return nil
}

func WriteLString(writer io.Writer, s string) error {
	if err := WriteUInt(writer, uint32(len(s))); err != nil {
		return err
	}
	if n, err := writer.Write([]byte(s)); err != nil {
		return err
	} else if n != len(s) {
		return io.ErrShortWrite
	}
	return nil
}

func ReadLString(reader io.Reader) (string, error) {
	l, err := ReadUInt(reader)
	if err != nil {
		return "", err
	}
	if l > 1024*1024 {
		return "", errors.New("ReadLString can not read string more than 1M")
	}

	buf := make([]byte, l)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func ReadLBuf(reader io.Reader) (*BufItem, error) {
	l, err := ReadUInt(reader)
	if err != nil {
		return nil, err
	}

	if l == 0 || l > DATA_BUF_SIZE {
		return nil, fmt.Errorf("ReadLBuf read len error")
	}

	buf := DataBufPool.Get()
	if _, err = buf.ReadNFromReader(reader, int(l)); err != nil {
		buf.Release()
		return nil, err
	}
	return buf, nil
}

func HashPassword(username, password string) string {
	p := sha1.Sum([]byte(password))
	k := append([]byte(username), p[:]...)
	m := md5.Sum([]byte(k))
	return string(hex.EncodeToString(m[:]))
}

func DelayClose(conn io.Closer) {
	if atomic.LoadUint32(&DelayCloseNum) > 1000 {
		conn.Close()
		return
	}

	atomic.AddUint32(&DelayCloseNum, 1)
	go func() {
		time.Sleep(time.Second * time.Duration(60+rand.Intn(60)))
		conn.Close()
		atomic.AddUint32(&DelayCloseNum, ^uint32(0))
	}()
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func SplitHostPort(addr string) (string, int, error) {
	host, portS, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}

	port, err := strconv.Atoi(portS)
	if err != nil {
		return "", 0, err
	}

	return host, port, nil
}

func HttpGetString(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HttpGetString Code=%d %s", resp.StatusCode, url)
	}

	return string(body), nil
}
