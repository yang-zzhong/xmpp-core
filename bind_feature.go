package xmppcore

import (
	"errors"
	"strings"

	"github.com/jackal-xmpp/stravaganza/v2"
)

var (
	ErrNotIqBind = errors.New("not iq bind")
)

type IqBind struct {
	IQ       Stanza
	Resource string
	JID      string
}

func (ib *IqBind) FromElem(elem stravaganza.Element) error {
	if err := ib.IQ.FromElem(elem, NameIQ); err != nil {
		return err
	}
	if b := elem.Child("bind"); b == nil {
		return ErrNotIqBind
	} else if b.Name() != "bind" || b.Attribute("xmlns") != nsBind {
		return ErrNotIqBind
	} else if r := b.Child("resource"); r != nil {
		ib.Resource = r.Text()
	} else if jid := b.Child("jid"); jid != nil {
		ib.JID = jid.Text()
	}
	return nil
}

func (ib IqBind) ToElem(elem *stravaganza.Element) {
	b := stravaganza.NewBuilder("bind").WithAttribute("xmlns", nsBind)
	if ib.IQ.Type == StanzaSet && ib.Resource != "" {
		b.WithChild(stravaganza.NewBuilder("resource").WithText(ib.Resource).Build())
	} else if ib.IQ.Type == StanzaResult && ib.JID != "" {
		b.WithChild(stravaganza.NewBuilder("jid").WithText(ib.JID).Build())
	}
	*elem = ib.IQ.ToElemBuilder().WithChild(b.Build()).Build()
}

type BindErr struct {
	ID      string
	Type    string
	DescTag string
}

func (be BindErr) ToElem(elem *stravaganza.Element) {
	var err stravaganza.Element
	Err{Type: be.Type, Xmlns: nsStanza, DescTag: be.DescTag}.ToElem(&err)
	*elem = Stanza{Name: NameIQ, ID: be.ID, Type: StanzaError}.
		ToElemBuilder().
		WithChild(err).Build()

}

type BindFeature struct {
	rsb       ResourceBinder
	handled   bool
	mandatory bool
	ib        IqBind
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

func BindErrFromError(id string, err error) BindErr {
	ss := strings.Split(err.Error(), ":")
	errTag := strings.Trim(ss[1], " ")
	return BindErr{ID: id, Type: ss[0], DescTag: errTag}
}

func NewBindFeature(rsb ResourceBinder) *BindFeature {
	return &BindFeature{IDAble: NewIDAble(), rsb: rsb, handled: false, mandatory: false}
}

func (bf *BindFeature) Mandatory() bool {
	return bf.mandatory
}

func (bf *BindFeature) Elem() stravaganza.Element {
	elem := stravaganza.NewBuilder("bind").WithAttribute("xmlns", nsBind)
	if bf.mandatory {
		elem.WithChild(stravaganza.NewBuilder("required").Build())
	}
	return elem.Build()
}

func (bf *BindFeature) Match(elem stravaganza.Element) bool {
	if err := bf.ib.FromElem(elem); err != nil {
		return false
	}
	if bf.ib.IQ.Type != StanzaSet || bf.ib.IQ.Name != NameIQ || bf.ib.IQ.ID == "" {
		return false
	}
	return true
}

func (bf *BindFeature) Handled() bool {
	return bf.handled
}

func (bf *BindFeature) Handle(elem stravaganza.Element, part Part) error {
	bf.handled = true
	rsc, err := bf.rsb.BindResource(part, bf.ib.Resource)
	if err != nil {
		BindErrFromError(bf.ib.IQ.ID, err).ToElem(&elem)
		part.Channel().SendElement(elem)
		return err
	}
	iq := bf.ib.IQ
	iq.Type = StanzaResult
	iq.To = iq.From
	iq.From = part.Attr().Domain
	IqBind{IQ: iq, JID: rsc}.ToElem(&elem)
	return part.Channel().SendElement(elem)
}
