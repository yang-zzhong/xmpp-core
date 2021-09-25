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
	} else if b.Name() != "bind" || b.Attribute("xmlns") != NSBind {
		return ErrNotIqBind
	} else if r := b.Child("resource"); r != nil {
		ib.Resource = r.Text()
	} else if jid := b.Child("jid"); jid != nil {
		ib.JID = jid.Text()
	}
	return nil
}

func (ib IqBind) ToElem(elem *stravaganza.Element) {
	b := stravaganza.NewBuilder("bind").WithAttribute("xmlns", NSBind)
	if ib.IQ.Type == TypeSet && ib.Resource != "" {
		b.WithChild(stravaganza.NewBuilder("resource").WithText(ib.Resource).Build())
	} else if ib.IQ.Type == TypeResult && ib.JID != "" {
		b.WithChild(stravaganza.NewBuilder("jid").WithText(ib.JID).Build())
	}
	*elem = ib.IQ.ToElemBuilder().WithChild(b.Build()).Build()
}

type bindFeature struct {
	rsb       ResourceBinder
	handled   bool
	mandatory bool
	ib        IqBind
	IDAble
}

type ResourceBinder interface {
	BindResource(part Part, resource string) (string, error)
}

const (
	NSBind   = "urn:ietf:params:xml:ns:xmpp-bind"
	NSStanza = "urn:ietf:params:xml:ns:xmpp-stanzas"

	BEResourceConstraint = "wait: resource constraint"
	BENotAllowed         = "cancel: not allowed"
)

func BindErrFromError(id string, err error) StanzaErr {
	ss := strings.Split(err.Error(), ":")
	errTag := strings.Trim(ss[1], " ")
	return StanzaErr{
		Stanza: Stanza{
			ID:   id,
			Type: TypeError,
		}, Err: Err{
			Type: ss[0],
			Desc: []ErrDesc{{Tag: errTag, Xmlns: NSStanza}},
		},
	}
}

func BindFeature(rsb ResourceBinder) bindFeature {
	return bindFeature{IDAble: CreateIDAble(), rsb: rsb, handled: false, mandatory: false}
}

func (bf bindFeature) Mandatory() bool {
	return bf.mandatory
}

func (bf bindFeature) Elem() stravaganza.Element {
	elem := stravaganza.NewBuilder("bind").WithAttribute("xmlns", NSBind)
	if bf.mandatory {
		elem.WithChild(stravaganza.NewBuilder("required").Build())
	}
	return elem.Build()
}

func (bf *bindFeature) match(elem stravaganza.Element) bool {
	if err := bf.ib.FromElem(elem); err != nil {
		return false
	}
	if bf.ib.IQ.Type != TypeSet || bf.ib.IQ.Name != NameIQ || bf.ib.IQ.ID == "" {
		return false
	}
	return true
}

func (bf bindFeature) Handled() bool {
	return bf.handled
}

func (bf *bindFeature) Handle(elem stravaganza.Element, part Part) (cached bool, err error) {
	if !bf.match(elem) {
		return false, nil
	}
	bf.handled = true
	rsc, err := bf.rsb.BindResource(part, bf.ib.Resource)
	if err != nil {
		BindErrFromError(bf.ib.IQ.ID, err).ToElem(&elem)
		part.Channel().SendElement(elem)
		return
	}
	IqBind{IQ: Stanza{
		ID:   bf.ib.IQ.ID,
		Name: NameIQ,
		Type: TypeResult,
		To:   bf.ib.IQ.From,
		From: part.Attr().Domain}, JID: rsc}.ToElem(&elem)
	err = part.Channel().SendElement(elem)
	return
}
