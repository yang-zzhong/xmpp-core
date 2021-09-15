package xmppcore

import (
	"crypto/tls"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type TlsFeature struct {
	certFile  string
	keyFile   string
	mandatory bool

	handled bool
	*IDAble
}

func TlsFailureElem() stravaganza.Element {
	return stravaganza.NewBuilder("failure").WithAttribute("xmlns", nsTLS).Build()
}

const (
	nsTLS = "urn:ietf:params:xml:ns:xmpp-tls"
)

func NewTlsFeature(certFile, keyFile string, mandatory bool) *TlsFeature {
	return &TlsFeature{
		certFile:  certFile,
		keyFile:   keyFile,
		mandatory: mandatory,
		handled:   false,
		IDAble:    NewIDAble()}
}

func (tf *TlsFeature) Elem() stravaganza.Element {
	elem := stravaganza.NewBuilder("starttls").
		WithAttribute("xmlns", nsTLS)
	elem.WithChild(stravaganza.NewBuilder("required").Build())
	return elem.Build()
}

func (tf *TlsFeature) Mandatory() bool {
	return true
}

func (tf *TlsFeature) Match(elem stravaganza.Element) bool {
	return elem.Name() == "starttls" && elem.Attribute("xmlns") == nsTLS
}

func (tf *TlsFeature) Handle(_ stravaganza.Element, part Part) error {
	tf.handled = true
	cert, err := tls.LoadX509KeyPair(tf.certFile, tf.keyFile)
	if err != nil {
		part.Channel().SendElement(TlsFailureElem())
		part.Logger().Printf(Error, "create tls cert error: %s\n", err.Error())
		return err
	}
	msg := stravaganza.NewBuilder("proceed").WithAttribute("xmlns", nsTLS).Build()
	part.Channel().SendElement(msg)
	part.Conn().StartTLS(&tls.Config{Certificates: []tls.Certificate{cert}})
	return nil
}

func (tf *TlsFeature) Handled() bool {
	return tf.handled
}
