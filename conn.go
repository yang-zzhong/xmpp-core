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
	isTls      bool
}

func NewTcpConn(underlying *net.TCPConn) *TcpConn {
	return &TcpConn{underlying, nil, false}
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
	if conn.comp != nil {
		if err := conn.comp.Close(); err != nil {
			return err
		}
	}
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
		w.Write([]byte(c.ConnectionState().TLSUnique))
		return nil
	}
	return ErrBindTlsUniqueNotSupported
}

func (conn *TcpConn) StartTLS(conf *tls.Config) {
	underlying := tls.Server(conn.underlying, conf)
	conn.underlying = underlying
	conn.isTls = true
}

func (conn *TcpConn) StartCompress(buildCompress BuildCompressor) {
	conn.comp = buildCompress(conn.underlying)
}
