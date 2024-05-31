package dom

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
)

// Element represents a node in an XML document.
// Elements are arranged in a tree which corresponds to
// the structure of the XML documents.
type Element struct {
	Name     xml.Name
	children []*Element
	parent   *Element
	// Unlike a full-fledged XML DOM, we only have a single Content field
	// instead of representing Text nodes seperately.
	Content    []byte
	Attributes []xml.Attr
}

// CreateElement creates a new element with the passed-in xml.Name.
// The created Element has no parent, no children, no content, and no
// attributes.
func CreateElement(n xml.Name) *Element {
	return &Element{
		Name:       n,
		children:   []*Element{},
		Attributes: []xml.Attr{},
	}
}

// Attr creates a new xml.Attr.  It is exactly equivalent to creating a new
// xml.Attr with:
//    xml.Attr{
//        Name: xml.Name{
//            Local: name,
//            Space: space,
//        },
//        Value: value,
//    }
func Attr(name, space, value string) xml.Attr {
	return xml.Attr{
		Name:  xml.Name{Space: space, Local: name},
		Value: value,
	}
}

// Elem creates a new Element.  It is equivalent to creating a new
// Element with:
//    CreateElement(xml.Name{Local: name, Space: space})
func Elem(name, space string) *Element {
	return CreateElement(xml.Name{Space: space, Local: name})
}

// ElemC creates a new Element with Content.  It is equivalent to
// creating a new Element with:
//    e := Elem(name,space)
//    e.Content = []byte(content)
func ElemC(name, space, content string) *Element {
	res := Elem(name, space)
	res.Content = []byte(content)
	return res
}

// AddChild adds child to node.
// child will be reparented if needed.
// The return value is node.
func (node *Element) AddChild(child *Element) *Element {
	if child.parent != nil {
		child.parent.RemoveChild(child)
	}
	child.parent = node
	node.children = append(node.children, child)
	return node
}

// GetAttr returns all the matching Attrs on the node.
func (node *Element) GetAttr(name, space, val string) []xml.Attr {
	res := []xml.Attr{}
	for i := range node.Attributes {
		if (name == "*" || name == node.Attributes[i].Name.Local) &&
			(space == "*" || space == node.Attributes[i].Name.Space) &&
			(val == "*" || val == node.Attributes[i].Value) {
			res = append(res, node.Attributes[i])
		}
	}
	return res
}

// AddChildren adds children to node.
// The children will be reparented as needed.
// The return value is node.
func (node *Element) AddChildren(children ...*Element) *Element {
	for _, c := range children {
		node.AddChild(c)
	}
	return node
}

// Replace performs an in-place replacement of node with other.
// other should not be used after this functions returns.
// node will be returned.
func (node *Element) Replace(other *Element) *Element {
	node.Name = other.Name
	node.Content = other.Content
	node.Attributes = other.Attributes
	node.children = []*Element{}
	node.AddChildren(other.children...)
	return node
}

// RemoveChild removes child from node.  The removed child
// will be returned if it was actually a child of node, otherwise
// nil will be returned.
func (node *Element) RemoveChild(child *Element) *Element {
	p := -1
	for i, v := range node.children {
		if v == child {
			p = i
			break
		}
	}

	if p == -1 {
		return nil
	}

	copy(node.children[p:], node.children[p+1:])
	node.children = node.children[0 : len(node.children)-1]
	child.parent = nil
	return child
}

// Children returns all the children of node.
func (node *Element) Children() (res []*Element) {
	res = make([]*Element, 0, len(node.children))
	return append(res, node.children...)
}

// Descendants returns all descendants of node in breadth order.
func (node *Element) Descendants() (res []*Element) {
	res = make([]*Element, 0, len(node.children))
	toProcess := node.Children()
	for len(toProcess) > 0 {
		nextToProcess := make([]*Element, 0, len(node.children))
		for _, n := range toProcess {
			nextToProcess = append(nextToProcess, n.Children()...)
		}
		res = append(res, toProcess...)
		toProcess = nextToProcess
	}
	return res
}

// All returns node + node.Descendants()
func (node *Element) All() []*Element {
	return append([]*Element{node}, node.Descendants()...)
}

// Parent returns the parent of this node. If there is no parent, returns nil.
func (node *Element) Parent() *Element {
	return node.parent
}

// Ancestors returns all the ancestors of this node with the most distant
// ancestor last.
func (node *Element) Ancestors() (res []*Element) {
	res = make([]*Element, 0, 1)
	t := node.parent
	for t != nil {
		res = append(res, t)
		t = t.parent
	}
	return res
}

// AddAttr adds attr to node.
// Duplicates are ignored. If attr has the same
// name as a preexisting attribute, then it will replace
// the preexsting attribute.
// Return is node.
func (node *Element) AddAttr(attr xml.Attr) *Element {
	for _, a := range node.Attributes {
		if a == attr {
			return node
		}
		if a.Name == attr.Name {
			a.Value = attr.Value
			return node
		}
	}
	node.Attributes = append(node.Attributes, attr)
	return node
}

// Attr creates a new xml.Attr and adds it to node.  It is equivalent to:
//    node.AddAttr(xml.Attr{
//        Name: xml.Name{
//            Space: space,
//            Local: name,
//        },
//        Value: value,
//    })
func (node *Element) Attr(name, space, value string) *Element {
	return node.AddAttr(Attr(name, space, value))
}

// SetParent makes parent the new parent of node, and returns node.
func (node *Element) SetParent(parent *Element) *Element {
	parent.AddChild(node)
	return node
}

func (node *Element) addNamespaces(encoder *Encoder) {
	// See if any of our attribs are in the xmlns namespace.
	// If they are, try to add them with their prefix
	for _, a := range node.Attributes {
		if a.Name.Space == "xmlns" {
			encoder.addNamespace(a.Value, a.Name.Local)
		}
	}

	encoder.addNamespace(node.Name.Space, "")
	for _, a := range node.Attributes {
		encoder.addNamespace(a.Name.Space, "")
	}
	for _, c := range node.children {
		c.addNamespaces(encoder)
	}
}

func namespacedName(e *Encoder, name xml.Name) string {
	if name.Space == "" {
		return name.Local
	}
	if name.Space == "xmlns" {
		return name.Space + ":" + name.Local
	}
	prefix, found := e.nsURLMap[name.Space]
	if !found {
		log.Panicf("No prefix found in %v for namespace %s", e.nsURLMap, name.Space)
	}
	return prefix + ":" + name.Local
}

// Encode encodes an element using the passed-in Encoder. If an error occurs
// during encoding, that error is returned.
func (node *Element) Encode(e *Encoder) (err error) {
	// This could use some refactoring. but it works Well Enough(tm)
	writeNamespaces := !e.started
	if writeNamespaces {
		node.addNamespaces(e)
		e.started = true
	}
	err = e.spaces()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(e, "<%s", namespacedName(e, node.Name))
	if err != nil {
		return err
	}
	for _, a := range node.Attributes {
		if a.Name.Space == "xmlns" {
			continue
		}
		_, err = fmt.Fprintf(e, " %s=\"%s\"", namespacedName(e, a.Name), a.Value)
		if err != nil {
			return err
		}
	}
	if writeNamespaces {
		for prefix, uri := range e.nsPrefixMap {
			_, err = fmt.Fprintf(e, " xmlns:%s=\"%s\"", prefix, uri)
			if err != nil {
				return err
			}
		}
	}
	if len(node.children) == 0 && len(node.Content) == 0 {
		ctag := "/>"
		if e.pretty {
			ctag = "/>\n"
		}
		_, err = e.WriteString(ctag)
		if err != nil {
			return err
		}
		return
	}
	_, err = e.WriteString(">")
	if len(node.Content) > 0 {
		xml.EscapeText(e, node.Content)
	}
	if len(node.children) > 0 {
		e.depth++
		if err = e.prettyEnd(); err != nil {
			return err
		}
		for _, c := range node.children {
			if err = c.Encode(e); err != nil {
				return err
			}
		}
		e.depth--
		if err = e.spaces(); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(e, "</%s>", namespacedName(e, node.Name))
	if err != nil {
		return err
	}
	return e.prettyEnd()
}

// Bytes returns a pretty-printed XML encoding of this part of the tree.
// The return is a byte array.
func (node *Element) Bytes() []byte {
	var b bytes.Buffer
	encoder := NewEncoder(&b)
	encoder.Pretty()
	node.Encode(encoder)
	encoder.Flush()
	return b.Bytes()
}

// String returns a pretty-printed XML encoding of this part of the tree.
//  The return is a string.
func (node *Element) String() string {
	return string(node.Bytes())
}
