package xmppcore

import (
	"github.com/jackal-xmpp/stravaganza/v2"
)

const (
	NSCompress = "http://jabber.org/protocol/compress"

	CESetupFailed       = "setup-failed"
	CEUnsupportedMethod = "unsupported-method"

	ZLIB = "zlib"
	LZW  = "lzw"
)

type CompressFailure struct {
	DescTag string
}

func (cf *CompressFailure) FromElem(elem stravaganza.Element) error {
	f := Failure{}
	if err := f.FromElem(elem, NSCompress); err != nil {
		return err
	}
	cf.DescTag = f.DescTag
	return nil
}

func (cf CompressFailure) ToElem(elem *stravaganza.Element) {
	Failure{Xmlns: NSCompress, DescTag: cf.DescTag}.ToElem(elem)
}

type compressionFeature struct {
	supported map[string]BuildCompressor
	handled   bool
	mandatory bool
	IDAble
}

func CompressFeature() compressionFeature {
	return compressionFeature{supported: make(map[string]BuildCompressor), handled: false, mandatory: false, IDAble: CreateIDAble()}
}

func (cf compressionFeature) Mandatory() bool {
	return cf.mandatory
}

func (cf compressionFeature) Handled() bool {
	return cf.handled
}

func (cf compressionFeature) Elem() stravaganza.Element {
	children := []stravaganza.Element{}
	for supported := range cf.supported {
		children = append(children, stravaganza.NewBuilder("method").WithText(supported).Build())
	}
	return stravaganza.NewBuilder("compression").
		WithAttribute(stravaganza.Namespace, NSCompress).
		WithChildren(children...).Build()
}

func (cf *compressionFeature) Support(name string, build BuildCompressor) {
	cf.supported[name] = build
}

func (cf compressionFeature) Match(elem stravaganza.Element) bool {
	return elem.Name() == "compress" && elem.Attribute("xmlns") == NSCompress
}

func (cf *compressionFeature) Handle(elem stravaganza.Element, part Part) (catched bool, err error) {
	if !cf.Match(elem) {
		return false, nil
	}
	catched = true
	cf.handled = true
	method := elem.Child("method")
	if method == nil || len(method.Text()) == 0 {
		CompressFailure{DescTag: CESetupFailed}.ToElem(&elem)
		err = part.Channel().SendElement(elem)
		return
	}
	build, ok := cf.supported[method.Text()]
	if !ok {
		CompressFailure{DescTag: CEUnsupportedMethod}.ToElem(&elem)
		err = part.Channel().SendElement(elem)
		return
	}
	if err = part.Channel().SendElement(stravaganza.NewBuilder("compressed").
		WithAttribute(stravaganza.Namespace, NSCompress).
		Build()); err != nil {
		return
	}
	part.Conn().StartCompress(build)
	return
}
