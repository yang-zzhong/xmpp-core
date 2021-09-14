package xmppcore

import (
	"encoding/xml"
	"errors"

	"github.com/google/uuid"

	"github.com/jackal-xmpp/stravaganza/v2"
)

var (
	ErrOnNextTokenTimeout      = errors.New("get next token timeout")
	ErrOnNextElementnTimeout   = errors.New("get next element timeout")
	ErrClientIgnoredTheFeature = errors.New("client ignored the feature")
)

type Feature interface {
	Elem() stravaganza.Element
	Mandatory() bool
	Handled() bool
	ElemHandler
}

type ElemHandler interface {
	Match(stravaganza.Element) bool
	Handle(elem stravaganza.Element, part Part) error
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
}

type ElemRunner struct {
	channel      Channel
	elemHandlers []ElemHandler
	quitChan     chan bool
	quit         bool
}

func NewElemRunner(channel Channel) *ElemRunner {
	return &ElemRunner{
		channel:      channel,
		elemHandlers: []ElemHandler{},
		quit:         false,
	}
}

func (er *ElemRunner) WithElemHandler(handler ElemHandler) {
	er.elemHandlers = append(er.elemHandlers, handler)
}

func (er *ElemRunner) Running() bool {
	return !er.quit
}

func (er *ElemRunner) Quit() {
	er.quit = true
	er.channel.Close()
}

func (er *ElemRunner) Run(part Part, errChan chan error) {
	go func() {
		for {
			var elem stravaganza.Element
			if err := er.channel.NextElement(&elem); err != nil {
				if er.quit == true {
					errChan <- nil
					return
				}
				part.Logger().Printf(Error, "a error from part instance [%s] message handler: %s", part.ID(), err.Error())
				errChan <- err
				return
			}
			for _, handler := range er.elemHandlers {
				if handler.Match(elem) {
					if err := handler.Handle(elem, part); err != nil {
						part.Logger().Printf(Error, "a error occured from part instance [%s] message handler: %s", part.ID(), err.Error())
						errChan <- err
					}
				}
			}
		}
	}()
}

type PartAttr struct {
	ID      string
	JID     JID
	Domain  string
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
	// 	eattr = append(eattr, xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: attr.XmlLang})
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
		} else if attr.Name.Local == "lang" {
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
			if !jid.Equal(&sa.JID) {
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
	if err := part.handleFeatures(); err != nil {
		return err
	}
	return part.handleElemHandlers()
}

func (part *XPart) Attr() *PartAttr {
	return &part.attr
}

func (part *XPart) handleElemHandlers() error {
	errChan := make(chan error)
	part.ElemRunner.Run(part, errChan)
	return <-errChan
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

func (part *XPart) handleFeatures() error {
	for {
		f := part.nextUnresolvedMandatoryFeature()
		if f == nil {
			break
		}
		if err := part.reopen(); err != nil {
			return err
		}
		if err := part.notifyFeatures(f.Elem()); err != nil {
			return err
		}
		var elem stravaganza.Element
		if err := part.Channel().NextElement(&elem); err != nil {
			return err
		}
		if !f.Match(elem) {
			return errors.New("client error")
		}
		if err := f.Handle(elem, part); err != nil {
			return err
		}
	}
	return part.handleOptionalFeatures()
}

func (part *XPart) nextUnresolvedMandatoryFeature() Feature {
	for _, feature := range part.features {
		if feature.Mandatory() && !feature.Handled() {
			return feature
		}
	}
	return nil
}

func (part *XPart) handleOptionalFeatures() error {
	elems := []stravaganza.Element{}
	for _, f := range part.features {
		if f.Handled() || f.Mandatory() {
			continue
		}
		elems = append(elems, f.Elem())
		part.WithElemHandler(f)
	}
	if err := part.reopen(); err != nil {
		return err
	}
	part.notifyFeatures(elems...)
	return nil
}

func (part *XPart) notifyFeatures(elems ...stravaganza.Element) error {
	return part.Channel().SendElement(stravaganza.NewBuilder("features").WithChildren(elems...).Build())
}

func (part *XPart) reopen() error {
	if err := part.channel.WaitHeader(&part.attr); err != nil {
		return err
	}
	return part.channel.Open(part.Attr())
}

func (part *XPart) Logger() Logger {
	return part.logger
}

func (part *XPart) Stop() {
	part.Quit()
}
