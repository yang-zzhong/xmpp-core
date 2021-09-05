package example

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"io"
	"os"
	"sync"
	xmppcore "xmpp-core"

	"github.com/gorilla/websocket"
)

type Server struct {
	config *Config
	logger xmppcore.Logger

	connGrabbers []xmppcore.ConnGrabber
	conns        []xmppcore.Conn
	wg           sync.WaitGroup
}

func NewServer(conf *Config) *Server {
	return &Server{
		config:       conf,
		connGrabbers: []xmppcore.ConnGrabber{},
		conns:        []xmppcore.Conn{},
		logger:       xmppcore.NewDefaultLogger(os.Stdout)}
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
}

func (s *Server) onConn(conn xmppcore.Conn, connFor xmppcore.ConnFor, connType xmppcore.ConnType) {
	c2s := xmppcore.NewC2S(conn, s.config.Domain, s.logger)

	sasl := xmppcore.NewSASLFeature()
	sasl.Support(xmppcore.SM_PLAIN, xmppcore.NewPlainAuth(memoryPlainAuthUserFetcher, md5.New))
	if s.config.CertFile != "" && s.config.KeyFile != "" {
		c2s.WithFeature(xmppcore.NewTlsFeature(s.config.CertFile, s.config.KeyFile))

		sasl.Support(xmppcore.SM_SCRAM_SHA_1_PLUS, xmppcore.NewScramAuth(true, memoryAuthUserFetcher, sha1.New))
		sasl.Support(xmppcore.SM_SCRAM_SHA_256_PLUS, xmppcore.NewScramAuth(true, memoryAuthUserFetcher, sha256.New))
		sasl.Support(xmppcore.SM_SCRAM_SHA_512_PLUS, xmppcore.NewScramAuth(true, memoryAuthUserFetcher, sha512.New))
	} else {
		sasl.Support(xmppcore.SM_SCRAM_SHA_1, xmppcore.NewScramAuth(false, memoryAuthUserFetcher, sha1.New))
		sasl.Support(xmppcore.SM_SCRAM_SHA_256, xmppcore.NewScramAuth(false, memoryAuthUserFetcher, sha256.New))
		sasl.Support(xmppcore.SM_SCRAM_SHA_512, xmppcore.NewScramAuth(false, memoryAuthUserFetcher, sha512.New))
	}
	c2s.WithFeature(sasl)
	c2s.WithFeature(xmppcore.NewBindFeature())

	comp := xmppcore.NewCompressFeature()
	comp.Support("zlib", func(conn io.ReadWriter) xmppcore.Compressor {
		return xmppcore.NewCompZlib(conn)
	})
	c2s.WithFeature(comp)
	c2s.WithMessageHandler(&xmppcore.EchoMessageHandler{})
	c2s.WithMessageHandler(&xmppcore.MessageRouter{})
	if err := c2s.Run(); err != nil {
		s.logger.Printf(xmppcore.Error, err.Error())
	}

}
