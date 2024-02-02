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
	"fmt"
	"path"
	"strings"

	"github.com/VictorLowther/simplexml/dom"
	"github.com/VictorLowther/simplexml/search"
)

// Invoke creates a wsman.Message that will invoke method on resource.
// After creating the Message, you need to add the appropriate selectors
// with msg.Selectors(), and the appropriate parameters with msg.Parameters().
func (c *Client) Invoke(resource, method string) *Message {
	return c.NewMessage(resource + "/" + method).ResourceURI(resource)
}

func (m *Message) InvokeResponse() (*dom.Element, string, error) {
	action, err := m.GHC("Action")
	if err != nil {
		return nil, "", err
	}
	method, resource := path.Split(strings.TrimSuffix(action, "Response"))
	retbody := search.First(search.Tag(method+"_OUTPUT", resource), m.Body())
	if retbody == nil {
		return nil, "", fmt.Errorf("no %s_OUTPUT section in response", method)
	}
	returnValue := ""
	retval := search.First(search.Tag("ReturnValue", resource), retbody.Children())
	if retval == nil {
		return retbody, "", fmt.Errorf("no ReturnValue in %s_OUTPUT", method)
	}
	returnValue = string(retval.Content)
	return retbody, returnValue, nil
}

// Get creates a wsman.Message that will get an instance
// at the passed-in resource.
func (c *Client) Get(resource string) *Message {
	return c.NewMessage(Get).ResourceURI(resource)
}

func (m *Message) GetItem() (*dom.Element, error) {
	action, err := m.GHC("Action")
	if err != nil {
		return nil, err
	}
	if action != Get+"Response" {
		return nil, fmt.Errorf("not a GetResponse message")
	}
	if err != nil {
		return nil, err
	}
	b := m.Body()
	if len(b) == 0 {
		return nil, fmt.Errorf("no SOAP body elements")
	}
	return b[0], nil
}

// Put creates a wsman.Message that will update the passed-in
// resource.  The updated resource should be passed in as the
// only element in the Body of the messate.
func (c *Client) Put(resource string) *Message {
	return c.NewMessage(Put).ResourceURI(resource)
}

// Create creates a wsman.Message that will update the passed-in
// resource.  The updated resource should be passed in as the
// only element in the Body of the messate.
func (c *Client) Create(resource string) *Message {
	return c.NewMessage(Create).ResourceURI(resource)
}

// Delete creates a wsman.Message that will update the passed-in
// resource.  The updated resource should be passed in as the
// only element in the Body of the messate.
func (c *Client) Delete(resource string) *Message {
	return c.NewMessage(Delete).ResourceURI(resource)
}
