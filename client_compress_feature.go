package xmppcore

import (
	"github.com/jackal-xmpp/stravaganza/v2"
)

type ClientCompressFeature struct {
	supported map[string]BuildCompressor
}

func NewClientCompressFeature() *ClientCompressFeature {
	return &ClientCompressFeature{make(map[string]BuildCompressor)}
}

func (ccf *ClientCompressFeature) Support(name string, b BuildCompressor) {
	ccf.supported[name] = b
}

func (ccf *ClientCompressFeature) Match(elem stravaganza.Element) bool {
	return elem.Name() == "compression"
}

func (ccf *ClientCompressFeature) Handle(elem stravaganza.Element, part Part) error {
	methods := elem.AllChildren()
	var selected string
	for _, m := range methods {
		if _, o := ccf.supported[m.Text()]; o {
			selected = m.Text()
			break
		}
	}
	compress := stravaganza.NewBuilder("compress").
		WithAttribute("xmlns", nsCompress).
		WithChild(stravaganza.NewBuilder("method").WithText(selected).Build()).Build()

	if err := part.Channel().SendElement(compress); err != nil {
		return err
	}
	if err := part.Channel().NextElement(&elem); err != nil {
		return err
	}
	if elem.Name() == "compressed" {
		part.Conn().StartCompress(ccf.supported[selected])
	}
	return nil
}
