package xmppcore

import (
	"bytes"
	"io"
	"testing"
)

func TestZLibReadWrite(t *testing.T) {
	var buf bytes.Buffer
	comp := NewCompZlib(&buf)
	str := "hello world\n"
	if l, e := comp.Write([]byte(str)); e != nil {
		t.Fatalf("zlib compress write error: %s\n", e.Error())
	} else if l != len(str) {
		t.Fatalf("zlib compress write lens not equal, wrote %d, require %d\n", l, len(str))
	}
	var resBuf bytes.Buffer
	if _, err := io.Copy(&resBuf, comp); err != nil {
		t.Fatalf("zlib compress read error: %s\n", err.Error())
	}
	res := resBuf.String()
	if str != res {
		t.Fatalf("write and read not equal. write: %s\t, read: %s\n", str, res)
	}
}
