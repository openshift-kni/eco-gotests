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
	"fmt"

	"github.com/VictorLowther/simplexml/dom"
	"github.com/VictorLowther/simplexml/search"
)

func (c *Client) enumRelease(ctx context.Context, elem *dom.Element) {
	req := c.NewMessage(Release)
	body := dom.Elem("Release", NSWSMEN)
	_ = req.SetBody(body)
	_ = body.AddChild(elem)
	_, _ = req.Send(ctx)
}

func enumHelper(ctx context.Context, firstreq, resp *Message) error {
	searchContext := search.Tag("EnumerationContext", NSWSMEN)
	searchEnd := search.Tag("EndOfSequence", NSWSMAN)
	if search.First(searchEnd, resp.AllBodyElements()) != nil {
		return nil
	}
	first := search.First(searchContext, resp.AllBodyElements())
	items := search.First(search.Tag("Items", "*"), resp.AllBodyElements())
	resource := firstreq.GetHeader(dom.Elem("ResourceURI", NSWSMAN))
	maxElem := search.First(search.Tag("MaxElements", NSWSMAN), firstreq.AllBodyElements())
	enumEpr := search.First(search.Tag("EnumerationMode", NSWSMAN), firstreq.AllBodyElements())
	if resource == nil {
		return fmt.Errorf("WSMAN Enumerate request did not have RequestURI")
	}
	if items == nil {
		enumResp := search.First(search.Tag("EnumerateResponse", "*"), resp.AllBodyElements())
		if enumResp == nil {
			return fmt.Errorf("enumeration response did not have EnumerateResponse body element")
		}
		items = dom.Elem("Items", NSWSMAN)
		enumResp.AddChild(items)
	}

	for first != nil {
		req := resp.client.NewMessage(Pull)
		req.SetHeader(resource)
		body := dom.Elem("Pull", NSWSMEN)
		req.SetBody(body)
		body.AddChild(first)
		if maxElem != nil {
			body.AddChild(maxElem)
		}
		if enumEpr != nil {
			body.AddChild(enumEpr)
		}
		nextResp, err := req.Send(ctx)
		if err != nil {
			resp.client.enumRelease(ctx, first)
			return err
		}
		first = search.First(searchContext, nextResp.AllBodyElements())
		extraItems := search.First(search.Tag("Items", "*"), nextResp.AllBodyElements())
		if extraItems != nil {
			items.AddChildren(extraItems.Children()...)
		}
		if search.First(searchEnd, nextResp.AllBodyElements()) != nil {
			break
		}
	}
	return nil
}

func (c *Client) doEnumerate(resource string, epr, optimize bool) *Message {
	req := c.NewMessage(Enumerate).ResourceURI(resource)
	body := dom.Elem("Enumerate", NSWSMEN)
	req.SetBody(body)
	if optimize {
		optimizeEnum := dom.Elem("OptimizeEnumeration", NSWSMAN)
		maxElem := dom.ElemC("MaxElements", NSWSMAN, "100")
		maxElem.Content = []byte("100")
		body.AddChildren(optimizeEnum, maxElem)
	}
	if epr {
		enumEpr := dom.Elem("EnumerationMode", NSWSMAN)
		enumEpr.Content = []byte("EnumerateEPR")
		body.AddChild(enumEpr)
	}
	req.replyHelper = enumHelper
	return req
}

// Enumerate creates a wsman.Message that will enumerate all the objects
// available at resource.  If there are many objects, it will arrange
// for the appropriate series of wsman Pull calls to be performed, so you can
// be certain that the response to this message has all the objects you specify.
func (c *Client) Enumerate(resource string) *Message {
	return c.doEnumerate(resource, false, c.OptimizeEnum)
}

// EnumerateEPR creates a message that will enumerate the endpoints for a given resource.
func (c *Client) EnumerateEPR(resource string) *Message {
	return c.doEnumerate(resource, true, c.OptimizeEnum)
}

func (m *Message) EnumItems() ([]*dom.Element, error) {
	action, err := m.GHC("Action")
	if err != nil || action != Enumerate+"Response" {
		return nil, fmt.Errorf("not an EnumerateResponse message")
	}
	items := search.First(search.Tag("Items", NSWSMAN), m.AllBodyElements())
	if items == nil {
		return nil, fmt.Errorf("no items returned from EnumItems")
	}
	return items.Children(), nil
}
