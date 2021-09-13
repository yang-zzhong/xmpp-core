package xmppcore

import (
	"crypto/tls"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type ClientTlsFeature struct {
	conf *tls.Config
}

func NewClientTlsFeature(conf *tls.Config) *ClientTlsFeature {
	return &ClientTlsFeature{conf: conf}
}

func (ctf *ClientTlsFeature) Match(elem stravaganza.Element) bool {
	return elem.Name() == "starttls"
}

func (ctf *ClientTlsFeature) Handle(_ stravaganza.Element, part Part) error {
	elem := stravaganza.NewBuilder("starttls").WithAttribute("xmlns", nsTLS).Build()
	if err := part.Channel().SendElement(elem); err != nil {
		return err
	}
	if err := part.Channel().NextElement(&elem); err != nil {
		return err
	}
	if elem.Name() != "proceed" {
		return nil
	}
	part.Conn().StartTLS(ctf.conf)
	return nil
}
