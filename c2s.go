package xmppcore

import (
	"github.com/jackal-xmpp/stravaganza/v2"
)

type C2S struct {
	commingStream   CommingStream
	goingStream     GoingStream
	features        []Feature
	messageHandlers []MessageHandler
	logger          Logger
	conn            Conn
}

func NewC2S(conn Conn, domain string, logger Logger) *C2S {
	return &C2S{
		commingStream:   NewXCommingStream(conn, domain),
		goingStream:     NewXGoingStream(conn),
		features:        []Feature{},
		messageHandlers: []MessageHandler{},
		logger:          logger,
		conn:            conn,
	}
}

func (c2s *C2S) Conn() Conn {
	return c2s.conn
}

func (c2s *C2S) CommingStream() CommingStream {
	return c2s.commingStream
}

func (c2s *C2S) GoingStream() GoingStream {
	return c2s.goingStream
}

func (c2s *C2S) WithFeature(f Feature) {
	c2s.features = append(c2s.features, f)
}

func (c2s *C2S) WithMessageHandler(mh MessageHandler) {
	c2s.messageHandlers = append(c2s.messageHandlers, mh)
}

func (c2s *C2S) Run() error {
	for _, f := range c2s.features {
		if f.Reopen() {
			c2s.reopen()
		}
		if err := f.Resolve(c2s); err != nil {
			c2s.logger.Printf(Error, "a error from resolver: %s", err.Error())
			return err
		}
	}
	c2s.GoingStream().SendElement(stravaganza.NewBuilder("features").Build())
	for {
		var elem stravaganza.Element
		if err := c2s.CommingStream().NextElement(&elem); err != nil {
			return err
		}
		for _, handler := range c2s.messageHandlers {
			if handler.Match(elem) {
				handler.Handle(elem, c2s)
			}
		}
	}
}

func (c2s *C2S) reopen() error {
	if err := c2s.CommingStream().WaitHeader(nil); err != nil {
		return err
	}
	return c2s.GoingStream().Open(c2s.CommingStream())
}

func (c2s *C2S) nextUnresolvedFeature() Feature {
	for _, f := range c2s.features {
		if f.Resolved() {
			continue
		}
		return f
	}
	return nil
}

func (c2s *C2S) Logger() Logger {
	return c2s.logger
}
