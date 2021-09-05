package example

import (
	xmppcore "xmpp-core"
)

type WsConnConfig struct {
	ListenOn     string `yml:"listen_on"`
	Path         string `yml:"path"`
	ReadBufSize  uint   `yml:"read_buf_size"`
	WriteBufSize uint   `yml:"write_buf_size"`
}

type TcpConnConfig struct {
	ListenOn string           `yml:"listen_on"`
	For      xmppcore.ConnFor `yml:"for"`
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
	DefaultConfig = Config{
		WsConns: []WsConnConfig{
			{ListenOn: ":8080", Path: "/ws", ReadBufSize: 1024, WriteBufSize: 1024},
		},
		TcpConns: []TcpConnConfig{
			{ListenOn: ":5222", For: xmppcore.ForC2S},
			{ListenOn: ":8082", For: xmppcore.ForS2S},
		},
		Domain:   "localhost",
		CertFile: "/Users/young/.openssl/server.crt",
		KeyFile:  "/Users/young/.openssl/server.key",
	}
}
