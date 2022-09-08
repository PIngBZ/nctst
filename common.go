package nctst

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/xtaci/smux"
)

var (
	DataBufPool = NewPool(DATA_BUF_SIZE)
)

func CheckError(err error) {
	if err != nil {
		log.Printf("%+v\n", err)
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
	streamCopy := func(to, from io.ReadWriteCloser) {
		defer to.Close()
		defer from.Close()

		buf := _copy_buf_pool.Get()
		defer buf.Release()

		io.CopyBuffer(to, from, buf.data)
	}

	go streamCopy(p1, p2)
	streamCopy(p2, p1)
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

func ToUint(data []byte) uint32 {
	return binary.BigEndian.Uint32(data)
}

func WriteData(data []byte, dst io.Writer, written int) (int, error) {
	var err error

	nw, ew := dst.Write(data)
	written += nw

	if ew != nil {
		err = ew
	} else if len(data) != nw {
		err = io.ErrShortWrite
	}
	return written, err
}

func ReadUInt(reader io.Reader) (uint32, error) {
	var buf [4]byte
	_, err := io.ReadFull(reader, buf[:])
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(buf[:]), nil
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
		return "", errors.New("can not read string more than 1M")
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

	if l == 0 || l > MAX_TCP_DATA_INTERNET_LEN {
		return nil, fmt.Errorf("receiveLoop read len error")
	}

	buf := DataBufPool.Get()
	if _, err = buf.ReadNFromReader(reader, int(l)); err != nil {
		buf.Release()
		return nil, err
	}
	return buf, nil
}
