package xmppcore

import (
	"context"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

type ConnFor string
type ConnType string

const (
	ForC2S = ConnFor("C2S")
	ForS2S = ConnFor("S2S")

	TCPConn = ConnType("TCP")
	WSConn  = ConnType("WS-TCP")
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

	srv *http.Server
}

func NewWsConnGrabber(listenOn string, path string, upgrader websocket.Upgrader, logger Logger) *WsConnGrabber {
	return &WsConnGrabber{
		upgrader: upgrader,
		path:     path,
		listenOn: listenOn, logger: logger}
}

func (wsc *WsConnGrabber) Grab(connChan chan Conn) error {
	mux := http.NewServeMux()
	mux.Handle(wsc.path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsconn, e := wsc.upgrader.Upgrade(w, r, nil)
		if e != nil {
			wsc.logger.Printf(Error, "upgrade ws conn err: %s\n", e.Error())
			return
		}
		uc := wsconn.UnderlyingConn()
		wsc.logger.Printf(Info, "comming a ws connection from %s\n", uc.RemoteAddr().String())
		c, _ := uc.(*net.TCPConn)
		connChan <- NewTcpConn(c, false)
	}))
	wsc.srv = &http.Server{Handler: mux, Addr: wsc.listenOn}
	wsc.srv.RegisterOnShutdown(func() {
		close(connChan)
	})
	go func() {
		wsc.logger.Printf(Info, "ws connection listen on %s for c2s\n", wsc.listenOn)
		if err := wsc.srv.ListenAndServe(); err != nil {
			wsc.logger.Printf(Info, "ws conn server close unexpected: %s\n", err.Error())
			return
		}
		wsc.logger.Printf(Info, "ws server quit normally\n")
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
	return WSConn
}

type TcpConnGrabber struct {
	listenOn string
	connFor  ConnFor
	logger   Logger

	keyFile  string
	certFile string

	quit chan bool
	ln   net.Listener
}

func NewTcpConnGrabber(listenOn string, connFor ConnFor, logger Logger) *TcpConnGrabber {
	return &TcpConnGrabber{listenOn: listenOn, connFor: connFor, quit: make(chan bool), logger: logger}
}

func (tc *TcpConnGrabber) ReplaceLogger(logger Logger) {
	tc.logger = logger
}

func (tc *TcpConnGrabber) Grab(connChan chan Conn) error {
	var err error
	tc.ln, err = net.Listen("tcp", tc.listenOn)
	tc.logger.Printf(Info, "tcp connection Listen on %s for %s\n", tc.listenOn, tc.connFor)
	if err != nil {
		return err
	}
	go func() {
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
				tc.logger.Printf(Info, "comming a tcp connection from %s on %s\n", conn.RemoteAddr().String(), tc.listenOn)
				c, _ := conn.(*net.TCPConn)
				connChan <- NewTcpConn(c, false)
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
	return TCPConn
}
