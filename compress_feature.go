package xmppcore

import (
	"github.com/jackal-xmpp/stravaganza/v2"
)

const (
	nsCompress          = "http://jabber.org/protocol/compress"
	CESetupFailed       = "setup-failed"
	CEUnsupportedMethod = "unsupported-method"
)

func CompressErrorElem(name string) stravaganza.Element {
	return stravaganza.NewBuilder("failure").
		WithAttribute(stravaganza.Namespace, nsCompress).
		WithChild(stravaganza.NewBuilder(name).Build()).Build()
}

type CompressionFeature struct {
	supported map[string]BuildCompressor
}

func NewCompressFeature() *CompressionFeature {
	return &CompressionFeature{make(map[string]BuildCompressor)}
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
	return elem.Name() == "compress"
}

func (cf *CompressionFeature) Handle(elem stravaganza.Element, part Part) error {
	method := elem.Child("method")
	if method == nil || len(method.Text()) == 0 {
		return part.GoingStream().SendElement(CompressErrorElem(CESetupFailed))
	}
	build, ok := cf.supported[method.Text()]
	if !ok {
		return part.GoingStream().SendElement(CompressErrorElem(CEUnsupportedMethod))
	}
	if err := part.GoingStream().SendElement(stravaganza.NewBuilder("compressed").
		WithAttribute(stravaganza.Namespace, nsCompress).
		Build(),
	); err != nil {
		return err
	}
	part.Conn().StartCompress(build)
	return nil
}
