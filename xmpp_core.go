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

type OptionalFeature interface {
	Elem() stravaganza.Element
	ElemHandler
}

type RequiredFeature interface {
	Resolve(Part) error
}

type ElemHandler interface {
	Match(stravaganza.Element) bool
	Handle(elem stravaganza.Element, part Part) error
}

type Part interface {
	ID() string
	Attr() *PartAttr
	CommingStream() CommingStream
	GoingStream() GoingStream
	Logger() Logger
	Conn() Conn
	Run() error
	Stop()
}

type ElemRunner struct {
	elemHandlers []ElemHandler
	quitChan     chan bool
	quit         bool
}

func NewElemRunner() *ElemRunner {
	return &ElemRunner{
		elemHandlers: []ElemHandler{},
		quitChan:     make(chan bool),
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
	er.quitChan <- true
}

func (er *ElemRunner) Run(part Part, errChan chan error) {
	go func() {
		for {
			select {
			case <-er.quitChan:
				errChan <- nil
				return
			default:
				var elem stravaganza.Element
				if err := part.CommingStream().NextElement(&elem); err != nil {
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
		}
	}()
}

type PartAttr struct {
	ID      string
	JID     JID
	Domain  string
	Version string
	Xmlns   string
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
			if jid.Equal(&sa.JID) {
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
	commingStream    CommingStream
	goingStream      GoingStream
	requiredFeatures []RequiredFeature
	optionalFeatures []OptionalFeature
	logger           Logger
	conn             Conn
	attr             PartAttr
	*ElemRunner
}

func NewXPart(conn Conn, domain string, logger Logger) *XPart {
	commingStream := NewXCommingStream(conn, true)
	return &XPart{
		commingStream:    commingStream,
		goingStream:      NewXGoingStream(conn, true),
		requiredFeatures: []RequiredFeature{},
		optionalFeatures: []OptionalFeature{},
		logger:           logger,
		conn:             conn,
		attr:             PartAttr{Domain: domain, ID: uuid.New().String()},
		ElemRunner:       NewElemRunner(),
	}
}

func (part *XPart) ID() string {
	return part.attr.ID
}

func (part *XPart) Conn() Conn {
	return part.conn
}

func (part *XPart) CommingStream() CommingStream {
	return part.commingStream
}

func (part *XPart) GoingStream() GoingStream {
	return part.goingStream
}

func (part *XPart) WithRequiredFeature(f RequiredFeature) {
	part.requiredFeatures = append(part.requiredFeatures, f)
}

func (part *XPart) WithOptionalFeature(f OptionalFeature) {
	part.optionalFeatures = append(part.optionalFeatures, f)
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

func (part *XPart) handleFeatures() error {
	reopened := false
	for _, f := range part.requiredFeatures {
		if !reopened {
			if err := part.reopen(); err != nil {
				return err
			}
		}
		if err := f.Resolve(part); err != nil {
			if err == ErrClientIgnoredTheFeature {
				part.goingStream.Open(part.Attr())
				reopened = true
				continue
			}
			part.Logger().Printf(Error, "a error from part feature: %s", err.Error())
			return err
		}
		reopened = false
	}
	part.reopen()
	if err := part.handleUnrequiredFeature(); err != nil {
		part.Logger().Printf(Error, "a error from part unrequired features: %s", err.Error())
	}
	return nil
}

func (part *XPart) handleUnrequiredFeature() error {
	if len(part.optionalFeatures) == 0 {
		return part.goingStream.SendElement(stravaganza.NewBuilder("features").Build())
	}
	es := []stravaganza.Element{}
	for _, f := range part.optionalFeatures {
		es = append(es, f.Elem())
		part.WithElemHandler(f)
	}
	fs := stravaganza.NewBuilder("features").WithChildren(es...).Build()
	return part.goingStream.SendElement(fs)
}

func (part *XPart) reopen() error {
	if err := part.CommingStream().WaitHeader(&part.attr); err != nil {
		return err
	}
	return part.GoingStream().Open(part.Attr())
}

func (part *XPart) Logger() Logger {
	return part.logger
}

func (part *XPart) Stop() {
	part.Quit()
}
