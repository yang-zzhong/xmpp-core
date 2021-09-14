package xmppcore

import (
	"errors"

	"github.com/google/uuid"
	"github.com/jackal-xmpp/stravaganza/v2"
)

type ClientBindFeature struct {
	id string
	rb ResourceBinder
}

func NewClientBindFeature(rb ResourceBinder) *ClientBindFeature {
	return &ClientBindFeature{id: uuid.New().String(), rb: rb}
}

func (cbf *ClientBindFeature) Match(elem stravaganza.Element) bool {
	return elem.Name() == "bind"
}

func (cbf *ClientBindFeature) Handle(elem stravaganza.Element, part Part) error {
	if elem.Attribute("xmlns") != nsBind {
		return errors.New("wrong name bind namespace")
	}
	src := stravaganza.NewBuilder("iq").
		WithAttribute("id", cbf.id).
		WithAttribute("type", "set").
		WithChild(stravaganza.NewBuilder("bind").WithAttribute("xmlns", nsBind).Build()).Build()
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
	if elem.Name() != "iq" || elem.Attribute("id") != cbf.id || elem.Attribute("type") != "result" {
		return errors.New("not a bind result")
	}
	bind := elem.Child("bind")
	if bind == nil {
		return errors.New("bind result error")
	}
	jid := bind.Child("jid")
	if jid == nil {
		return errors.New("bind result error")
	}
	cbf.rb.BindResource(part, jid.Text())

	return nil
}
