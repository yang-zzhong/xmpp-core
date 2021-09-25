package xmppcore

import (
	"errors"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type clientBindFeature struct {
	rb       ResourceBinder
	resource string
	IDAble
}

func ClientBindFeature(rb ResourceBinder, resource string) clientBindFeature {
	return clientBindFeature{rb: rb, IDAble: CreateIDAble(), resource: resource}
}

func (cbf clientBindFeature) Match(elem stravaganza.Element) bool {
	return elem.Name() == "bind" && elem.Attribute("xmlns") == NSBind
}

func (cbf clientBindFeature) Handle(elem stravaganza.Element, part Part) (catched bool, err error) {
	if !cbf.Match(elem) {
		return false, nil
	}
	catched = true
	var src stravaganza.Element
	IqBind{IQ: Stanza{Name: NameIQ, ID: cbf.ID(), Type: TypeSet}, Resource: cbf.resource}.ToElem(&src)
	if err = part.Channel().SendElement(src); err != nil {
		part.Logger().Printf(LogError, "send bind message error: %s", err.Error())
		return
	}
	if err = part.Channel().NextElement(&elem); err != nil {
		return
	}
	var ie StanzaErr
	if e := ie.FromElem(elem, NameIQ); e == nil {
		err = ie.Err
	}
	var ib IqBind
	if err = ib.FromElem(elem); err != nil {
		return
	}
	if ib.IQ.ID != cbf.ID() {
		err = errors.New("not a bind result")
		return
	}
	var jid JID
	ParseJID(ib.JID, &jid)
	cbf.rb.BindResource(part, jid.Resource)
	return
}
