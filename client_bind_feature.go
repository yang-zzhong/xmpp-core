package xmppcore

import (
	"errors"
	"fmt"

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
	part.WithElemHandler(NewBindResultHandler(cbf))
	return nil
}

type BindResultHandler struct {
	bf *ClientBindFeature
}

func NewBindResultHandler(cbf *ClientBindFeature) *BindResultHandler {
	return &BindResultHandler{cbf}
}

func (brh *BindResultHandler) Match(elem stravaganza.Element) bool {
	return elem.Name() == "iq" && elem.Attribute("id") == brh.bf.id
}

func (brh *BindResultHandler) Handle(elem stravaganza.Element, part Part) error {
	if elem.Attribute("type") == "error" {
		// handle error
	}
	bind := elem.Child("bind")
	fmt.Printf("%s\n", bind.GoString())
	brh.bf.rb.BindResource(part, "")
	return nil
}
