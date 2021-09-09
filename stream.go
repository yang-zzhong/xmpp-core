package xmppcore

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jackal-xmpp/stravaganza/v2"
)

var (
	ErrNotHeaderStart       = errors.New("not a start header")
	ErrNotForThisDomainHead = errors.New("not for this domain header")
)

type CommingStream interface {
	WaitHeader(header *PartAttr) error
	NextToken(token *xml.Token) error
	NextElement(elem *stravaganza.Element) error
}

type GoingStream interface {
	Open(attr *PartAttr) error
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

func (jid *JID) Equal(a *JID) bool {
	return jid.Username == a.Username && jid.Domain == a.Domain && jid.Resource == a.Resource
}

type XCommingStream struct {
	conn     io.Reader
	decoder  *xml.Decoder
	max      int
	isServer bool
}

func NewXCommingStream(conn Conn, isServer bool) *XCommingStream {
	return &XCommingStream{
		conn:     conn,
		decoder:  xml.NewDecoder(conn),
		max:      1024 * 1024 * 2,
		isServer: isServer,
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

func (xc *XCommingStream) WaitHeader(attr *PartAttr) error {
	var token xml.Token
	for {
		if err := xc.NextToken(&token); err != nil {
			return err
		}
		switch elem := token.(type) {
		case xml.StartElement:
			if xc.isServer {
				return attr.ParseToServer(elem)
			}
			return attr.ParseToClient(elem)
		default:
			continue
		}
	}
}

type XGoingStream struct {
	conn     io.Writer
	encoder  *xml.Encoder
	isServer bool
}

func NewXGoingStream(conn io.Writer, isServer bool) *XGoingStream {
	return &XGoingStream{conn, xml.NewEncoder(conn), isServer}
}

func (gs *XGoingStream) Open(attr *PartAttr) error {
	gs.Send([]byte("<?xml version='1.0'?>"))
	var elem xml.StartElement
	if gs.isServer {
		attr.ToClientHead(&elem)
		return gs.SendToken(elem)
	}
	attr.ToServerHead(&elem)
	return gs.SendToken(elem)
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
