package xmppcore

import (
	"io"
	"net"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type PartFinder interface {
	FindPart(*JID) Part
}

type MessageRouter struct {
	hub    *MsgHub
	finder PartFinder
}

func NewMessageRouter(finder PartFinder) *MessageRouter {
	return &MessageRouter{finder: finder}
}

func (msg *MessageRouter) Match(elem stravaganza.Element) bool {
	return elem.Name() == "message"
}

func (msg *MessageRouter) Handle(elem stravaganza.Element, part Part) error {
	to := elem.Attribute("to")
	var jid JID
	if err := ParseJID(to, &jid); err != nil {
		return err
	}
	if jid.Domain == part.Attr().Domain {
		other := msg.finder.FindPart(&jid)
		if other == nil {
			return nil
		}
		return other.Channel().SendElement(elem)
	}
	if msg.hub == nil {
		msg.hub = NewMsgHub(part)
	}
	conn, err := net.Dial("tcp", jid.Domain+":5223")
	if err != nil {
		return err
	}
	mp := msg.outClient(conn, &jid, part)
	return mp.Channel().SendElement(elem)
}

func (msg *MessageRouter) outClient(conn net.Conn, jid *JID, c2s Part) Part {
	part := NewClientPart(NewTcpConn(conn, true), c2s.Logger(), &PartAttr{JID: *jid, Domain: c2s.Attr().Domain})
	part.WithFeature(&ClientTlsFeature{})
	ccf := NewClientCompressFeature()
	ccf.Support(ZLIB, func(rw io.ReadWriter) Compressor {
		return NewCompZlib(rw)
	})
	part.WithFeature(ccf)
	go func() {
		part.Run()
	}()
	part.WithElemHandler(msg.hub)
	msg.hub.AddRemote(jid.Domain, part)
	return part
}

type MsgHub struct {
	c2s Part
	out map[string]Part
}

func NewMsgHub(c2s Part) *MsgHub {
	return &MsgHub{c2s: c2s, out: make(map[string]Part)}
}

func (msgHub *MsgHub) Match(_ stravaganza.Element) bool {
	return true
}

func (msgHub *MsgHub) Handle(elem stravaganza.Element, _ Part) error {
	return msgHub.c2s.Channel().SendElement(elem)
}

func (msgHub *MsgHub) AddRemote(domain string, out Part) {
	if _, ok := msgHub.out[domain]; ok {
		return
	}
	msgHub.out[domain] = out
}
