package xmppcore

import (
	"errors"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type ClientBindFeature struct {
	rb       ResourceBinder
	resource string
	*IDAble
}

func NewClientBindFeature(rb ResourceBinder, resource string) *ClientBindFeature {
	return &ClientBindFeature{rb: rb, IDAble: NewIDAble(), resource: resource}
}

func (cbf *ClientBindFeature) Match(elem stravaganza.Element) bool {
	return elem.Name() == "bind" && elem.Attribute("xmlns") == nsBind
}

func (cbf *ClientBindFeature) Handle(elem stravaganza.Element, part Part) error {
	var src stravaganza.Element
	IqBind{IQ: IQ{ID: cbf.ID(), Type: IqSet}, Resource: cbf.resource}.ToElem(&src)

	if err := part.Channel().SendElement(src); err != nil {
		part.Logger().Printf(Error, "send bind message error: %s", err.Error())
		return err
	}
	if err := part.Channel().NextElement(&elem); err != nil {
		return err
	}
	if elem.Attribute("type") == "error" {
		return errors.New("server bind error")
	}
	var ib IqBind
	if err := IqBindFromElem(elem, &ib); err != nil {
		return err
	}
	if ib.IQ.ID != cbf.ID() || ib.IQ.Type != IqResult {
		return errors.New("not a bind result")
	}
	var jid JID
	ParseJID(ib.JID, &jid)
	cbf.rb.BindResource(part, jid.Resource)
	return nil
}
