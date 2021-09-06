package xmppcore

import (
	"errors"
	"github.com/google/uuid"

	"github.com/jackal-xmpp/stravaganza/v2"
)

var (
	ErrOnNextTokenTimeout      = errors.New("get next token timeout")
	ErrOnNextElementnTimeout   = errors.New("get next element timeout")
	ErrClientIgnoredTheFeature = errors.New("client ignored the feature")
)

type OptionalFeature interface {
	Elem() stravaganza.Element
	ElemHandler
}

type RequiredFeature interface {
	Resolve(Part) error
}

type ElemHandler interface {
	Match(stravaganza.Element) bool
	Handle(elem stravaganza.Element, part Part) error
}

type Part interface {
	ID() string
	CommingStream() CommingStream
	GoingStream() GoingStream
	Logger() Logger
	WithRequiredFeature(RequiredFeature)
	WithOptionalFeature(OptionalFeature)
	WithElemHandler(ElemHandler)
	Conn() Conn
	Run() error
	Stop()
}

type XPart struct {
	id               string
	commingStream    CommingStream
	goingStream      GoingStream
	requiredFeatures []RequiredFeature
	optionalFeatures []OptionalFeature
	elemHandlers     []ElemHandler
	logger           Logger
	conn             Conn
	quit             chan bool
}

func NewXPart(conn Conn, domain string, logger Logger) *XPart {
	return &XPart{
		id:               uuid.New().String(),
		commingStream:    NewXCommingStream(conn, domain),
		goingStream:      NewXGoingStream(conn),
		requiredFeatures: []RequiredFeature{},
		optionalFeatures: []OptionalFeature{},
		elemHandlers:     []ElemHandler{},
		logger:           logger,
		conn:             conn,
	}
}

func (part *XPart) ID() string {
	return part.id
}

func (part *XPart) Conn() Conn {
	return part.conn
}

func (part *XPart) CommingStream() CommingStream {
	return part.commingStream
}

func (part *XPart) GoingStream() GoingStream {
	return part.goingStream
}

func (part *XPart) WithRequiredFeature(f RequiredFeature) {
	part.requiredFeatures = append(part.requiredFeatures, f)
}

func (part *XPart) WithOptionalFeature(f OptionalFeature) {
	part.optionalFeatures = append(part.optionalFeatures, f)
}

func (part *XPart) WithElemHandler(mh ElemHandler) {
	part.elemHandlers = append(part.elemHandlers, mh)
}

func (part *XPart) Run() error {
	part.logger.Printf(Info, "part instance [%s] start running", part.id)
	if err := part.handleFeatures(); err != nil {
		return err
	}
	for {
		select {
		case <-part.quit:
			return nil
		default:
			var elem stravaganza.Element
			if err := part.CommingStream().NextElement(&elem); err != nil {
				part.logger.Printf(Error, "a error from part instance [%s] message handler: %s", part.id, err.Error())
				return err
			}
			for _, handler := range part.elemHandlers {
				if handler.Match(elem) {
					if err := handler.Handle(elem, part); err != nil {
						part.logger.Printf(Error, "a error occured from part instance [%s] message handler: %s", part.id, err.Error())
						continue
					}
				}
			}
		}
	}
}

func (part *XPart) handleFeatures() error {
	reopened := false
	for _, f := range part.requiredFeatures {
		if !reopened {
			if err := part.reopen(); err != nil {
				return err
			}
		}
		if err := f.Resolve(part); err != nil {
			if err == ErrClientIgnoredTheFeature {
				part.goingStream.Open(part.CommingStream())
				reopened = true
				continue
			}
			part.Logger().Printf(Error, "a error from part feature: %s", err.Error())
			return err
		}
		reopened = false
	}
	part.reopen()
	if err := part.handleUnrequiredFeature(); err != nil {
		part.Logger().Printf(Error, "a error from part unrequired features: %s", err.Error())
	}
	return nil
}

func (part *XPart) handleUnrequiredFeature() error {
	if len(part.optionalFeatures) == 0 {
		return part.goingStream.SendElement(stravaganza.NewBuilder("features").Build())
	}
	es := []stravaganza.Element{}
	for _, f := range part.optionalFeatures {
		es = append(es, f.Elem())
		part.WithElemHandler(f)
	}
	fs := stravaganza.NewBuilder("features").WithChildren(es...).Build()
	return part.goingStream.SendElement(fs)
}

func (part *XPart) reopen() error {
	if err := part.CommingStream().WaitHeader(nil); err != nil {
		part.logger.Printf(Error, "wait header error: %s", err.Error())
		return err
	}
	return part.GoingStream().Open(part.CommingStream())
}

func (part *XPart) Logger() Logger {
	return part.logger
}

func (part *XPart) Stop() {
	go func() {
		part.quit <- true
	}()
}
