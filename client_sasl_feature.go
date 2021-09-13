package xmppcore

import (
	"github.com/jackal-xmpp/stravaganza/v2"
)

type ClientSASLFeature struct {
	supports map[string]ToAuth
}

func NewClientSASLFeature() *ClientSASLFeature {
	return &ClientSASLFeature{supports: make(map[string]ToAuth)}
}

type ToAuth interface {
	ToAuth(mech string, part Part) error
}

func (csf *ClientSASLFeature) Support(name string, auth ToAuth) {
	csf.supports[name] = auth
}

func (csf *ClientSASLFeature) Match(elem stravaganza.Element) bool {
	return elem.Name() == "mechanisms"
}

func (csf *ClientSASLFeature) Handle(elem stravaganza.Element, part Part) error {
	mechs := elem.AllChildren()
	for _, mech := range mechs {
		if auth, ok := csf.supports[mech.Text()]; ok {
			if err := auth.ToAuth(mech.Text(), part); err != nil {
				return err
			}
			return nil
		}
	}
	return SaslFailureError(SFInvalidMechanism, "")
}
