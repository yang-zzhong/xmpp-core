package xmppcore

import (
	"compress/zlib"
	"io"
	"sync"
)

type BuildCompressor func(io.ReadWriter) Compressor

type Compressor interface {
	io.ReadWriter
}

type CompZlib struct {
	rw io.ReadWriter

	mt sync.Mutex
}

func NewCompZlib(rw io.ReadWriter) *CompZlib {
	return &CompZlib{rw, sync.Mutex{}}
}

func (comp *CompZlib) Write(b []byte) (int, error) {
	comp.mt.Lock()
	defer comp.mt.Unlock()
	zw := zlib.NewWriter(comp.rw)
	defer zw.Close()
	return zw.Write(b)
}

func (comp *CompZlib) Read(b []byte) (int, error) {
	comp.mt.Lock()
	defer comp.mt.Unlock()
	zr, err := zlib.NewReader(comp.rw)
	if err != nil {
		return 0, err
	}
	defer zr.Close()
	return zr.Read(b)
}
