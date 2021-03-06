package xmppcore

import (
	"encoding/xml"
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
	elemRunner
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
		elemRunner: ElemRunner(channel),
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

func (od *ClientPart) Negotiate() error {
	od.Channel().Open(od.Attr())
	var header xml.StartElement
	if err := od.Channel().WaitHeader(&header); err != nil {
		return err
	}
	return od.handleFeatures(header)
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
func (od *ClientPart) handleFeatures(header xml.StartElement) error {
	for {
		features, err := od.serverFeatures()
	handle:
		if err != nil || len(features) == 0 {
			return err
		}
		f, rest := od.selectOne(features)
		handled, err := od.handle(f)
		if err != nil {
			return err
		}
		if !handled {
			features = rest
			goto handle
		}
		if err := od.Channel().Open(od.Attr()); err != nil {
			return err
		}
		if err := od.Channel().WaitHeader(&header); err != nil {
			return err
		}
	}
}

func (od *ClientPart) Run() chan error {
	return od.elemRunner.Run(od)
}

func (od *ClientPart) handle(f stravaganza.Element) (handled bool, err error) {
	for _, h := range od.features {
		if handled, err = h.Handle(f, od); handled {
			if err != nil {
				return
			}
			break
		}
	}
	if !handled && od.isMandatory(f) {
		err = fmt.Errorf("feature %s not handled", f.Name())
	}
	return
}

func (od *ClientPart) OnOpenHeader(header xml.StartElement) error {
	return errors.New("unexpected open header")
}

func (od *ClientPart) OnWhiteSpace(bs []byte) {}

func (od *ClientPart) OnCloseToken() {
	od.Quit()
}

func (od *ClientPart) selectOne(features []stravaganza.Element) (f stravaganza.Element, rest []stravaganza.Element) {
	priorities := []string{"starttls", "mechanisms", "bind"}
	for _, s := range priorities {
		var i int
		for i, f = range features {
			if f.Name() == s {
				rest = append(features[:i], features[i+1:]...)
				return
			}
		}
	}
	f = features[0]
	rest = features[1:]
	return
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
