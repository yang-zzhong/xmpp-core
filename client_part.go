package xmppcore

import (
	"github.com/google/uuid"
	"github.com/jackal-xmpp/stravaganza/v2"
)

type ClientPart struct {
	id            string
	features      []ElemHandler
	commingStream CommingStream
	goingStream   GoingStream
	attr          PartAttr
	logger        Logger
	conn          Conn
	*ElemRunner
}

func NewClientPart(conn Conn, logger Logger, s *PartAttr) *ClientPart {
	commingStream := NewXCommingStream(conn, false)
	return &ClientPart{
		id:            uuid.New().String(),
		commingStream: commingStream,
		features:      []ElemHandler{},
		goingStream:   NewXGoingStream(conn, false),
		logger:        logger,
		ElemRunner:    NewElemRunner(),
		attr:          *s,
		conn:          conn,
	}
}

func (od *ClientPart) Attr() *PartAttr {
	return &od.attr
}

func (od *ClientPart) GoingStream() GoingStream {
	return od.goingStream
}

func (od *ClientPart) CommingStream() CommingStream {
	return od.commingStream
}

func (od *ClientPart) WithFeature(h ElemHandler) {
	od.features = append(od.features, h)
}

func (od *ClientPart) Logger() Logger {
	return od.logger
}

func (od *ClientPart) ID() string {
	return od.id
}

func (od *ClientPart) Conn() Conn {
	return od.conn
}

func (od *ClientPart) Run() error {
	if err := od.handleFeatures(); err != nil {
		return err
	}
	errChan := make(chan error)
	od.ElemRunner.Run(od, errChan)
	return <-errChan
}

func (od *ClientPart) Stop() {
	od.Quit()
}

func (od *ClientPart) handleFeatures() error {
	for {
		features, err := od.serverFeatures()
		if err != nil {
			return err
		}
		if len(features) == 0 {
			return nil
		}
		for _, f := range features {
			handled := false
			for _, h := range od.features {
				if h.Match(f) {
					h.Handle(f, od)
					handled = true
					break
				}
			}
			if !handled {
				// no handler
			}
		}
	}
}

func (od *ClientPart) serverFeatures() (res []stravaganza.Element, err error) {
	od.goingStream.Open(&od.attr)
	if err = od.CommingStream().WaitHeader(&od.attr); err != nil {
		return
	}
	var elem stravaganza.Element
	if err = od.CommingStream().NextElement(&elem); err != nil {
		return
	}
	if elem.Name() != "features" {
		return
	}
	res = elem.AllChildren()
	return
}
