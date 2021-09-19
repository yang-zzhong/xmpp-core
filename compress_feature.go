package xmppcore

import (
	"github.com/jackal-xmpp/stravaganza/v2"
)

const (
	nsCompress = "http://jabber.org/protocol/compress"

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
	if err := f.FromElem(elem, nsCompress); err != nil {
		return err
	}
	cf.DescTag = f.DescTag
	return nil
}

func (cf CompressFailure) ToElem(elem *stravaganza.Element) {
	Failure{Xmlns: nsCompress, DescTag: cf.DescTag}.ToElem(elem)
}

type CompressionFeature struct {
	supported map[string]BuildCompressor
	handled   bool
	mandatory bool
	*IDAble
}

func NewCompressFeature() *CompressionFeature {
	return &CompressionFeature{supported: make(map[string]BuildCompressor), handled: false, mandatory: false, IDAble: NewIDAble()}
}

func (cf *CompressionFeature) Mandatory() bool {
	return cf.mandatory
}

func (cf *CompressionFeature) Handled() bool {
	return cf.handled
}

func (cf *CompressionFeature) Elem() stravaganza.Element {
	children := []stravaganza.Element{}
	for supported := range cf.supported {
		children = append(children, stravaganza.NewBuilder("method").WithText(supported).Build())
	}
	return stravaganza.NewBuilder("compression").
		WithAttribute(stravaganza.Namespace, nsCompress).
		WithChildren(children...).Build()
}

func (cf *CompressionFeature) Support(name string, build BuildCompressor) {
	cf.supported[name] = build
}

func (cf *CompressionFeature) Match(elem stravaganza.Element) bool {
	return elem.Name() == "compress" && elem.Attribute("xmlns") == nsCompress
}

func (cf *CompressionFeature) Handle(elem stravaganza.Element, part Part) error {
	method := elem.Child("method")
	if method == nil || len(method.Text()) == 0 {
		CompressFailure{DescTag: CESetupFailed}.ToElem(&elem)
		return part.Channel().SendElement(elem)
	}
	build, ok := cf.supported[method.Text()]
	if !ok {
		CompressFailure{DescTag: CEUnsupportedMethod}.ToElem(&elem)
		return part.Channel().SendElement(elem)
	}
	if err := part.Channel().SendElement(stravaganza.NewBuilder("compressed").
		WithAttribute(stravaganza.Namespace, nsCompress).
		Build()); err != nil {
		return err
	}
	part.Conn().StartCompress(build)
	return nil
}
