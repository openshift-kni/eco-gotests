// Package soap wraps github.com/VictorLowther/simplexml/dom to provide
// convienent methods for dealing with SOAP messages as a SOAP client.
package soap

/*
Copyright 2015 Victor Lowther <victor.lowther@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"encoding/xml"
	"errors"
	"github.com/VictorLowther/simplexml/dom"
	"github.com/VictorLowther/simplexml/search"
	"io"
)

const ContentType = "application/soap+xml; charset=utf-8"

var (
	envName    xml.Name = xml.Name{Space: NS_ENVELOPE, Local: "Envelope"}
	headerName xml.Name = xml.Name{Space: NS_ENVELOPE, Local: "Header"}
	bodyName   xml.Name = xml.Name{Space: NS_ENVELOPE, Local: "Body"}
	faultName  xml.Name = xml.Name{Space: NS_ENVELOPE, Local: "Fault"}
)

// Message represents a SOAP message.
type Message struct {
	*dom.Document
	header, body *dom.Element
}

// NewMessage creates the skeleton of a new SOAP message.
func NewMessage() *Message {
	res := &Message{
		Document: dom.CreateDocument(),
		body:     dom.CreateElement(bodyName),
		header:   dom.CreateElement(headerName),
	}
	res.SetRoot(dom.CreateElement(envName))
	res.Root().AddChild(res.header)
	res.Root().AddChild(res.body)
	return res
}

// Headers returns the children of the Header element.
func (m *Message) Headers() []*dom.Element {
	return m.header.Children()
}

func (m *Message) AllHeaderElements() []*dom.Element {
	return m.header.Descendants()
}

// Body returns the children of the Body element.
func (m *Message) Body() []*dom.Element {
	return m.body.Children()
}

func (m *Message) AllBodyElements() []*dom.Element {
	return m.body.Descendants()
}

func get(loc, template *dom.Element) *dom.Element {
	match := search.Tag(template.Name.Local,template.Name.Space)
	for _,a := range template.Attributes {
		match = search.And(match,
			search.Attr(a.Name.Local,a.Name.Space, a.Value))
	}
	return search.First(match,loc.Children())
}

func set(loc *dom.Element, elems ...*dom.Element) {
	for _, elem := range elems {
		e := search.First(search.Tag(elem.Name.Local, elem.Name.Space),
			loc.Children())
		if e == nil {
			loc.AddChild(elem)
		} else {
			e.Replace(elem)
		}
	}
}

// GetHeader retrieves the first child of the SOAP header that
// matches the name and attributes on the passed element.
func (m *Message) GetHeader(template *dom.Element) *dom.Element {
	return get(m.header, template)
}

// SetHeader adds (or updates) any number of elements to the SOAP
// header.
//
// Elements that do not exist will be appended to the rest of the headers,
// and headers that already exist will be replaced in place.
// The SOAP message is returned.
func (m *Message) SetHeader(elems ...*dom.Element) *Message {
	set(m.header, elems...)
	return m
}

// RemoveHeader removes an element from the SOAP Header.
// If the element was not a header, nil is returned, otherwise the element
// is returned.
func (m *Message) RemoveHeader(elem *dom.Element) *dom.Element {
	return m.header.RemoveChild(elem)
}

// GetBody retrieves the first child of the SOAP body that
// matches the name and attributes on the passed element.
func (m *Message) GetBody(template *dom.Element) *dom.Element {
	return get(m.body, template)
}

// SetBody adds (or updates) any number of elements to the SOAP
// header.
//
// Elements that do not exist will be appended to the rest of the body,
// and ones that already exist will be replaced in place.
// The SOAP message is returned.
func (m *Message) SetBody(elems ...*dom.Element) *Message {
	set(m.body, elems...)
	return m
}

// RemoveBody removes an element from the SOAP Body.
// If the element was not a child of the SOAP body, nil is returned,
// otherwise the element is returned.
func (m *Message) RemoveBody(elem *dom.Element) *dom.Element {
	return m.body.RemoveChild(elem)
}

// Fault returns the Fault element if it is present in the SOAP body,
// otherwise it returns nil.
func (m *Message) Fault() *dom.Element {
	return m.GetBody(dom.Elem("Fault", NS_ENVELOPE))
}

// MustUnderstand ensures that the given element has the
// mustUnderstand attribute set.  WSMAN uses this to
// cause requests to fail if the endpoint does not know
// how to process a certian event.
func MustUnderstand(e *dom.Element) *dom.Element {
	return e.Attr("mustUnderstand", NS_ENVELOPE, "true")
}

// MuElem wraps a call to dom.Elem with a call to MustUnderstand.
// It is intended to be used as shorthand for generating headers.
func MuElem(name, space string) *dom.Element {
	return MustUnderstand(dom.Elem(name, space))
}

// MuElemC wraps a call to dom.ElemC with a call to MustUnderstand.
// It is intended to be used as shorthand for generating headers.
func MuElemC(name, space, content string) *dom.Element {
	return MustUnderstand(dom.ElemC(name, space, content))
}

// IsSoap takes a simplexml dom.Document and validates that
// it contains a valid SOAP message.  If it does, it returns a Message.
// If it does not, it returns an error explaining why not.
func IsSoap(doc *dom.Document) (res *Message, err error) {
	envelope := doc.Root()
	if envelope == nil {
		return nil, errors.New(NoEnvelope)
	}
	if envelope.Name != envName {
		return nil, errors.New(BadEnvelope)
	}
	children := envelope.Children()
	if len(children) > 2 {
		return nil, errors.New(EnvelopeOverstuffed)
	}
	var header, body *dom.Element
	for _, c := range children {
		if c.Name == headerName {
			if header != nil {
				return nil, errors.New(TooManyHeader)
			}
			header = c
		} else if c.Name == bodyName {
			if body != nil {
				return nil, errors.New(TooManyBody)
			}
			body = c
		} else {
			return nil, errors.New(BadTag)
		}
	}
	if header == nil {
		header = dom.CreateElement(headerName)
		doc.Root().AddChild(header)
	}
	if body == nil {
		body = dom.CreateElement(bodyName)
		doc.Root().AddChild(body)
	}
	return &Message{Document: doc, header: header, body: body}, nil
}

// Parse parses what is hopefully a well-formed SOAP message
// from the passed io.Reader.  If it is not, err will say why not.
func Parse(r io.Reader) (msg *Message, err error) {
	doc, err := dom.Parse(r)
	if err != nil {
		return nil, err
	}
	msg, err = IsSoap(doc)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
