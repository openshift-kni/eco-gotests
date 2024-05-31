// Package dom implements a simple XML DOM that is a light wrapper on top of
// encoding/xml.  It is oriented towards processing XML used as an RPC
// encoding mechanism (XMLRPC, SOAP, etc.), and not for general XML document
// processing.  Specifically:
//
// 1. We ignore comments and document processing directives.  They are stripped
// out as part of document processing.
//
// 2. We do not have seperate Text fields.  Instead, each Element has a single
// Contents field which holds the contents of the last enclosed text in a tag.
//
package dom

import (
	"bytes"
)

// A Document represents an entire XML document.  Documents hold the root
// Element.
type Document struct {
	root *Element
}

// CreateDocument creates a new XML document.
func CreateDocument() *Document {
	return &Document{}
}

// Root returns the root element of the document.
func (doc *Document) Root() (node *Element) {
	return doc.root
}

// SetRoot sets the new root element of the document.
func (doc *Document) SetRoot(node *Element) {
	node.parent = nil
	doc.root = node
}

// Encode encodes the entire Document using the passed-in Encoder.
// The output is a well-formed XML document.
func (doc *Document) Encode(e *Encoder) (err error) {
	_, err = e.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	if err != nil {
		return err
	}
	if err = e.prettyEnd(); err != nil {
		return err
	}
	if doc.root != nil {
		return doc.root.Encode(e)
	}
	return nil
}

// Bytes encodes a Document into a byte array.  The document will be
// pretty-printed.
func (doc *Document) Bytes() []byte {
	b := bytes.Buffer{}
	encoder := NewEncoder(&b)
	encoder.Pretty()
	// since we are encoding to a bytes.Buffer, assume
	// Encode never fails.
	doc.Encode(encoder)
	encoder.Flush()
	return b.Bytes()
}

// Reader returns a bytes.Reader that can be used wherever
// something wants to consume this document via io.Reader
// This would be implemented using io.Pipe() for some nice
// streaming reads, but that does not play nice for some reason
// when using the returned Reader as an http.Request.Body
func (doc *Document) Reader() *bytes.Reader {
	return bytes.NewReader(doc.Bytes())
}

// String returns the result of stringifying the byte array that Bytes returns.
func (doc *Document) String() string {
	return string(doc.Bytes())
}
