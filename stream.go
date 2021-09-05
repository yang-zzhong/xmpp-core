package xmppcore

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/stravaganza/v2"
)

type StreamAttr interface {
	ID() string
	JID() *JID
	Version() string

	SetJID(JID)
	SetVersion(string)
	SetID(string)
}

type CommingStream interface {
	StreamAttr
	WaitHeader(header *xml.StartElement) error
	NextToken(token *xml.Token) error
	NextElement(elem *stravaganza.Element) error
}

type GoingStream interface {
	StreamAttr
	Open(commingStream CommingStream) error
	Send([]byte) error
	SendToken(xml.Token) error
	SendElement(stravaganza.Element) error
}

const (
	nsStream = "http://etherx.jabber.org/streams"
)

var (
	ErrUnproperFromAttr     = errors.New("unproper from attr")
	ErrIncorrectJidEncoding = errors.New("incorrect jid encoding")
)

type JID struct {
	Username string
	Domain   string
	Resource string
}

func ParseJID(src string, jid *JID) error {
	v := strings.Split(src, "@")
	if len(v) != 2 {
		return ErrIncorrectJidEncoding
	}
	jid.Username = v[0]
	idx := strings.Index(v[1], "/")
	if idx > 0 {
		jid.Domain = v[1][:idx]
		jid.Resource = v[1][idx:]
		return nil
	}
	jid.Domain = v[1]
	return nil
}

func (jid *JID) String() string {
	return fmt.Sprintf("%s@%s%s", jid.Username, jid.Domain, jid.Resource)
}

type XStreamAttr struct {
	id      string
	jid     JID
	version string
}

func NewXStreamAttr() *XStreamAttr {
	return &XStreamAttr{id: uuid.New().String(), jid: JID{}}
}

func (xs *XStreamAttr) ID() string {
	return xs.id
}

func (xs *XStreamAttr) JID() *JID {
	return &xs.jid
}

func (xs *XStreamAttr) Domain() string {
	return xs.jid.Domain
}

func (xs *XStreamAttr) Version() string {
	return xs.version
}

func (xs *XStreamAttr) SetJID(jid JID) {
	xs.jid = jid
}

func (xs *XStreamAttr) SetID(id string) {
	xs.id = id
}

func (xs *XStreamAttr) SetVersion(v string) {
	xs.version = v
}

type XCommingStream struct {
	conn    Conn
	decoder *xml.Decoder
	max     int
	*XStreamAttr
}

func NewXCommingStream(conn Conn, domain string) *XCommingStream {
	xs := NewXStreamAttr()
	xs.JID().Domain = domain
	return &XCommingStream{
		conn:        conn,
		decoder:     xml.NewDecoder(conn),
		max:         1024 * 1024 * 2,
		XStreamAttr: xs,
	}
}

func (xc *XCommingStream) NextToken(token *xml.Token) error {
	var err error
	*token, err = xc.decoder.Token()
	return err
}

func (xc *XCommingStream) NextElement(elem *stravaganza.Element) error {
	var err error
	*elem, err = NewParser(xc.conn, xc.max).Parse()
	return err
}

func (xc *XCommingStream) WaitHeader(header *xml.StartElement) error {
	var token xml.Token
	for {
		if err := xc.NextToken(&token); err != nil {
			return err
		}
		switch elem := token.(type) {
		case xml.StartElement:
			if elem.Name.Local != "stream" || elem.Name.Space != nsStream {
				continue
			}
			if header != nil {
				*header = elem
			}
			var version string
			var domain string
			for _, attr := range elem.Attr {
				if attr.Name.Local == "from" && attr.Value != "" {
					value := strings.Split(attr.Value, "@")
					if len(value) < 2 {
						return ErrUnproperFromAttr
					}
					xc.JID().Username = value[0]
					if xc.JID().Domain != value[1] {
						return ErrUnproperFromAttr
					}
				} else if attr.Name.Local == "to" {
					domain = attr.Value
				} else if attr.Name.Local == "version" {
					version = attr.Value
				}
			}
			if domain != xc.JID().Domain {
				return ErrUnproperFromAttr
			}
			xc.SetVersion(version)
			return nil
		default:
			continue
		}
	}
}

type XGoingStream struct {
	conn    io.Writer
	encoder *xml.Encoder
	*XStreamAttr
}

func NewXGoingStream(conn io.Writer) *XGoingStream {
	return &XGoingStream{conn, xml.NewEncoder(conn), NewXStreamAttr()}
}

func (gs *XGoingStream) Open(commingStream CommingStream) error {
	gs.SetID(commingStream.ID())
	gs.SetVersion(commingStream.Version())

	gs.Send([]byte("<?xml version='1.0'?>"))
	attr := []xml.Attr{
		{Name: xml.Name{Local: "version"}, Value: gs.Version()},
		{Name: xml.Name{Local: "xmlns"}, Value: nsStream},
		{Name: xml.Name{Local: "from"}, Value: gs.JID().Domain},
		{Name: xml.Name{Local: "id"}, Value: gs.ID()},
	}
	if gs.JID().Username != "" {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "to"}, Value: gs.JID().String()})
	}
	return gs.SendToken(xml.StartElement{
		Name: xml.Name{Space: nsStream, Local: "stream"},
		Attr: attr})
}

func (gs *XGoingStream) Send(bs []byte) error {
	_, err := gs.conn.Write(bs)
	return err
}

func (gs *XGoingStream) SendToken(token xml.Token) error {
	err := gs.encoder.EncodeToken(token)
	gs.encoder.Flush()
	return err
}

func (gs *XGoingStream) SendElement(elem stravaganza.Element) error {
	_, err := gs.conn.Write([]byte(elem.GoString()))
	return err
}
