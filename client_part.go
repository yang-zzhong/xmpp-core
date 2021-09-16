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
	od.Channel().Open(od.Attr())
	var header xml.StartElement
	i, err := od.Channel().next()
	if err != nil {
		return err
	}
	var ok bool
	if header, ok = i.(xml.StartElement); !ok {
		return errors.New("unexpected elem")
	}
	if err := od.handleFeatures(header); err != nil {
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
func (od *ClientPart) handleFeatures(header xml.StartElement) error {
	for {
		features, err := od.serverFeatures()
		if err != nil || len(features) == 0 {
			return err
		}
	handle:
		f := od.selectOne(features)
		handled, err := od.handle(f)
		if err != nil {
			return err
		}
		if !handled {
			goto handle
		}
		if err := od.Channel().Open(od.Attr()); err != nil {
			return err
		}
		i, err := od.Channel().next()
		if err != nil {
			return err
		}
		var ok bool
		if header, ok = i.(xml.StartElement); !ok {
			return errors.New("not a header")
		}
	}
}

func (od *ClientPart) handle(f stravaganza.Element) (handled bool, err error) {
	for _, h := range od.features {
		if h.Match(f) {
			handled = true
			if err = h.Handle(f, od); err != nil {
				return
			}
			break
		}
	}
	if !handled && od.isMandatory(f) {
		err = errors.New(fmt.Sprintf("feature %s not handled", f.Name()))
	}
	return
}

func (od *ClientPart) selectOne(features []stravaganza.Element) stravaganza.Element {
	priorities := []string{"starttls", "mechanisms", "bind"}
	for _, s := range priorities {
		for i, f := range features {
			if f.Name() == s {
				features = append(features[:i], features[i+1:]...)
				return f
			}
		}
	}
	feature := features[0]
	features = features[1:]
	return feature
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
	// od.channel.Open(&od.attr)
	// var header stravaganza.Element
	// od.channel.NextElement(&header)
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
