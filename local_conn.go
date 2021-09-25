package xmppcore

import (
	"bytes"
	"crypto/tls"
	"github.com/google/uuid"
	"io"
	"net"
	"os"
	"time"
)

type LocalConnAddr struct {
	fakePort string
}

func NewLocalConnAddr(port string) *LocalConnAddr {
	return &LocalConnAddr{port}
}

func (addr *LocalConnAddr) Network() string {
	return "local"
}

func (addr *LocalConnAddr) String() string {
	return addr.fakePort
}

// a conn mainly for test
type LocalConn struct {
	id                   string
	comming              chan []byte
	going                chan []byte
	readDeadline         time.Time
	readDeadlineEnabled  bool
	writeDeadline        time.Time
	writeDeadlineEnabled bool
	buf                  *bytes.Buffer
	localAddr            net.Addr
	remoteAddr           net.Addr
}

func NewLocalConnPair(oneAddr, twoAddr net.Addr) []*LocalConn {
	one := make(chan []byte)
	two := make(chan []byte)
	return []*LocalConn{
		NewLocalConn(one, two, oneAddr, twoAddr),
		NewLocalConn(two, one, twoAddr, oneAddr),
	}
}

func NewLocalConn(comming, going chan []byte, localAddr, remoteAddr net.Addr) *LocalConn {
	bs := []byte{}
	return &LocalConn{
		id:                   uuid.New().String(),
		comming:              comming,
		going:                going,
		buf:                  bytes.NewBuffer(bs),
		readDeadlineEnabled:  false,
		writeDeadlineEnabled: false,
		localAddr:            localAddr,
		remoteAddr:           remoteAddr}
}

func (lc *LocalConn) Read(b []byte) (n int, err error) {
	if !lc.readDeadlineEnabled {
		c := copy(b, <-lc.comming)
		return c, nil
	}
	du := lc.readDeadline.Sub(time.Now())
	to := make(chan bool)
	timer := time.AfterFunc(du, func() {
		to <- true
	})
	select {
	case tb := <-lc.comming:
		timer.Stop()
		return copy(b, tb), nil
	case <-to:
		return 0, os.ErrDeadlineExceeded
	}
}

func (lc *LocalConn) Write(b []byte) (n int, err error) {
	du := lc.readDeadline.Sub(time.Now())
	to := make(chan bool)
	timer := time.AfterFunc(du, func() {
		to <- true
	})
	select {
	case <-to:
		return 0, os.ErrDeadlineExceeded
	case lc.going <- b:
		timer.Stop()
	}
	return len(b), nil
}

func (lc *LocalConn) Close() error {
	defer func() {
		if e := recover(); e != nil {
			// already closed
		}
	}()
	close(lc.comming)
	close(lc.going)
	return nil
}

func (lc *LocalConn) LocalAddr() net.Addr {
	return lc.localAddr
}

func (lc *LocalConn) RemoteAddr() net.Addr {
	return lc.remoteAddr
}

func (lc *LocalConn) SetDeadline(t time.Time) error {
	lc.SetReadDeadline(t)
	lc.SetWriteDeadline(t)
	return nil
}

func (lc *LocalConn) SetReadDeadline(t time.Time) error {
	lc.readDeadline = t
	lc.readDeadlineEnabled = true
	return nil
}

func (lc *LocalConn) SetWriteDeadline(t time.Time) error {
	lc.writeDeadline = t
	lc.writeDeadlineEnabled = true
	return nil
}

func (lc *LocalConn) StartTLS(*tls.Config) error {
	return nil
}

func (lc *LocalConn) BindTlsUnique(w io.Writer) error {
	_, err := w.Write([]byte(lc.id))
	return err
}
