package xmppcore

import (
	"crypto/tls"
	"fmt"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type clientTlsFeature struct {
	conf *tls.Config
	IDAble
}

func ClientTlsFeature(conf *tls.Config) clientTlsFeature {
	return clientTlsFeature{conf: conf, IDAble: CreateIDAble()}
}

func (ctf clientTlsFeature) Match(elem stravaganza.Element) bool {
	return elem.Name() == "starttls"
}

func (ctf clientTlsFeature) Handle(elem stravaganza.Element, part Part) (catched bool, err error) {
	if !ctf.Match(elem) {
		return false, nil
	}
	catched = true
	elem = stravaganza.NewBuilder("starttls").WithAttribute("xmlns", NSTls).Build()
	if err = part.Channel().SendElement(elem); err != nil {
		return
	}
	if err = part.Channel().NextElement(&elem); err != nil {
		return
	}
	if elem.Name() != "proceed" {
		var f Failure
		if e := f.FromElem(elem, NSTls); e == nil {
			err = f
			return
		}
		err = fmt.Errorf("error on start tls, server returns: [%s]", elem.GoString())
		return
	}
	part.Conn().StartTLS(ctf.conf)
	return
}
