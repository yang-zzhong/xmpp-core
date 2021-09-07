package xmppcore

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestLocalConn(t *testing.T) {
	pair := NewLocalConnPair(NewLocalConnAddr(":1"), NewLocalConnAddr(":2"))
	str := "hello world"
	time.AfterFunc(time.Second*2, func() {
		pair[0].Write([]byte(str))
	})
	bs := make([]byte, 128)
	l, err := pair[1].Read(bs)
	if err != nil {
		t.Fatalf("read error: %s", err.Error())
	}
	bstr := string(bs[:l])
	if bstr != str {
		t.Fatalf("read error: read not equal what it's written")
	}
}

func TestLocalConnReadDeadline(t *testing.T) {
	pair := NewLocalConnPair(NewLocalConnAddr(":1"), NewLocalConnAddr(":2"))
	pair[0].SetReadDeadline(time.Now().Add(time.Second))
	bs := make([]byte, 128)
	_, err := pair[0].Read(bs)
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("read deadline error: %s", err.Error())
	}
}

func TestLocalConnWriteDeadline(t *testing.T) {
	pair := NewLocalConnPair(NewLocalConnAddr(":1"), NewLocalConnAddr(":2"))
	pair[0].SetWriteDeadline(time.Now().Add(time.Second))
	str := "hello world"
	_, err := pair[0].Write([]byte(str))
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("read deadline error: %s", err.Error())
	}
}

func TestLocalConnClose(t *testing.T) {
	pair := NewLocalConnPair(NewLocalConnAddr(":1"), NewLocalConnAddr(":2"))
	pair[0].Close()
	pair[1].Close()
}
