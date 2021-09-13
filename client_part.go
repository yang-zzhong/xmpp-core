package xmppcore

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/stravaganza/v2"
)

type ClientPart struct {
	features []ElemHandler
	channel  Channel
	attr     PartAttr
	logger   Logger
	conn     Conn
	*ElemRunner
}

func NewClientPart(conn Conn, logger Logger, s *PartAttr) *ClientPart {
	channel := NewXChannel(conn, false)
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return &ClientPart{
		features:   []ElemHandler{},
		channel:    channel,
		logger:     logger,
		ElemRunner: NewElemRunner(channel),
		attr:       *s,
		conn:       conn,
	}
}

func (od *ClientPart) Attr() *PartAttr {
	return &od.attr
}

func (od *ClientPart) Channel() Channel {
	return od.channel
}

func (od *ClientPart) WithFeature(h ElemHandler) {
	od.features = append(od.features, h)
}

func (od *ClientPart) Logger() Logger {
	return od.logger
}

func (od *ClientPart) ID() string {
	return od.attr.ID
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
				err := errors.New(fmt.Sprintf("feature [%s] not support", f.GoString()))
				od.logger.Printf(Error, err.Error())
				od.channel.Close()
				return err
				// no handler
			}
		}
	}
}

func (od *ClientPart) serverFeatures() (res []stravaganza.Element, err error) {
	od.channel.Open(&od.attr)
	if err = od.channel.WaitHeader(&od.attr); err != nil {
		return
	}
	var elem stravaganza.Element
	if err = od.channel.NextElement(&elem); err != nil {
		return
	}
	if elem.Name() != "features" {
		return
	}
	res = elem.AllChildren()
	return
}
