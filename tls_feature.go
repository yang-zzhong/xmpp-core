package xmppcore

import (
	"crypto/tls"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type tlsFeature struct {
	certFile  string
	keyFile   string
	mandatory bool

	handled bool
	IDAble
}

func TlsFailureElem() stravaganza.Element {
	return stravaganza.NewBuilder("failure").WithAttribute("xmlns", NSTls).Build()
}

const (
	NSTls = "urn:ietf:params:xml:ns:xmpp-tls"
)

func TlsFeature(certFile, keyFile string, mandatory bool) tlsFeature {
	return tlsFeature{
		certFile:  certFile,
		keyFile:   keyFile,
		mandatory: mandatory,
		handled:   false,
		IDAble:    CreateIDAble()}
}

func (tf tlsFeature) Elem() stravaganza.Element {
	elem := stravaganza.NewBuilder("starttls").
		WithAttribute("xmlns", NSTls)
	elem.WithChild(stravaganza.NewBuilder("required").Build())
	return elem.Build()
}

func (tf tlsFeature) Mandatory() bool {
	return true
}

func (tf tlsFeature) Match(elem stravaganza.Element) bool {
	return elem.Name() == "starttls" && elem.Attribute("xmlns") == NSTls
}

func (tf *tlsFeature) Handle(elem stravaganza.Element, part Part) (catched bool, err error) {
	if !tf.Match(elem) {
		return false, nil
	}
	tf.handled = true
	cert, err := tls.LoadX509KeyPair(tf.certFile, tf.keyFile)
	if err != nil {
		part.Channel().SendElement(TlsFailureElem())
		part.Logger().Printf(LogError, "create tls cert error: %s\n", err.Error())
		return
	}
	msg := stravaganza.NewBuilder("proceed").WithAttribute("xmlns", NSTls).Build()
	part.Channel().SendElement(msg)
	part.Conn().StartTLS(&tls.Config{Certificates: []tls.Certificate{cert}})
	return
}

func (tf tlsFeature) Handled() bool {
	return tf.handled
}
