package xmppcore

import "github.com/jackal-xmpp/stravaganza/v2"

type BindFeature struct {
	resolved bool
}

const (
	nsBind = "urn:ietf:params:xml:ns:xmpp-bind"
)

func NewBindFeature() *BindFeature {
	return &BindFeature{false}
}

func (bf *BindFeature) notifyPeer(sender GoingStream) error {
	elem := stravaganza.NewBuilder("features").WithChild(
		stravaganza.NewBuilder("bind").WithAttribute("xmlns", nsBind).Build(),
	).Build()
	return sender.SendElement(elem)
}

func (bf *BindFeature) Reopen() bool {
	return true
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

func (bf *BindFeature) Resolve(part Part) error {
	bf.notifyPeer(part.GoingStream())
	var elem stravaganza.Element
	if err := part.CommingStream().NextElement(&elem); err != nil {
		return nil
	}
	if bf.Match(elem) {
		bf.Handle(elem, part)
	}
	bf.resolved = true
	return nil
}

func (bf *BindFeature) Resolved() bool {
	return bf.resolved
}
