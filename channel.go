package xmppcore

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackal-xmpp/stravaganza/v2"
)

var (
	ErrNotHeaderStart       = errors.New("not a start header")
	ErrNotForThisDomainHead = errors.New("not for this domain header")
)

type Receiver interface {
	NextElement(elem *stravaganza.Element) error
	next() (interface{}, error)
}

type Sender interface {
	Send([]byte) error
	SendToken(xml.Token) error
	SendElement(stravaganza.Element) error
}

type Channel interface {
	Receiver
	Sender
	WaitHeader(*xml.StartElement) error
	Open(attr *PartAttr) error
	SetLogger(Logger)
	Close()
}

const (
	nsStream  = "http://etherx.jabber.org/streams"
	nsFraming = "urn:ietf:params:xml:ns:xmpp-framing"

	stateInit = iota
	stateWSOpened
	stateTCPOpened
	stateClosed
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

type XChannel struct {
	conn           io.ReadWriteCloser
	decoder        *xml.Decoder
	encoder        *xml.Encoder
	isServer       bool
	state          int
	waitSecOnClose int
	parser         *Parser
	logger         Logger
}

func NewXChannel(conn Conn, isServer bool) *XChannel {
	return &XChannel{
		conn:           conn,
		decoder:        xml.NewDecoder(conn),
		encoder:        xml.NewEncoder(conn),
		isServer:       isServer,
		state:          stateInit,
		parser:         NewParser(conn, 1024*1024*2),
		waitSecOnClose: 2,
	}
}

func (xc *XChannel) SetLogger(logger Logger) {
	xc.logger = logger
}

func (xc *XChannel) WaitSecOnClose(sec int) {
	xc.waitSecOnClose = sec
}

func (xc *XChannel) WaitHeader(header *xml.StartElement) error {
	for {
		i, err := xc.next()
		if err != nil {
			return err
		}
		switch t := i.(type) {
		case stravaganza.Element:
			return fmt.Errorf("unexpected element: %s", t.GoString())
		case xml.StartElement:
			*header = t
			return nil
		}
	}
}

func (xc *XChannel) NextElement(elem *stravaganza.Element) error {
	var err error
	*elem, err = xc.parser.NextElement()
	if err == nil {
		xc.logElement("RECV", *elem)
	}
	return err
}

func (xc *XChannel) next() (interface{}, error) {
	i, e := xc.parser.Next()
	if e != nil {
		return i, e
	}
	if xc.logger != nil {
		switch t := i.(type) {
		case xml.StartElement:
			xc.logToken("RECV", t)
		case stravaganza.Element:
			xc.logElement("RECV", t)
		default:
			xc.logger.Printf(Debug, "RECV: %s", t)
		}
	}
	return i, nil
}

func (xc *XChannel) Close() {
	var token xml.Token
	switch xc.state {
	case stateInit:
		xc.conn.Close()
		return
	case stateWSOpened:
		token = xml.EndElement{Name: xml.Name{Local: "open", Space: nsFraming}}
	case stateTCPOpened:
		token = xml.EndElement{Name: xml.Name{Local: "stream", Space: nsStream}}
	case stateClosed:
		return
	}
	xc.SendToken(token)
	xc.state = stateClosed
	time.AfterFunc(time.Second*time.Duration(xc.waitSecOnClose), func() {
		xc.conn.Close()
	})
}

func (gs *XChannel) Open(attr *PartAttr) error {
	gs.Send([]byte("<?xml version='1.0'?>"))
	var elem xml.StartElement
	if gs.isServer {
		attr.ToClientHead(&elem)
		return gs.SendToken(elem)
	}
	attr.ToServerHead(&elem)
	return gs.SendToken(elem)
}

func (gs *XChannel) Send(bs []byte) error {
	sent := 0
	total := len(bs)
	for sent < total {
		s, err := gs.conn.Write(bs)
		if err != nil {
			return err
		}
		sent = sent + s
	}
	if gs.logger != nil {
		tmp := "SEND: \n%s"
		if gs.isServer {
			tmp = "server " + tmp
		} else {
			tmp = "client " + tmp
		}
		gs.logger.Printf(Debug, tmp, string(bs))
	}
	return nil
}

func (gs *XChannel) SendToken(token xml.Token) error {
	err := gs.encoder.EncodeToken(token)
	gs.encoder.Flush()
	gs.logToken("SEND", token)
	return err
}

func (gs *XChannel) SendElement(elem stravaganza.Element) error {
	_, err := gs.conn.Write([]byte(elem.GoString()))
	gs.logElement("SEND", elem)
	return err
}

func (gs *XChannel) logElement(leading string, elem stravaganza.Element) {
	if gs.logger == nil {
		return
	}
	tmp := leading + ": \n %s"
	if gs.isServer {
		tmp = "server " + tmp
	} else {
		tmp = "client " + tmp
	}
	gs.logger.Printf(Debug, tmp, elem.GoString())
}

func (gs *XChannel) logToken(leading string, token xml.Token) {
	if gs.logger == nil {
		return
	}
	tmp := leading + ": "
	if gs.isServer {
		tmp = "server " + tmp
	} else {
		tmp = "client " + tmp
	}
	gs.logger.Printf(Debug, tmp)
	encoder := xml.NewEncoder(gs.logger.Writer())
	encoder.EncodeToken(token)
	encoder.Flush()
	gs.logger.Writer().Write([]byte("\n"))
}
