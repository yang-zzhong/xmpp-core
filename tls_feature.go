package xmppcore

import (
	"crypto/tls"
	"errors"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type TlsFeature struct {
	resolved bool
	certFile string
	keyFile  string
}

func TlsFailureElem() stravaganza.Element {
	return stravaganza.NewBuilder("failure").WithAttribute("xmlns", nsTLS).Build()
}

const (
	nsTLS = "urn:ietf:params:xml:ns:xmpp-tls"
)

func NewTlsFeature(certFile, keyFile string) *TlsFeature {
	return &TlsFeature{false, certFile, keyFile}
}

func (tf *TlsFeature) notifyPeer(sender GoingStream) error {
	msg := stravaganza.NewBuilder("features").
		WithChild(stravaganza.NewBuilder("starttls").
			WithAttribute("xmlns", nsTLS).Build()).Build()
	return sender.SendElement(msg)
}

func (tf *TlsFeature) Reopen() bool {
	return true
}

func (tf *TlsFeature) Resolve(part Part) error {
	tf.notifyPeer(part.GoingStream())
	var elem stravaganza.Element
	if e := part.CommingStream().NextElement(&elem); e != nil {
		return e
	}
	if elem.Name() != "starttls" {
		return errors.New("not a starttls error")
	}
	msg := stravaganza.NewBuilder("proceed").WithAttribute("xmlns", nsTLS).Build()
	part.GoingStream().SendElement(msg)

	cert, err := tls.LoadX509KeyPair(tf.certFile, tf.keyFile)
	if err != nil {
		part.GoingStream().SendElement(TlsFailureElem())
		part.Logger().Printf(Error, "create tls cert error: %s\n", err.Error())
		return err
	}
	if err := part.Conn().StartTLS(&tls.Config{Certificates: []tls.Certificate{cert}}); err != nil {
		part.GoingStream().SendElement(TlsFailureElem())
		part.Logger().Printf(Error, "start tls error: %s\n", err.Error())
		return err
	}
	tf.resolved = true
	return nil
}

func (tf *TlsFeature) Resolved() bool {
	return tf.resolved
}
