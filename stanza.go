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

	StanzaGet    = StanzaType("get")
	StanzaSet    = StanzaType("set")
	StanzaResult = StanzaType("result")
	StanzaError  = StanzaType("error")
)

var (
	ErrNotRequiredFailure = errors.New("parsed element not a required failure")
	ErrStanzaType         = errors.New("stanza type error")
)

type Err struct {
	Type    string
	DescTag string
	Xmlns   string
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
	ec := elem.AllChildren()
	if len(ec) > 0 {
		err.DescTag = ec[0].Name()
		err.Xmlns = ec[0].Attribute("xmlns")
	}
}

func (err Err) ToElem(elem *stravaganza.Element) {
	ee := stravaganza.NewBuilder("error")
	if err.Type != "" {
		ee.WithAttribute("type", err.Type)
	}
	if err.DescTag != "" {
		ee.WithChild(stravaganza.NewBuilder(err.DescTag).WithAttribute("xmlns", err.Xmlns).Build())
	}
	*elem = ee.Build()
}

func (err *Err) Error() string {
	return strings.ReplaceAll(err.DescTag, "-", " ")
}

func (stanza *Stanza) FromElem(elem stravaganza.Element, name string) error {
	stanza.Name = elem.Name()
	if stanza.Name != name {
		return fmt.Errorf("not a stanza %s element", name)
	}
	stanza.ID = elem.Attribute("id")
	var err error
	stanza.Type, err = GetStanzaType(elem.Attribute("type"))
	if err != nil {
		return err
	}
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

func GetStanzaType(typ string) (StanzaType, error) {
	t := StanzaType(typ)
	if t != StanzaGet && t != StanzaSet && t != StanzaResult && t != StanzaError {
		return t, ErrStanzaType
	}
	return t, nil
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
