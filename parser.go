package xmppcore

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"github.com/jackal-xmpp/stravaganza/v2"
)

const rootElementIndex = -1

const (
	streamName = "stream"
	openName   = "open"
)

// ParsingMode defines the way in which special parsed element
// should be considered or not according to the reader nature.
type ParsingMode int

const (
	// DefaultMode treats incoming elements as provided from raw byte reader.
	DefaultMode = ParsingMode(iota)

	// SocketStream treats incoming elements as provided from a socket transport.
	SocketStream
)

// ErrTooLargeStanza will be returned Parse when the size of the incoming stanza is too large.
var ErrTooLargeStanza = errors.New("parser: too large stanza")

var ErrUnexpectedOpenHeader = errors.New("parser: unexpected open header")

// ErrStreamClosedByPeer will be returned by Parse when stream closed element is parsed.
var ErrStreamClosedByPeer = errors.New("parser: stream closed by peer")

// ErrNoElement will be returned by Parse when no elements are available to be parsed in the reader buffer stream.
var ErrNoElement = errors.New("parser: no elements")

// Parser parses arbitrary XML input and builds an array with the structure of all tag and data elements.
type Parser struct {
	dec           *xml.Decoder
	mode          ParsingMode
	nextElement   stravaganza.Element
	stack         []*stravaganza.Builder
	pIndex        int
	inElement     bool
	lastOffset    int64
	maxStanzaSize int64
}

// New creates an empty Parser instance.
func NewParser(reader io.Reader, maxStanzaSize int) *Parser {
	return &Parser{
		mode:          SocketStream,
		dec:           xml.NewDecoder(reader),
		pIndex:        rootElementIndex,
		maxStanzaSize: int64(maxStanzaSize),
	}
}

func (p *Parser) NextElement() (stravaganza.Element, error) {
	i, err := p.Next()
	if err != nil {
		return nil, err
	}
	if _, ok := i.(xml.StartElement); ok {
		return nil, ErrUnexpectedOpenHeader
	}
	return i.(stravaganza.Element), nil
}

// Parse parses next available XML element from reader.
func (p *Parser) Next() (interface{}, error) {
	for {
		t, err := p.dec.Token()
		if err != nil {
			return nil, err
		}
		// check max stanza size limit
		off := p.dec.InputOffset()
		if p.maxStanzaSize > 0 && off-p.lastOffset > p.maxStanzaSize {
			return nil, ErrTooLargeStanza
		}
		switch t1 := t.(type) {
		case xml.StartElement:
			// got <stream>/<open>
			if p.mode == SocketStream && (t1.Name.Local == streamName || t1.Name.Local == openName) {
				p.lastOffset = p.dec.InputOffset()
				p.nextElement = nil
				return t1, nil
			}
			p.startElement(t1)
		case xml.CharData:
			if !p.inElement {
				return nil, ErrNoElement
			}
			p.setElementText(t1)

		case xml.EndElement:
			if p.mode == SocketStream && t1.Name.Local == streamName && t1.Name.Space == streamName {
				return nil, ErrStreamClosedByPeer
			}
			if err := p.endElement(t1); err != nil {
				return nil, err
			}
			if p.pIndex == rootElementIndex {
				goto done
			}
		}
	}

done:
	p.lastOffset = p.dec.InputOffset()
	elem := p.nextElement
	p.nextElement = nil

	return elem, nil
}

func (p *Parser) logToken(token xml.Token) {
	var buf bytes.Buffer
	en := xml.NewEncoder(&buf)
	en.EncodeToken(token)
	en.Flush()
	fmt.Printf("token: %s\n", buf.String())
}

func (p *Parser) startElement(t xml.StartElement) {
	name := t.Name.Local

	var attrs []stravaganza.Attribute
	for _, a := range t.Attr {
		name := xmlName(a.Name.Space, a.Name.Local)
		attrs = append(attrs, stravaganza.Attribute{Label: name, Value: a.Value})
	}
	builder := stravaganza.NewBuilder(name).WithAttributes(attrs...)
	p.stack = append(p.stack, builder)

	p.pIndex = len(p.stack) - 1
	p.inElement = true
}

func (p *Parser) setElementText(t xml.CharData) {
	p.stack[p.pIndex] = p.stack[p.pIndex].WithText(string(t))
}

func (p *Parser) endElement(t xml.EndElement) error {
	return p.closeElement(xmlName(t.Name.Space, t.Name.Local))
}

func (p *Parser) closeElement(name string) error {
	if p.pIndex == rootElementIndex {
		return errUnexpectedEnd(name)
	}
	builder := p.stack[p.pIndex]
	p.stack = p.stack[:p.pIndex]

	element := builder.Build()

	if name != element.Name() {
		return errUnexpectedEnd(name)
	}
	p.pIndex = len(p.stack) - 1
	if p.pIndex == rootElementIndex {
		p.nextElement = element
	} else {
		p.stack[p.pIndex] = p.stack[p.pIndex].WithChild(element)
	}
	p.inElement = false
	return nil
}

func xmlName(space, local string) string {
	return local
}

func errUnexpectedEnd(name string) error {
	return fmt.Errorf("xmppparser: unexpected end element </%s>", name)
}
