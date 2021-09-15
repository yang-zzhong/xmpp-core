package xmppcore

import (
	"errors"
	"strings"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type BindFeature struct {
	rsb       ResourceBinder
	handled   bool
	mandatory bool
	*IDAble
}

type ResourceBinder interface {
	BindResource(part Part, resource string) (string, error)
}

const (
	nsBind   = "urn:ietf:params:xml:ns:xmpp-bind"
	nsStanza = "urn:ietf:params:xml:ns:xmpp-stanzas"

	BEResourceConstraint = "wait: resource constraint"
	BENotAllowed         = "cancel: not allowed"
)

func BindErrorElem(id, tag, typ string) stravaganza.Element {
	return stravaganza.NewIQBuilder().
		WithAttribute("id", id).
		WithAttribute("type", "error").
		WithChild(stravaganza.NewBuilder("error").
			WithAttribute("type", typ).
			WithChild(stravaganza.NewBuilder(tag).WithAttribute("xmlns", nsStanza).Build()).Build()).Build()
}

func ErrBind(err string) error {
	return errors.New(err)
}

func BindErrorElemFromError(id string, err error) stravaganza.Element {
	ss := strings.Split(err.Error(), ":")
	errTag := strings.Trim(ss[1], " ")
	return BindErrorElem(id, errTag, ss[0])
}

func NewBindFeature(rsb ResourceBinder, mandatory bool) *BindFeature {
	return &BindFeature{IDAble: NewIDAble(), rsb: rsb, handled: false, mandatory: mandatory}
}

func (bf *BindFeature) Mandatory() bool {
	return bf.mandatory
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

func (bf *BindFeature) Handled() bool {
	return bf.handled
}

func (bf *BindFeature) Handle(elem stravaganza.Element, part Part) error {
	bf.handled = true
	id := elem.Attribute("id")
	rsc, err := bf.rsb.BindResource(part, bf.resource(elem))
	if err != nil {
		part.Channel().SendElement(BindErrorElemFromError(id, err))
		return err
	}
	return part.Channel().SendElement(stravaganza.NewBuilder("iq").
		WithAttribute("type", "result").
		WithAttribute("id", id).WithChild(
		stravaganza.NewBuilder("bind").
			WithAttribute("xmlns", nsBind).
			WithChild(
				stravaganza.NewBuilder("jid").
					WithText(rsc).
					Build(),
			).Build(),
	).Build())
}

func (bf *BindFeature) resource(elem stravaganza.Element) string {
	return ""
}
