package xmppcore

import (
	"github.com/jackal-xmpp/stravaganza/v2"
)

type clientSASLFeature struct {
	supports map[string]ToAuth
	IDAble
}

func ClientSASLFeature() clientSASLFeature {
	return clientSASLFeature{supports: make(map[string]ToAuth), IDAble: CreateIDAble()}
}

type ToAuth interface {
	ToAuth(mech string, part Part) error
}

func (csf *clientSASLFeature) Support(name string, auth ToAuth) {
	csf.supports[name] = auth
}

func (csf clientSASLFeature) Match(elem stravaganza.Element) bool {
	return elem.Name() == "mechanisms"
}

func (csf clientSASLFeature) Handle(elem stravaganza.Element, part Part) (catched bool, err error) {
	if !csf.Match(elem) {
		return false, nil
	}
	catched = true
	mechs := elem.AllChildren()
	for _, mech := range mechs {
		if auth, ok := csf.supports[mech.Text()]; ok {
			err = auth.ToAuth(mech.Text(), part)
			return
		}
	}
	err = SaslFailureError(SFInvalidMechanism, "")
	return
}
