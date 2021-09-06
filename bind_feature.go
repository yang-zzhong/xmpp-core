package xmppcore

import "github.com/jackal-xmpp/stravaganza/v2"

type BindFeature struct{}

const (
	nsBind = "urn:ietf:params:xml:ns:xmpp-bind"
)

func NewBindFeature() *BindFeature {
	return &BindFeature{}
}

func (bf *BindFeature) Elem() stravaganza.Element {
	return stravaganza.NewBuilder("bind").WithAttribute("xmlns", nsBind).Build()
}

func (bf *BindFeature) Match(elem stravaganza.Element) bool {
	if elem.Name() != "iq" {
		return false
	}
	if elem.Attribute("type") != "set" || elem.Attribute("id") == "" {
		return false
	}
	bind := elem.Child("bind")
	if bind == nil {
		return false
	}
	if bind.Attribute("xmlns") != nsBind {
		return false
	}
	return true
}

func (bf *BindFeature) Handle(elem stravaganza.Element, part Part) error {
	id := elem.Attribute("id")
	err := part.GoingStream().SendElement(stravaganza.NewBuilder("iq").
		WithAttribute("type", "result").
		WithAttribute("id", id).WithChild(
		stravaganza.NewBuilder("bind").
			WithAttribute("xmlns", nsBind).
			WithChild(
				stravaganza.NewBuilder("jid").
					WithText(part.CommingStream().JID().String()).
					Build(),
			).Build(),
	).Build())
	return err
}
