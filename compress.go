package xmppcore

import (
	"compress/zlib"
	"io"
)

type BuildCompressor func(io.ReadWriter) Compressor

type Compressor interface {
	io.ReadWriter
}

type CompZlib struct {
	rw io.ReadWriter
}

func NewCompZlib(rw io.ReadWriter) *CompZlib {
	return &CompZlib{rw: rw}
}

func (comp *CompZlib) Write(b []byte) (int, error) {
	zw := zlib.NewWriter(comp.rw)
	zw.Flush()
	defer zw.Close()
	return zw.Write(b)
}

func (comp *CompZlib) Read(b []byte) (int, error) {
	zr, err := zlib.NewReader(comp.rw)
	if err != nil {
		return 0, err
	}
	defer zr.Close()
	return zr.Read(b)
}
