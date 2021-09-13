package xmppcore

import (
	"crypto/tls"
	"errors"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type TlsFeature struct {
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
	return &TlsFeature{certFile, keyFile}
}

func (tf *TlsFeature) notifyPeer(sender Sender) error {
	msg := stravaganza.NewBuilder("features").
		WithChild(tf.Elem()).Build()
	return sender.SendElement(msg)
}

func (tf *TlsFeature) Elem() stravaganza.Element {
	return stravaganza.NewBuilder("starttls").WithAttribute("xmlns", nsTLS).Build()
}

func (tf *TlsFeature) Resolve(part Part) error {
	if err := tf.notifyPeer(part.Channel()); err != nil {
		return err
	}
	var elem stravaganza.Element
	if e := part.Channel().NextElement(&elem); e != nil {
		if e == ErrNoElement {
			return ErrClientIgnoredTheFeature
		}
		return e
	}
	if elem.Name() != "starttls" {
		part.Channel().SendElement(TlsFailureElem())
		return errors.New("not a starttls error")
	}
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
