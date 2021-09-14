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

//                 +---------------------+
//                 | open TCP connection |
//                 +---------------------+
//                            |
//                            v
//                     +---------------+
//                     | send initial  |<-------------------------+
//                     | stream header |                          ^
//                     +---------------+                          |
//                            |                                   |
//                            v                                   |
//                    +------------------+                        |
//                    | receive response |                        |
//                    | stream header    |                        |
//                    +------------------+                        |
//                            |                                   |
//                            v                                   |
//                     +----------------+                         |
//                     | receive stream |                         |
// +------------------>| features       |                         |
// ^   {OPTIONAL}      +----------------+                         |
// |                          |                                   |
// |                          v                                   |
// |       +<-----------------+                                   |
// |       |                                                      |
// |    {empty?} ----> {all voluntary?} ----> {some mandatory?}   |
// |       |      no          |          no         |             |
// |       | yes              | yes                 | yes         |
// |       |                  v                     v             |
// |       |           +---------------+    +----------------+    |
// |       |           | MAY negotiate |    | MUST negotiate |    |
// |       |           | any or none   |    | one feature    |    |
// |       |           +---------------+    +----------------+    |
// |       v                  |                     |             |
// |   +---------+            v                     |             |
// |   |  DONE   |<----- {negotiate?}               |             |
// |   +---------+   no       |                     |             |
// |                     yes  |                     |             |
// |                          v                     v             |
// |                          +--------->+<---------+             |
// |                                     |                        |
// |                                     v                        |
// +<-------------------------- {restart mandatory?} ------------>+
//              no                                     yes
func (od *ClientPart) handleFeatures() error {
	for {
		features, err := od.serverFeatures()
		if err != nil || len(features) == 0 {
			return err
		}
		for _, f := range features {
			handled := false
			for _, h := range od.features {
				if h.Match(f) {
					handled = true
					if err := h.Handle(f, od); err != nil {
						return err
					}
					break
				}
			}
			if !handled && od.isMandatory(f) {
				return errors.New(fmt.Sprintf("feature %s not handled", f.Name()))
			}
		}
		if !od.containMandatories(features) {
			return nil
		}
	}
}

func (od *ClientPart) containMandatories(elems []stravaganza.Element) bool {
	for _, elem := range elems {
		if od.isMandatory(elem) {
			return true
		}
	}
	return false
}

func (od *ClientPart) isMandatory(elem stravaganza.Element) bool {
	if elem.Name() == "starttls" || elem.Name() == "bind" {
		return elem.Child("required") != nil
	}
	if elem.Name() == "compression" {
		return false
	}
	return true
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
