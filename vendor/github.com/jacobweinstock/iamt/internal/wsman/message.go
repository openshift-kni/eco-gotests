package wsman

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
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/VictorLowther/simplexml/dom"
	"github.com/VictorLowther/simplexml/search"
	"github.com/VictorLowther/soap"
	uuid "github.com/satori/go.uuid"
)

// Message represents WSMAN messages.
type Message struct {
	*soap.Message
	client *Client
	// First arg is the request, second is the initial response.
	// For now, this is used to allow Enumerate to Pull additional
	// replys without having to make API users do it.
	replyHelper func(context.Context, *Message, *Message) error
}

// Resource turns a resource URI into an appropriate DOM element
// for inclusion in the SOAP header.
func Resource(uri string) *dom.Element {
	return soap.MuElemC("ResourceURI", NSWSMAN, uri)
}

// NewMessage creates a new wsman.Message that can be sent
// via c.  It populates the message with the passed action
// and some other necessary headers.
func (c *Client) NewMessage(action string) (msg *Message) {
	msg = &Message{
		Message: soap.NewMessage(),
		client:  c,
	}
	msg.SetHeader(
		soap.MuElemC("Action", NSWSA, action),
		soap.MuElemC("To", NSWSA, c.target),
		soap.MuElemC("MessageID", NSWSA, fmt.Sprintf("uuid:%s", uuid.NewV4())),
		dom.Elem("ReplyTo", NSWSA).AddChild(
			soap.MuElemC("Address", NSWSA,
				"http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous")))
	return msg
}

// Options are used to modify how certain WSMAN operations work.
// For now, the only thing we use it for is to make EnumerateEPR work.
// See http://www.dmtf.org/sites/default/files/standards/documents/DSP0226_1.2.0.pdf,
// section 6.4 for more information.
// Options takes an even number of strings, which should be key/value pairs.
// They will be added to the soap Header as Options in an OptionSet.
//
// Example:
//
// client.Enumerate("http://my.resource/url") -> msg:
//    <?xml version="1.0" encoding="UTF-8"?>
//    <ns0:Envelope xmlns:ns0="http://www.w3.org/2003/05/soap-envelope" xmlns:ns1="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:ns2="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:ns3="http://schemas.xmlsoap.org/ws/2004/09/enumeration">
//     <ns0:Header>
//     <ns1:Action ns0:mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/enumeration/Enumerate</ns1:Action>
//      <ns1:To ns0:mustUnderstand="true">https://192.168.128.41:443/wsman</ns1:To>
//      <ns1:MessageID ns0:mustUnderstand="true">uuid:0590dc5f-369b-4ed3-9da4-8461a6f66dae</ns1:MessageID>
//      <ns1:ReplyTo>
//       <ns1:Address ns0:mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</ns1:Address>
//      </ns1:ReplyTo>
//      <ns2:ResourceURI ns0:mustUnderstand="true">http://my.resource/url</ns2:ResourceURI>
//     </ns0:Header>
//     <ns0:Body>
//      <ns3:Enumerate>
//       <ns2:OptimizeEnumeration/>
//       <ns2:MaxElements>100</ns2:MaxElements>
//      </ns3:Enumerate>
//     </ns0:Body>
//    </ns0:Envelope>
// msg.Options("EnumerationMode", "EnumerateEPR") -> msg:
//    <?xml version="1.0" encoding="UTF-8"?>
//    <ns0:Envelope xmlns:ns0="http://www.w3.org/2003/05/soap-envelope" xmlns:ns1="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:ns2="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:ns3="http://schemas.xmlsoap.org/ws/2004/09/enumeration">
//     <ns0:Header>
//      <ns1:Action ns0:mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/enumeration/Enumerate</ns1:Action>
//      <ns1:To ns0:mustUnderstand="true">https://192.168.128.41:443/wsman</ns1:To>
//      <ns1:MessageID ns0:mustUnderstand="true">uuid:0590dc5f-369b-4ed3-9da4-8461a6f66dae</ns1:MessageID>
//      <ns1:ReplyTo>
//       <ns1:Address ns0:mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</ns1:Address>
//      </ns1:ReplyTo>
//      <ns2:ResourceURI ns0:mustUnderstand="true">http://my.resource/url</ns2:ResourceURI>
//      <ns2:OptionSet>
//       <ns2:Option Name="EnumerationMode">EnumerateEPR</ns2:Option>
//      </ns2:OptionSet>
//     </ns0:Header>
//     <ns0:Body>
//      <ns3:Enumerate>
//       <ns2:OptimizeEnumeration/>
//       <ns2:MaxElements>100</ns2:MaxElements>
//      </ns3:Enumerate>
//     </ns0:Body>
//    </ns0:Envelope>
//
// Why not use a map?  Because that would make it much more painful to handle
// arrays, and I am not that interested in adding evem more magic.

// MakeOption makes an element that can be added as an option to the
// message header with AddOption.
func (m *Message) MakeOption(name string) *dom.Element {
	return dom.Elem("Option", NSWSMAN).Attr("Name", "", name)
}

// AddOption adds any number of elements (created with MakeOption) to
// the message header.
func (m *Message) AddOption(options ...*dom.Element) *Message {
	optset := dom.Elem("OptionSet", NSWSMAN)
	if found := m.GetHeader(optset); found != nil {
		optset = found
	} else {
		m.SetHeader(optset)
	}
	optset.AddChildren(options...)
	return m
}

// Options is a fast way of creating options that are basically
// key/value pairs.
func (m *Message) Options(opts ...string) *Message {
	if len(opts)%2 != 0 {
		panic("message.Options passed an odd number of args!")
	}
	options := make([]*dom.Element, len(opts)/2)
	for i := 0; i < len(opts); i += 2 {
		options[i/2] = m.MakeOption(opts[i])
		options[i/2].Content = []byte(opts[i+1])
	}
	m.AddOption(options...)
	return m
}

// GHC gets the contents of a specific Header field.
func (m *Message) GHC(field string) (string, error) {
	h := m.GetHeader(dom.Elem(field, NSWSA))
	if h == nil {
		return "", fmt.Errorf("Message has no header %s", field)
	}
	return string(h.Content), nil
}

// ResourceURI sets the ResourceURI header of the message.
func (m *Message) ResourceURI(resource string) *Message {
	m.SetHeader(Resource(resource))
	return m
}

// MakeSelector makes an element that can be added to the message with
// AddSelector.
func (m *Message) MakeSelector(name string) *dom.Element {
	return dom.Elem("Selector", NSWSMAN).Attr("Name", "", name)
}

// AddSelector adds any number of elements (created with MakeSelector)
// to the message.
func (m *Message) AddSelector(selector ...*dom.Element) *Message {
	selset := dom.Elem("SelectorSet", NSWSMAN)
	if found := m.GetHeader(selset); found != nil {
		selset = found
	} else {
		m.SetHeader(selset)
	}
	selset.AddChildren(selector...)
	return m
}

// Selectors are used to target the resource that Get, Put, and Invoke
// should work with.  They work like Options does, except they add a SelectorSet
// element with Selectors instead of Options.
func (m *Message) Selectors(args ...string) *Message {
	if len(args)%2 != 0 {
		panic("message.Selectors passed an odd number of args!")
	}
	selectors := make([]*dom.Element, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		selectors[i/2] = m.MakeSelector(args[i])
		selectors[i/2].Content = []byte(args[i+1])
	}
	m.AddSelector(selectors...)
	return m
}

func (m *Message) paramNamespace() (psetNamespace, psetName string) {
	resourceNS, err := m.GHC("Action")
	if err != nil {
		panic(err.Error())
	}
	idx := strings.LastIndex(resourceNS, "/")
	if idx == -1 {
		panic("Action is malformed!")
	}
	resourceName := fmt.Sprintf("%s_INPUT", resourceNS[idx+1:])
	resourceNS = resourceNS[:idx]

	return resourceNS, resourceName
}

// MakeParameter makes an element which can be added to the message
// with AddParameter.
func (m *Message) MakeParameter(name string) *dom.Element {
	resourceNS, _ := m.paramNamespace()
	return dom.Elem(name, resourceNS)
}

// AddParameter adds any number of elements (created with
// MakeParameter) to the message.
func (m *Message) AddParameter(parameters ...*dom.Element) *Message {
	resourceNS, resourceName := m.paramNamespace()
	paramSet := dom.Elem(resourceName, resourceNS)
	if found := m.GetBody(paramSet); found != nil {
		paramSet = found
	} else {
		m.SetBody(paramSet)
	}
	paramSet.AddChildren(parameters...)
	return m
}

// Parameters sets the parameters for an invoke call.
// It takes an even number of strings, which should be key:value pairs.
// It works alot like Options, except it adds the parameters to the Body
// in the format that WSMAN expects parameter elements to be in.
func (m *Message) Parameters(args ...string) *Message {
	if len(args)%2 != 0 {
		panic("message.Selectors passed an odd number of args!")
	}
	params := make([]*dom.Element, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		params[i/2] = m.MakeParameter(args[i])
		params[i/2].Content = []byte(args[i+1])
	}
	m.AddParameter(params...)
	return m
}

func (m *Message) GetResource() string {
	hdr := search.First(search.Tag("ResourceURI", NSWSMAN), m.AllHeaderElements())
	if hdr == nil {
		panic("No ResourceURI")
	}
	return string(hdr.Content)
}

// MakeValue makes an element that can be used to set a new instance value for a Put call.
func (m *Message) MakeValue(name string) *dom.Element {
	ns := m.GetResource()
	return dom.Elem(name, ns)
}

func (m *Message) AddValue(values ...*dom.Element) *Message {
	ns := m.GetResource()
	idx := strings.LastIndex(ns, `/`)
	if idx == -1 {
		panic("Resource is malformed")
	}
	valueSet := dom.Elem(ns[idx+1:], ns)
	if found := m.GetBody(valueSet); found != nil {
		valueSet = found
	} else {
		m.SetBody(valueSet)
	}
	valueSet.AddChildren(values...)
	return m
}

func (m *Message) Values(args ...string) *Message {
	if len(args)%2 != 0 {
		panic("message.Values passed an odd number of args")
	}
	values := make([]*dom.Element, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		values[i/2] = m.MakeValue(args[i])
		values[i/2].Content = []byte(args[i+1])
	}
	m.AddValue(values...)
	return m
}

// Send sends a message to the endpoint of the Client it was
// constructed with, and returns either the Message that was
// returned, or an error stating what went wrong.
func (m *Message) Send(ctx context.Context) (*Message, error) {
	res, err := m.client.Post(ctx, m.Message)
	if err != nil {
		return nil, err
	}
	msg := &Message{Message: res, client: m.client}
	if m.replyHelper != nil {
		// TODO: figure out how to handle this error
		_ = m.replyHelper(ctx, m, msg)
	}
	if msg.Fault() != nil {
		return msg, errors.New("SOAP Fault")
	}
	return msg, nil
}
