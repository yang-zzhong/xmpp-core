package xmppcore

import (
	"github.com/jackal-xmpp/stravaganza/v2"
)

type clientCompressFeature struct {
	supported map[string]BuildCompressor
	IDAble
}

func ClientCompressFeature() clientCompressFeature {
	return clientCompressFeature{supported: make(map[string]BuildCompressor), IDAble: CreateIDAble()}
}

func (ccf *clientCompressFeature) Support(name string, b BuildCompressor) {
	ccf.supported[name] = b
}

func (ccf clientCompressFeature) Match(elem stravaganza.Element) bool {
	return elem.Name() == "compression"
}

func (ccf clientCompressFeature) Handle(elem stravaganza.Element, part Part) (catched bool, err error) {
	if !ccf.Match(elem) {
		return false, nil
	}
	catched = true
	methods := elem.AllChildren()
	var selected string
	for _, m := range methods {
		if _, o := ccf.supported[m.Text()]; o {
			selected = m.Text()
			break
		}
	}
	compress := stravaganza.NewBuilder("compress").
		WithAttribute("xmlns", NSCompress).
		WithChild(stravaganza.NewBuilder("method").WithText(selected).Build()).Build()

	if err = part.Channel().SendElement(compress); err != nil {
		return
	}
	if err = part.Channel().NextElement(&elem); err != nil {
		return
	}
	if elem.Name() == "compressed" {
		part.Conn().StartCompress(ccf.supported[selected])
	}
	return
}
