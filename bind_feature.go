package xmppcore

import "github.com/jackal-xmpp/stravaganza/v2"

type BindFeature struct {
	rsb ResourceBinder
}

type ResourceBinder interface {
	BindResource(userdomain, resource string) error
}

const (
	nsBind = "urn:ietf:params:xml:ns:xmpp-bind"
)

func NewBindFeature(rsb ResourceBinder) *BindFeature {
	return &BindFeature{rsb: rsb}
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
	jid := part.Attr().JID.String()
	part.Attr().JID.Resource = bf.resource(elem)
	bf.rsb.BindResource(jid, part.Attr().JID.Resource)
	err := part.GoingStream().SendElement(stravaganza.NewBuilder("iq").
		WithAttribute("type", "result").
		WithAttribute("id", id).WithChild(
		stravaganza.NewBuilder("bind").
			WithAttribute("xmlns", nsBind).
			WithChild(
				stravaganza.NewBuilder("jid").
					WithText(part.Attr().JID.String()).
					Build(),
			).Build(),
	).Build())
	return err
}

func (bf *BindFeature) resource(elem stravaganza.Element) string {
	return "resource"
}
