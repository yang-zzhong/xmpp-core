package server

import (
	xmppcore "github.com/yang-zzhong/xmpp-core"
)

type WsConnConfig struct {
	ListenOn     string `yml:"listen_on"`
	Path         string `yml:"path"`
	ReadBufSize  uint   `yml:"read_buf_size"`
	WriteBufSize uint   `yml:"write_buf_size"`

	CertFile string `yml:"cert_file"`
	KeyFile  string `yml:"key_file"`
}

type TcpConnConfig struct {
	ListenOn string           `yml:"listen_on"`
	For      xmppcore.ConnFor `yml:"for"`

	CertFile string `yml:"cert_file"`
	KeyFile  string `yml:"key_file"`
}

type Config struct {
	WsConns  []WsConnConfig  `yml:"ws_conns"`
	TcpConns []TcpConnConfig `yml:"tcp_conns"`
	Domain   string          `yml:"domain"`
	CertFile string          `yml:"cert_file"`
	KeyFile  string          `yml:"key_file"`
}

var DefaultConfig Config

func init() {
	cf := "/Users/young/.openssl/server.crt"
	kf := "/Users/young/.openssl/server.key"
	DefaultConfig = Config{
		WsConns: []WsConnConfig{
			{ListenOn: ":80", Path: "/ws", ReadBufSize: 1024, WriteBufSize: 1024},
			{ListenOn: ":443", Path: "/ws", ReadBufSize: 1024, WriteBufSize: 1024, CertFile: cf, KeyFile: kf},
		},
		TcpConns: []TcpConnConfig{
			{ListenOn: ":5221", For: xmppcore.ForC2S},
			{ListenOn: ":5222", For: xmppcore.ForC2S, CertFile: cf, KeyFile: kf},
			{ListenOn: ":5223", For: xmppcore.ForS2S},
		},
		Domain:   "hello-world.im",
		CertFile: cf,
		KeyFile:  kf,
	}
}
