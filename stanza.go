// Copyright (c) 2021 Yang,Zhong
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package xmppcore

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type StanzaType string

const (
	NameIQ       = "iq"
	NamePresence = "presence"
	NameMsg      = "message"
	// for iq
	TypeGet    = StanzaType("get")
	TypeSet    = StanzaType("set")
	TypeResult = StanzaType("result")
	TypeError  = StanzaType("error")

	// for presence
	TypeSub         = StanzaType("subscribe")
	TypeUnsub       = StanzaType("unsubscribe")
	TypeSubed       = StanzaType("subscribed")
	TypeUnsubed     = StanzaType("unsubscribed")
	TypeUnavailable = StanzaType("unavailable")
)

var (
	ErrNotRequiredFailure = errors.New("parsed element not a required failure")
	ErrNotStanzaErr       = errors.New("not stanza error")
)

type ElementableError interface {
	error
	ToElem(*stravaganza.Element)
}

type ErrDesc struct {
	Tag   string
	Xmlns string
}

type Err struct {
	Type string
	Desc []ErrDesc
}

type Stanza struct {
	Name string
	Type StanzaType
	ID   string
	From string
	To   string
}

func (err *Err) FromElem(elem stravaganza.Element) {
	err.Type = elem.Attribute("type")
	err.Desc = []ErrDesc{}
	for _, desc := range elem.AllChildren() {
		err.Desc = append(err.Desc, ErrDesc{Tag: desc.Name(), Xmlns: desc.Attribute("xmlns")})
	}
}

func (err Err) ToElem(elem *stravaganza.Element) {
	ee := stravaganza.NewBuilder("error")
	if err.Type != "" {
		ee.WithAttribute("type", err.Type)
	}
	elems := []stravaganza.Element{}
	for _, desc := range err.Desc {
		elems = append(elems, stravaganza.NewBuilder(desc.Tag).WithAttribute("xmlns", desc.Xmlns).Build())
	}
	*elem = ee.WithChildren(elems...).Build()
}

func (err Err) Error() string {
	msg := ""
	for i, desc := range err.Desc {
		if i < len(err.Desc)-1 {
			msg = msg + ":"
		}
		msg = msg + strings.ReplaceAll(desc.Tag, "-", " ")
	}
	return msg
}

func (stanza *Stanza) FromElem(elem stravaganza.Element, name string) error {
	stanza.Name = elem.Name()
	if stanza.Name != name {
		return fmt.Errorf("not a stanza %s element", name)
	}
	stanza.ID = elem.Attribute("id")
	stanza.Type = StanzaType(elem.Attribute("type"))
	stanza.From = elem.Attribute("from")
	stanza.To = elem.Attribute("to")
	return nil
}

func (stanza Stanza) ToElemBuilder() *stravaganza.Builder {
	iqe := stravaganza.NewBuilder(stanza.Name).
		WithAttribute("type", string(stanza.Type)).
		WithAttribute("id", stanza.ID)
	if stanza.From != "" {
		iqe.WithAttribute("from", stanza.From)
	}
	if stanza.To != "" {
		iqe.WithAttribute("to", stanza.To)
	}
	return iqe
}

type Failure struct {
	Xmlns    string
	DescTag  string
	More     string
	MoreLang string
}

func (f *Failure) FromElem(elem stravaganza.Element, xmlns string) error {
	if elem.Name() != "failure" {
		return ErrNotRequiredFailure
	}
	f.Xmlns = elem.Attribute("xmlns")
	if f.Xmlns != xmlns {
		return ErrNotRequiredFailure
	}
	desc := elem.AllChildren()
	if len(desc) > 0 {
		f.DescTag = desc[0].Name()
		if len(desc) > 1 {
			f.More = desc[1].Text()
			f.MoreLang = desc[1].Attribute("xml:lang")
		}
	}
	return nil
}

func (f Failure) ToElem(elem *stravaganza.Element) {
	err := []stravaganza.Element{stravaganza.NewBuilder(f.DescTag).WithAttribute("xmlns", f.Xmlns).Build()}
	if f.More != "" {
		more := stravaganza.NewBuilder("text").WithText(f.More)
		if f.MoreLang != "" {
			more.WithAttribute("xml:lang", f.MoreLang)
		}
		err = append(err, more.Build())
	}
	*elem = stravaganza.NewBuilder("failure").WithChildren(err...).Build()
}

func (f Failure) Error() string {
	return strings.ReplaceAll(f.DescTag, "-", " ")
}

type StanzaErr struct {
	Stanza Stanza
	Err    Err
}

func (se StanzaErr) ToElem(elem *stravaganza.Element) {
	var err stravaganza.Element
	se.Err.ToElem(&err)
	*elem = se.Stanza.ToElemBuilder().WithChild(err).Build()
}

func (se *StanzaErr) FromElem(elem stravaganza.Element, name string) error {
	if err := se.Stanza.FromElem(elem, name); err != nil {
		return err
	}
	if se.Stanza.Type != TypeError {
		return ErrNotStanzaErr
	}
	if err := elem.Child("error"); err != nil {
		se.Err.FromElem(elem.Child("error"))
	}
	return nil
}

type IqErrHandler interface {
	HandleIqError(StanzaErr, Part) error
}

type StanzaErrHandler struct {
	Name             string
	lastMatchedError StanzaErr
	handler          IqErrHandler
}

func (seh *StanzaErrHandler) Match(elem stravaganza.Element) bool {
	if err := seh.lastMatchedError.FromElem(elem, seh.Name); err != nil {
		return false
	}
	return true
}

func (seh StanzaErrHandler) Handle(_ stravaganza.Element, part Part) error {
	return seh.handler.HandleIqError(seh.lastMatchedError, part)
}
