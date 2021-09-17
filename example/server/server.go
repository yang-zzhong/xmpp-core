package server

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"io"
	"os"
	"sync"

	xmppcore "github.com/yang-zzhong/xmpp-core"

	"github.com/gorilla/websocket"
)

type Server struct {
	config *Config
	logger xmppcore.Logger

	connGrabbers []xmppcore.ConnGrabber
	conns        []xmppcore.Conn
	wg           sync.WaitGroup
}

func New(conf *Config) *Server {
	return &Server{
		config:       conf,
		connGrabbers: []xmppcore.ConnGrabber{},
		conns:        []xmppcore.Conn{},
		logger:       xmppcore.NewLogger(os.Stdout)}
}

func (s *Server) Start() error {
	wsConnsConfig := DefaultConfig.WsConns
	if len(s.config.WsConns) > 0 {
		wsConnsConfig = s.config.WsConns
	}
	for _, conf := range wsConnsConfig {
		s.initConnGrabber(conf, func(c interface{}) xmppcore.ConnGrabber {
			conf := c.(WsConnConfig)
			return xmppcore.NewWsConnGrabber(
				conf.ListenOn,
				conf.Path,
				websocket.Upgrader{
					ReadBufferSize:  int(conf.ReadBufSize),
					WriteBufferSize: int(conf.WriteBufSize)}, s.logger)
		})
	}
	tcpConnsConfig := DefaultConfig.TcpConns
	if len(s.config.TcpConns) > 0 {
		tcpConnsConfig = s.config.TcpConns
	}
	for _, conf := range tcpConnsConfig {
		s.initConnGrabber(conf, func(c interface{}) xmppcore.ConnGrabber {
			conf := c.(TcpConnConfig)
			return xmppcore.NewTcpConnGrabber(conf.ListenOn, conf.For, s.logger)
		})
	}
	s.wg.Wait()
	return nil
}

type connGrabberBuilder func(interface{}) xmppcore.ConnGrabber

func (s *Server) initConnGrabber(conf interface{}, builder connGrabberBuilder) {
	grabber := builder(conf)
	connChan := make(chan xmppcore.Conn)
	if err := grabber.Grab(connChan); err != nil {
		panic(err)
	}
	s.connGrabbers = append(s.connGrabbers, grabber)
	s.wg.Add(1)
	go func() {
		for {
			select {
			case conn := <-connChan:
				if conn == nil {
					s.wg.Done()
					return
				}
				s.conns = append(s.conns, conn)
				go func() {
					s.onConn(conn, grabber.For(), grabber.Type())
				}()
			}
		}
	}()
}

func (s *Server) Stop() {
	for _, conn := range s.conns {
		conn.Close()
	}
	for _, grabber := range s.connGrabbers {
		grabber.Cancel()
	}
}

var (
	memoryAuthUserFetcher      *xmppcore.MemoryAuthUserFetcher
	memoryPlainAuthUserFetcher *xmppcore.MemoryPlainAuthUserFetcher
	memoryAuthorized           *xmppcore.MemoryAuthorized
)

func init() {
	memoryAuthUserFetcher = xmppcore.NewMemoryAuthUserFetcher()
	memoryAuthUserFetcher.Add(xmppcore.NewMemomryAuthUser("test", "123456", map[string]func() hash.Hash{
		"MD5":     md5.New,
		"SHA-1":   sha1.New,
		"SHA-256": sha256.New,
		"SHA-512": sha512.New,
	}, 5))
	memoryPlainAuthUserFetcher = xmppcore.NewMemoryPlainAuthUserFetcher()
	memoryPlainAuthUserFetcher.Add(xmppcore.NewMemoryPlainAuthUser("test", string(md5.New().Sum([]byte("123456")))))
	memoryAuthorized = xmppcore.NewMemoryAuthorized()
}

func (s *Server) onConn(conn xmppcore.Conn, connFor xmppcore.ConnFor, connType xmppcore.ConnType) {
	if connFor == xmppcore.ForC2S {
		s.c2sHandler(conn, connType)
	} else if connFor == xmppcore.ForS2S {
		s.s2sHandler(conn, connType)
	}
}

func (s *Server) c2sHandler(conn xmppcore.Conn, connType xmppcore.ConnType) {
	c2s := xmppcore.NewXPart(conn, s.config.Domain, s.logger)
	c2s.Channel().SetLogger(s.logger)
	sasl := xmppcore.NewSASLFeature(memoryAuthorized)
	sasl.Support(xmppcore.SM_PLAIN, xmppcore.NewPlainAuth(memoryPlainAuthUserFetcher, md5.New))
	if s.config.CertFile != "" && s.config.KeyFile != "" || connType == xmppcore.TLSConn || connType == xmppcore.WSTLSConn {
		if connType == xmppcore.TCPConn || connType == xmppcore.WSConn {
			c2s.WithFeature(xmppcore.NewTlsFeature(s.config.CertFile, s.config.KeyFile, true))
		}
		sasl.Support(xmppcore.SM_SCRAM_SHA_1_PLUS, xmppcore.NewScramAuth(memoryAuthUserFetcher, sha1.New, true))
		sasl.Support(xmppcore.SM_SCRAM_SHA_256_PLUS, xmppcore.NewScramAuth(memoryAuthUserFetcher, sha256.New, true))
		sasl.Support(xmppcore.SM_SCRAM_SHA_512_PLUS, xmppcore.NewScramAuth(memoryAuthUserFetcher, sha512.New, true))
	} else {
		sasl.Support(xmppcore.SM_SCRAM_SHA_1, xmppcore.NewScramAuth(memoryAuthUserFetcher, sha1.New, true))
		sasl.Support(xmppcore.SM_SCRAM_SHA_256, xmppcore.NewScramAuth(memoryAuthUserFetcher, sha256.New, true))
		sasl.Support(xmppcore.SM_SCRAM_SHA_512, xmppcore.NewScramAuth(memoryAuthUserFetcher, sha512.New, true))
	}
	c2s.WithFeature(sasl)
	compress := xmppcore.NewCompressFeature()
	compress.Support(xmppcore.ZLIB, func(conn io.ReadWriter) xmppcore.Compressor {
		return xmppcore.NewCompZlib(conn)
	})
	c2s.WithFeature(compress)
	c2s.WithFeature(xmppcore.NewBindFeature(memoryAuthorized))
	c2s.WithElemHandler(xmppcore.NewMessageRouter(memoryAuthorized))
	if err := c2s.Run(); err != nil {
		s.logger.Printf(xmppcore.Error, err.Error())
	}
}

func (s *Server) s2sHandler(conn xmppcore.Conn, connType xmppcore.ConnType) {
	s.c2sHandler(conn, connType)
}
