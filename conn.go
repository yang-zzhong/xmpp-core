package xmppcore

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"time"
)

var (
	ErrBindTlsUniqueNotSupported = errors.New("bind tls unique not supported")
)

type Conn interface {
	net.Conn
	StartTLS(*tls.Config)
	BindTlsUnique(io.Writer) error
	StartCompress(BuildCompressor)
}

type TcpConn struct {
	underlying net.Conn
	comp       Compressor
	isClient   bool
}

func NewTcpConn(underlying net.Conn, isClient bool) *TcpConn {
	_, isTcp := underlying.(*net.TCPConn)
	_, isTls := underlying.(*tls.Conn)
	if !isTcp && !isTls {
		panic("not a tcp conn nor a tls conn")
	}
	return &TcpConn{underlying, nil, isClient}
}

func (conn *TcpConn) Read(b []byte) (int, error) {
	if conn.comp != nil {
		return conn.comp.Read(b)
	}
	return conn.underlying.Read(b)
}

func (conn *TcpConn) Write(b []byte) (int, error) {
	if conn.comp != nil {
		return conn.comp.Write(b)
	}
	return conn.underlying.Write(b)
}

func (conn *TcpConn) Close() error {
	return conn.underlying.Close()
}

func (conn *TcpConn) LocalAddr() net.Addr {
	return conn.underlying.LocalAddr()
}

func (conn *TcpConn) RemoteAddr() net.Addr {
	return conn.underlying.RemoteAddr()
}

func (conn *TcpConn) SetDeadline(t time.Time) error {
	return conn.underlying.SetDeadline(t)
}

func (conn *TcpConn) SetReadDeadline(t time.Time) error {
	return conn.underlying.SetReadDeadline(t)
}

func (conn *TcpConn) SetWriteDeadline(t time.Time) error {
	return conn.underlying.SetWriteDeadline(t)
}

func (conn *TcpConn) BindTlsUnique(w io.Writer) error {
	if c, ok := conn.underlying.(*tls.Conn); ok {
		cs := c.ConnectionState()
		if cs.Version < tls.VersionTLS13 {
			return ErrBindTlsUniqueNotSupported
		}
		w.Write([]byte(cs.TLSUnique))
		return nil
	}
	return ErrBindTlsUniqueNotSupported
}

func (conn *TcpConn) StartTLS(conf *tls.Config) {
	if _, ok := conn.underlying.(*tls.Conn); ok {
		return
	}
	if !conn.isClient {
		conn.underlying = tls.Server(conn.underlying, conf)
		return
	}
	conn.underlying = tls.Client(conn.underlying, conf)
}

func (conn *TcpConn) StartCompress(buildCompress BuildCompressor) {
	conn.comp = buildCompress(conn.underlying)
}
