package xmppcore

import (
	"crypto/tls"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type ClientTlsFeature struct{}

func (ctf *ClientTlsFeature) Match(elem stravaganza.Element) bool {
	return elem.Name() == "starttls"
}

func (ctf *ClientTlsFeature) Handle(_ stravaganza.Element, part Part) error {
	elem := stravaganza.NewBuilder("starttls").WithAttribute("xmlns", nsTLS).Build()
	if err := part.GoingStream().SendElement(elem); err != nil {
		return err
	}
	if err := part.CommingStream().NextElement(&elem); err != nil {
		return err
	}
	if elem.Name() != "proceed" {
		return nil
	}
	part.Conn().StartTLS(&tls.Config{})
	return nil
}
