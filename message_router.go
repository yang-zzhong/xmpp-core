package xmppcore

import "github.com/jackal-xmpp/stravaganza/v2"

type MessageRouter struct{}

func (msg *MessageRouter) Match(elem stravaganza.Element) bool {
	return elem.Name() == "message"
}

func (msg *MessageRouter) Handle(elem stravaganza.Element, part Part) error {
	to := elem.Attribute("to")
	var jid JID
	if err := ParseJID(to, &jid); err != nil {
		return err
	}
	if jid.Domain != part.CommingStream().JID().Domain {
		// route to other server
	}
	// route to other user on this server
	return nil
}
