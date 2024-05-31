package dom

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/url"
)

// Encoder holds the state needed to encode the DOM into
// a well-formed XML document.
type Encoder struct {
	*bufio.Writer
	depth           int
	pretty          bool
	started         bool
	namespacesAdded int
	nsPrefixMap     map[string]string
	nsURLMap        map[string]string
}

// NewEncoder returns a new Encoder that will output to the
// passed-in io.Writer.
//
// The encoded docuemnt will have all namespace declarations lifted to the
// root element of the document.
func NewEncoder(writer io.Writer) *Encoder {
	res := &Encoder{Writer: bufio.NewWriter(writer)}
	res.nsPrefixMap = make(map[string]string)
	res.nsURLMap = make(map[string]string)
	return res
}

// Pretty puts the passed Encoder into pretty-print mode.
func (e *Encoder) Pretty() {
	if e.started {
		log.Panic("xml: Encoding has started, cannot set Pretty flag")
	}
	e.pretty = true
}

func (e *Encoder) addNamespace(ns string, prefix string) {
	if e.started {
		log.Panic("Cannot add element namespaces after encoding starts!")
	}
	if ns == "" || ns == "xmlns" {
		return
	}
	if prefix != "" {
		if _, found := e.nsURLMap[ns]; found {
			delete(e.nsPrefixMap, e.nsURLMap[ns])
			delete(e.nsURLMap, prefix)
		}
		e.nsPrefixMap[prefix] = ns
		e.nsURLMap[ns] = prefix
		return
	}

	if _, found := e.nsURLMap[ns]; found {
		return
	}
	if _, err := url.Parse(ns); err != nil {
		log.Panic(err)
	}
	prefix = fmt.Sprintf("ns%v", e.namespacesAdded)
	e.namespacesAdded++
	e.nsPrefixMap[prefix] = ns
	e.nsURLMap[ns] = prefix
}

func (e *Encoder) prettyEnd() error {
	if !e.pretty {
		return nil
	}
	_, err := e.WriteString("\n")
	return err
}

func (e *Encoder) spaces() error {
	if e.pretty {
		for i := 0; i < e.depth; i++ {
			if _, err := e.WriteString(" "); err != nil {
				return err
			}
		}
	}
	return nil
}
