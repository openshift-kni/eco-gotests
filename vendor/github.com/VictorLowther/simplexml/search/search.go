// Package search contains searching routines for the simplexml/dom package.
// For some basic usage exmaples, see search_test.go
package search

import (
	"bytes"
	"regexp"

	"github.com/VictorLowther/simplexml/dom"
)

// Match is the basic type of a search function.
// It takes a single element, and returns a boolean
// indicating whether the element matched the func.
type Match func(*dom.Element) bool

// And takes any number of Match, and returns another
// Match that will match if all of passed Match functions
// match.
func And(funcs ...Match) Match {
	return func(e *dom.Element) bool {
		for _, fn := range funcs {
			if !fn(e) {
				return false
			}
		}
		return true
	}
}

// Or takes any number of Match, and returns another Match
// that will match if any of the passed Match functions match.
func Or(funcs ...Match) Match {
	return func(e *dom.Element) bool {
		for _, fn := range funcs {
			if fn(e) {
				return true
			}
		}
		return false
	}
}

// Not takes a single Match, and returns another Match
// that matches if fn does not match.
func Not(fn Match) Match {
	return func(e *dom.Element) bool {
		return !fn(e)
	}
}

// NoParent returns a matcher that matches iff the element
// does not have a parent
func NoParent() Match {
	return func(e *dom.Element) bool {
		return e.Parent() == nil
	}
}

// Ancestor returns a matcher that matches iff the element has an
// ancestor that matches the passed matcher
func Ancestor(fn Match) Match {
	return func(e *dom.Element) bool {
		return First(fn, e.Ancestors()) != nil
	}
}

// AncestorN returns a matcher that matches against the
// nth ancestor of the node being tested.
// If n == 0, then the node itself will be tested as a degenerate case.
// If there is no such ancestor the match fails.
func AncestorN(fn Match, distance uint) Match {
	return func(e *dom.Element) bool {
		if distance == 0 {
			return fn(e)
		}
		ancestors := e.Ancestors()
		if len(ancestors) < int(distance) {
			return false
		}
		return fn(ancestors[distance-1])
	}
}

// Parent returns a matcher that matches iff the element
// has a parent and that parent matches the passed fn.
func Parent(fn Match) Match {
	return func(e *dom.Element) bool {
		p := e.Parent()
		if p == nil {
			return false
		}
		return fn(p)
	}
}

// Child returns a matcher that matches iff the element has a
// child that matches the passed fn.
func Child(fn Match) Match {
	return func(e *dom.Element) bool {
		for _, c := range e.Children() {
			if fn(c) {
				return true
			}
		}
		return false
	}
}

// Always returns a matcher that always matches
func Always() Match {
	return func(e *dom.Element) bool {
		return true
	}
}

//Never returns a matcher that never matches
func Never() Match {
	return Not(Always())
}

// All returns all the nodes that fn matches
func All(fn Match, nodes []*dom.Element) []*dom.Element {
	res := make([]*dom.Element, 0, 0)
	for _, n := range nodes {
		if fn(n) {
			res = append(res, n)
		}
	}
	return res
}

// First returns the first element that fn matches
func First(fn Match, nodes []*dom.Element) *dom.Element {
	for _, n := range nodes {
		if fn(n) {
			return n
		}
	}
	return nil
}

// Tag is a helper function for matching against a specific tag.
// It takes a name and a namespace URL to match against.
// If either name or space are "*", then they will match
// any value.
// Return is a Match.
func Tag(name, space string) Match {
	return func(e *dom.Element) bool {
		return (space == "*" || space == e.Name.Space) &&
			(name == "*" || name == e.Name.Local)
	}
}

// FirstTag finds the first element in the set of nodes that matches the tag name and namespace.
func FirstTag(name, space string, nodes []*dom.Element) *dom.Element {
	return First(Tag(name, space), nodes)
}

// MustFirstTag is the same as FirstTag, but it panics if the tag cannot be found
func MustFirstTag(name, space string, nodes []*dom.Element) *dom.Element {
	res := FirstTag(name, space, nodes)
	if res == nil {
		panic("Failed to find tag " + name + " in namespace " + space)
	}
	return res
}

// TagRE is a helper function for matching against a specific tag
// using regular expressions.  It follows roughly the same rules as
// search.Tag
// Return is a Match
func TagRE(name, space *regexp.Regexp) Match {
	return func(e *dom.Element) bool {
		return (space == nil || space.MatchString(e.Name.Space)) &&
			(name == nil || name.MatchString(e.Name.Local))
	}
}

// Attr creates a Match against the attributes of an element.
// It follows the same rules as Tag
func Attr(name, space, value string) Match {
	return func(e *dom.Element) bool {
		for _, a := range e.Attributes {
			if (space == "*" || space == a.Name.Space) &&
				(name == "*" || name == a.Name.Local) &&
				(value == "*" || value == a.Value) {
				return true
			}
		}
		return false
	}
}

// AttrRE creates a Match against the attributes of an element.
// It follows the same rules as MatchRE
func AttrRE(name, space, value *regexp.Regexp) Match {
	return func(e *dom.Element) bool {
		for _, a := range e.Attributes {
			if (space == nil || space.MatchString(a.Name.Space)) &&
				(name == nil || name.MatchString(a.Name.Local)) &&
				(value == nil || value.MatchString(a.Value)) {
				return true
			}
		}
		return false
	}
}

// ContentExists creates a Match against an element that has non-empty
// Content.
func ContentExists() Match {
	return func(e *dom.Element) bool {
		return len(e.Content) > 0
	}
}

// Content creates a Match against an element that tests to see if
// it matches the supplied content.
func Content(content []byte) Match {
	return func(e *dom.Element) bool {
		return bytes.Equal(e.Content, content)
	}
}

// ContentRE creates a Match against the Content of am element
// that passes if the regex matches the content.
func ContentRE(regex *regexp.Regexp) Match {
	return func(e *dom.Element) bool {
		return regex.Match(e.Content)
	}
}
