package xmppcore

import (
	"compress/zlib"
	"io"
	"sync"
)

type BuildCompressor func(io.ReadWriter) Compressor

type Compressor interface {
	io.ReadWriteCloser
}

type CompZlib struct {
	rw io.ReadWriter

	zr io.ReadCloser
	zw io.Writer

	mt sync.Mutex
}

func NewCompZlib(rw io.ReadWriter) *CompZlib {
	return &CompZlib{rw, nil, nil, sync.Mutex{}}
}

func (comp *CompZlib) Write(b []byte) (int, error) {
	comp.mt.Lock()
	defer comp.mt.Unlock()
	if comp.zw == nil {
		comp.zw = zlib.NewWriter(comp.rw)
	}
	return comp.zw.Write(b)
}

func (comp *CompZlib) Read(b []byte) (int, error) {
	comp.mt.Lock()
	defer comp.mt.Unlock()
	if comp.zr == nil {
		var err error
		comp.zr, err = zlib.NewReader(comp.rw)
		if err != nil {
			return 0, err
		}
	}
	return comp.zr.Read(b)
}

func (comp *CompZlib) Close() error {
	comp.mt.Lock()
	defer comp.mt.Unlock()
	if comp.zr != nil {
		return comp.zr.Close()
	}
	return nil
}
