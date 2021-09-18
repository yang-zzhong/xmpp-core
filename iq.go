package xmppcore

import (
	"errors"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type IqType string

const (
	IqGet    = IqType("get")
	IqSet    = IqType("set")
	IqResult = IqType("result")
	IqError  = IqType("error")
)

var (
	ErrNotIQ      = errors.New("not iq element")
	ErrIQType     = errors.New("not a valid iq type")
	ErrNotIqQuery = errors.New("not a iq query")
)

type IQ struct {
	Type IqType
	ID   string
	From string
	To   string
}

func IqFromElem(elem stravaganza.Element, iq *IQ) error {
	if elem.Name() != "iq" {
		return ErrNotIQ
	}
	iq.ID = elem.Attribute("id")
	var err error
	iq.Type, err = IqTypeFromStr(elem.Attribute("type"))
	if err != nil {
		return err
	}
	iq.From = elem.Attribute("from")
	iq.To = elem.Attribute("to")
	return nil
}

func (iq IQ) ToElemBuilder() *stravaganza.Builder {
	iqe := stravaganza.NewBuilder("iq").
		WithAttribute("type", string(iq.Type)).
		WithAttribute("id", iq.ID)
	if iq.From != "" {
		iqe.WithAttribute("from", iq.From)
	}
	if iq.To != "" {
		iqe.WithAttribute("to", iq.To)
	}
	return iqe
}

type IqQuery struct {
	IQ       IQ
	Xmlns    string
	Children []stravaganza.Element
}

func IqQueryFromElem(elem stravaganza.Element, iqq *IqQuery) error {
	if err := IqFromElem(elem, &iqq.IQ); err != nil {
		return err
	}
	query := elem.Child("query")
	if query == nil {
		return ErrNotIqQuery
	}
	iqq.Xmlns = query.Attribute("xmlns")
	iqq.Children = query.AllChildren()
	return nil
}

func (iqq IqQuery) ToElem(elem *stravaganza.Element) {
	b := iqq.IQ.ToElemBuilder()
	query := stravaganza.NewBuilder("query").WithAttribute("xmlns", iqq.Xmlns)
	if len(iqq.Children) > 0 {
		query.WithChildren(iqq.Children...)
	}
	*elem = b.WithChild(query.Build()).Build()
}

func IqTypeFromStr(typ string) (IqType, error) {
	t := IqType(typ)
	if t != IqGet && t != IqSet && t != IqResult && t != IqError {
		return t, ErrIQType
	}
	return t, nil
}
