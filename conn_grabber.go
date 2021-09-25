package xmppcore

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

type ConnFor string
type ConnType string

const (
	ForC2S = ConnFor("C2S")
	ForS2S = ConnFor("S2S")

	TCPConn   = ConnType("TCP")
	TLSConn   = ConnType("TLS")
	WSConn    = ConnType("WS-TCP")
	WSTLSConn = ConnType("WS-TLS")
)

type ConnGrabber interface {
	Grab(chan Conn) error
	Cancel()
	For() ConnFor
	Type() ConnType
}

type WsConnGrabber struct {
	upgrader websocket.Upgrader
	logger   Logger
	path     string
	listenOn string

	certFile, keyFile string

	grabbing bool

	srv *http.Server
}

func NewWsConnGrabber(listenOn string, path string, upgrader websocket.Upgrader, logger Logger) *WsConnGrabber {
	return &WsConnGrabber{
		upgrader: upgrader,
		grabbing: false,
		path:     path,
		listenOn: listenOn, logger: logger}
}

func (wsc *WsConnGrabber) UpgradeToTls(certFile, keyFile string) error {
	if wsc.grabbing {
		return errors.New("can't update to tls within grabbing")
	}
	wsc.certFile = certFile
	wsc.keyFile = keyFile
	return nil
}

func (wsc *WsConnGrabber) Grab(connChan chan Conn) error {
	mux := http.NewServeMux()
	mux.Handle(wsc.path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsconn, e := wsc.upgrader.Upgrade(w, r, nil)
		if e != nil {
			wsc.logger.Printf(LogError, "upgrade ws conn err: %s\n", e.Error())
			return
		}
		uc := wsconn.UnderlyingConn()
		wsc.logger.Printf(LogInfo, "comming a ws connection from %s\n", uc.RemoteAddr().String())
		c, _ := uc.(*net.TCPConn)
		connChan <- NewTcpConn(c, false)
	}))
	wsc.srv = &http.Server{Handler: mux, Addr: wsc.listenOn}
	wsc.srv.RegisterOnShutdown(func() {
		close(connChan)
	})
	go func() {
		wsc.grabbing = true
		defer func() {
			wsc.grabbing = false
		}()
		wsc.logger.Printf(LogInfo, "ws connection listen on %s for C2S\n", wsc.listenOn)
		var err error
		if wsc.certFile != "" && wsc.keyFile != "" {
			err = wsc.srv.ListenAndServeTLS(wsc.certFile, wsc.keyFile)
		} else {
			err = wsc.srv.ListenAndServe()
		}
		if err != nil {
			wsc.logger.Printf(LogInfo, "ws conn server close unexpected: %s\n", err.Error())
			return
		}
		wsc.logger.Printf(LogInfo, "ws server quit normally\n")
	}()
	return nil
}

func (wsc *WsConnGrabber) Cancel() {
	if wsc.srv != nil {
		wsc.srv.Shutdown(context.Background())
	}
}

func (wsc *WsConnGrabber) For() ConnFor {
	return ForC2S
}

func (wsc *WsConnGrabber) Type() ConnType {
	if wsc.certFile != "" && wsc.keyFile != "" {
		return WSTLSConn
	}
	return WSConn
}

type TcpConnGrabber struct {
	listenOn string
	connFor  ConnFor
	logger   Logger
	grabbing bool

	keyFile  string
	certFile string

	quit chan bool
	ln   net.Listener
}

func NewTcpConnGrabber(listenOn string, connFor ConnFor, logger Logger) *TcpConnGrabber {
	return &TcpConnGrabber{listenOn: listenOn, connFor: connFor, quit: make(chan bool), logger: logger}
}

func (tc *TcpConnGrabber) UpgradeToTls(certFile, keyFile string) error {
	if tc.grabbing {
		return errors.New("tcp conn grabber grabbing, can't upgrade to tls")
	}
	tc.certFile = certFile
	tc.keyFile = keyFile
	return nil
}

func (tc *TcpConnGrabber) ReplaceLogger(logger Logger) {
	tc.logger = logger
}

func (tc *TcpConnGrabber) Grab(connChan chan Conn) error {
	var err error
	if tc.certFile != "" && tc.keyFile != "" {
		cert, e := tls.LoadX509KeyPair(tc.certFile, tc.keyFile)
		if e != nil {
			return e
		}
		config := tls.Config{Certificates: []tls.Certificate{cert}}
		tc.ln, err = tls.Listen("tcp", tc.listenOn, &config)
	} else {
		tc.ln, err = net.Listen("tcp", tc.listenOn)
	}
	tc.logger.Printf(LogInfo, "tcp connection Listen on %s for %s\n", tc.listenOn, tc.connFor)
	if err != nil {
		return err
	}
	go func() {
		tc.grabbing = true
		defer func() {
			tc.grabbing = false
		}()
		for {
			select {
			case <-tc.quit:
				close(connChan)
				return
			default:
				conn, err := tc.ln.Accept()
				if err != nil {
					continue
				}
				if conn == nil {
					close(connChan)
					return
				}
				tc.logger.Printf(LogInfo, "comming a tcp connection from %s on %s\n", conn.RemoteAddr().String(), tc.listenOn)
				connChan <- NewTcpConn(conn, false)
			}
		}
	}()
	return nil
}

func (tc *TcpConnGrabber) Cancel() {
	tc.ln.Close()
	tc.quit <- true
}

func (tc *TcpConnGrabber) For() ConnFor {
	return tc.connFor
}

func (tc *TcpConnGrabber) Type() ConnType {
	if tc.certFile != "" && tc.keyFile != "" {
		return TLSConn
	}
	return TCPConn
}
