package xmppcore

import (
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/jackal-xmpp/stravaganza/v2"
)

var (
	ErrOnNextTokenTimeout      = errors.New("get next token timeout")
	ErrOnNextElementnTimeout   = errors.New("get next element timeout")
	ErrClientIgnoredTheFeature = errors.New("client ignored the feature")
	ErrUnhandledElement        = errors.New("unhandled element")
)

type Feature interface {
	Elem() stravaganza.Element
	Mandatory() bool
	Handled() bool
	ElemHandler
}

type ElemHandler interface {
	ID() string
	Match(stravaganza.Element) bool
	Handle(elem stravaganza.Element, part Part) error
}

type IDAble struct {
	id string
}

func NewIDAble() *IDAble {
	return &IDAble{id: uuid.New().String()}
}

func (idable *IDAble) ID() string {
	return idable.id
}

type Part interface {
	ID() string
	Attr() *PartAttr
	Channel() Channel
	WithElemHandler(ElemHandler)
	Logger() Logger
	Conn() Conn
	Run() error
	Stop()
	handleFeatures(header xml.StartElement) error
}

type ElemRunner struct {
	channel      Channel
	elemHandlers []ElemHandler
	handleLimit  int
	handled      int
	quit         bool
}

func NewElemRunner(channel Channel) *ElemRunner {
	return &ElemRunner{
		channel:      channel,
		handleLimit:  -1,
		handled:      0,
		elemHandlers: []ElemHandler{},
		quit:         false,
	}
}

func (er *ElemRunner) WithElemHandler(handler ElemHandler) {
	for _, h := range er.elemHandlers {
		if h.ID() == handler.ID() {
			return
		}
	}
	er.elemHandlers = append(er.elemHandlers, handler)
}

func (er *ElemRunner) Running() bool {
	return !er.quit
}

func (er *ElemRunner) SetHandleLimit(limit int) {
	er.handleLimit = limit
}

func (er *ElemRunner) Quit() {
	er.quit = true
	er.channel.Close()
}

func (er *ElemRunner) Run(part Part, errChan chan error) {
	go func() {
		i := 0
		for {
			i = i + 1
			i, err := er.channel.next()
			if err != nil {
				if er.quit {
					errChan <- nil
					part.Logger().Printf(Info, "quit!")
					return
				}
				part.Logger().Printf(Error, "a error from part instance [%s] message handler: %s", part.ID(), err.Error())
				errChan <- err
				return
			}
			if header, ok := i.(xml.StartElement); ok {
				if err := part.handleFeatures(header); err != nil {
					errChan <- err
					return
				}
			} else if elem, ok := i.(stravaganza.Element); ok {
				for _, handler := range er.elemHandlers {
					if handler.Match(elem) {
						er.handled = er.handled + 1
						if err := handler.Handle(elem, part); err != nil {
							part.Logger().Printf(Error, "a error occured from part instance [%s] message handler: %s", part.ID(), err.Error())
							errChan <- err
						}
					}
				}
				if er.handleLimit > 0 && er.handled >= er.handleLimit {
					errChan <- nil
					close(errChan)
					return
				}
			} else if bs, ok := i.(xml.CharData); ok {
				fmt.Printf("bs: %v\n", bs)
			}
		}
	}()
}

type PartAttr struct {
	ID      string
	JID     JID    // client's jid
	Domain  string // server domain
	Version string
	Xmlns   string
	XmlLang string
	OpenTag bool
}

func (attr *PartAttr) ToClientHead(elem *xml.StartElement) {
	to := ""
	if attr.JID.Username != "" {
		to = attr.JID.String()
	}
	attr.head(elem, attr.Domain, to)
}

func (attr *PartAttr) ToServerHead(elem *xml.StartElement) {
	from := ""
	if attr.JID.Username != "" {
		from = attr.JID.String()
	}
	attr.head(elem, from, attr.Domain)
}

func (attr *PartAttr) head(elem *xml.StartElement, from, to string) {
	eattr := []xml.Attr{
		{Name: xml.Name{Local: "version"}, Value: attr.Version},
		{Name: xml.Name{Local: "id"}, Value: attr.ID},
	}
	if from != "" {
		eattr = append(eattr, xml.Attr{Name: xml.Name{Local: "from"}, Value: from})
	}
	if to != "" {
		eattr = append(eattr, xml.Attr{Name: xml.Name{Local: "to"}, Value: to})
	}
	if attr.XmlLang != "" {
		eattr = append(eattr, xml.Attr{Name: xml.Name{Local: "lang", Space: "xml"}, Value: attr.XmlLang})
	}
	// if attr.Xmlns != "" {
	// 	eattr = append(eattr, xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: attr.Xmlns})
	// }
	if !attr.OpenTag {
		*elem = xml.StartElement{
			Name: xml.Name{Space: nsStream, Local: "stream"},
			Attr: eattr}
		return
	}
	*elem = xml.StartElement{
		Name: xml.Name{Space: nsFraming, Local: "open"},
		Attr: eattr}
}

func (sa *PartAttr) ParseToServer(elem xml.StartElement) error {
	isStream := elem.Name.Local == "stream" && elem.Name.Space == nsStream
	sa.OpenTag = elem.Name.Local == "open" && elem.Name.Space == nsFraming
	if !isStream && !sa.OpenTag {
		return ErrNotHeaderStart
	}
	for _, attr := range elem.Attr {
		if attr.Name.Local == "from" && attr.Value != "" {
			if err := ParseJID(attr.Value, &sa.JID); err != nil {
				return err
			}
		} else if attr.Name.Local == "to" {
			if attr.Value != sa.Domain {
				return ErrNotForThisDomainHead
			}
		} else if attr.Name.Local == "xmlns" {
			sa.Xmlns = attr.Value
		} else if attr.Name.Local == "version" {
			sa.Version = attr.Value
		} else if attr.Name.Local == "id" && attr.Value != "" {
			sa.ID = attr.Value
		} else if attr.Name.Local == "xml:lang" {
			sa.XmlLang = attr.Value
		}
	}
	return nil
}

func (sa *PartAttr) ParseToClient(elem xml.StartElement) error {
	isStream := elem.Name.Local == "stream" && elem.Name.Space == nsStream
	sa.OpenTag = elem.Name.Local == "open" && elem.Name.Space == nsFraming
	if !isStream && !sa.OpenTag {
		return ErrNotHeaderStart
	}
	for _, attr := range elem.Attr {
		if attr.Name.Local == "to" && attr.Value != "" {
			var jid JID
			if err := ParseJID(attr.Value, &jid); err != nil {
				return err
			}
			if !jid.Equal(sa.JID) {
				return ErrNotForThisDomainHead
			}
		} else if attr.Name.Local == "from" {
			sa.Domain = attr.Value
		} else if attr.Name.Local == "version" {
			sa.Version = attr.Value
		} else if attr.Name.Local == "id" && attr.Value != "" {
			sa.ID = attr.Value
		}
	}
	return nil
}

type XPart struct {
	channel  Channel
	features []Feature
	logger   Logger
	conn     Conn
	attr     PartAttr
	*ElemRunner
}

func NewXPart(conn Conn, domain string, logger Logger) *XPart {
	channel := NewXChannel(conn, true)
	return &XPart{
		channel:    channel,
		features:   []Feature{},
		logger:     logger,
		conn:       conn,
		attr:       PartAttr{Domain: domain, ID: uuid.New().String()},
		ElemRunner: NewElemRunner(channel),
	}
}

func (part *XPart) ID() string {
	return part.attr.ID
}

func (part *XPart) Conn() Conn {
	return part.conn
}

func (part *XPart) Channel() Channel {
	return part.channel
}

func (part *XPart) WithFeature(f Feature) {
	part.features = append(part.features, f)
}

func (part *XPart) Run() error {
	part.logger.Printf(Info, "part instance [%s] start running", part.attr.ID)
	errChan := make(chan error)
	part.ElemRunner.Run(part, errChan)
	return <-errChan
}

func (part *XPart) Attr() *PartAttr {
	return &part.attr
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
func (part *XPart) handleFeatures(header xml.StartElement) error {
	for {
		if err := part.Attr().ParseToServer(header); err != nil {
			return err
		}
		if err := part.Channel().Open(part.Attr()); err != nil {
			return err
		}
		features, hasMandatory := part.unresolvedFeatures()
		if !hasMandatory {
			return part.handleUnmandatoryFeatures(features)
		}
		elems := []stravaganza.Element{}
		runner := NewElemRunner(part.Channel())
		runner.SetHandleLimit(1)
		for _, f := range features {
			runner.WithElemHandler(f)
			elems = append(elems, f.Elem())
		}
		if err := part.notifyFeatures(elems...); err != nil {
			return err
		}
		errChan := make(chan error)
		runner.Run(part, errChan)
		if err := <-errChan; err != nil {
			return err
		}
		if err := part.Channel().WaitHeader(&header); err != nil {
			return err
		}
	}
}

func (part *XPart) handleUnmandatoryFeatures(features []Feature) error {
	elems := []stravaganza.Element{}
	for _, f := range features {
		elems = append(elems, f.Elem())
		part.WithElemHandler(f)
	}
	part.notifyFeatures(elems...)
	return nil
}

func (part *XPart) unresolvedFeatures() (features []Feature, hasMandatory bool) {
	for _, f := range part.features {
		if f.Handled() {
			continue
		}
		if f.Mandatory() {
			hasMandatory = true
		}
		features = append(features, f)
	}
	return
}

func (part *XPart) notifyFeatures(elems ...stravaganza.Element) error {
	return part.Channel().SendElement(stravaganza.NewBuilder("features").WithChildren(elems...).Build())
}

func (part *XPart) Logger() Logger {
	return part.logger
}

func (part *XPart) Stop() {
	part.Quit()
}
