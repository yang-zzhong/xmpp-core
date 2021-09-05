package xmppcore

import (
	"fmt"

	"github.com/jackal-xmpp/stravaganza/v2"
)

type EchoMessageHandler struct{}

func (emh *EchoMessageHandler) Match(elem stravaganza.Element) bool {
	return elem.Name() == "message"
}

func (emh *EchoMessageHandler) Handle(elem stravaganza.Element, part Part) error {
	fmt.Print(part.Logger())
	part.Logger().Printf(Info, "grab a message %s\n", elem.GoString())
	msg := stravaganza.NewBuilder("message").
		WithAttribute("to", part.CommingStream().JID().String()).
		WithChildren(elem.AllChildren()...).Build()
	part.GoingStream().SendElement(msg)
	return nil
}
