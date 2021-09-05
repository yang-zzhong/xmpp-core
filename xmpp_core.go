package xmppcore

import (
	"errors"

	"github.com/jackal-xmpp/stravaganza/v2"
)

var (
	ErrOnNextTokenTimeout    = errors.New("get next token timeout")
	ErrOnNextElementnTimeout = errors.New("get next element timeout")
)

type Feature interface {
	Reopen() bool
	Resolve(Part) error
	Resolved() bool
}

type MessageHandler interface {
	Match(stravaganza.Element) bool
	Handle(elem stravaganza.Element, part Part) error
}

type Part interface {
	CommingStream() CommingStream
	GoingStream() GoingStream
	Logger() Logger
	WithFeature(Feature)
	WithMessageHandler(MessageHandler)
	Conn() Conn
	Run() error
}
