// Package htl parses a simplified html variant.  It can also serialize the
// parsed trees to string.
//
// Parses (a :href http://foo "body") to mean:
// Node{tag: "a", attr: {"href": "http://foo"}, kind: ElementNode,
//      content: {Node{tag: "body", kind: TextNode}}}
// It can then be turned to the string: <a href="foo">body</a>.
package htl

import (
	"fmt"
	"sort"
	"unicode"
)

const (
	maxStackDepth = 256 // Maximum depth of nodes in the html tree.

	openParenRune    = '('
	closeParenRune   = ')'
	quoteRune        = '"'
	escapingRune     = '\\'
	keywordStartRune = ':'
	commentStartRune = ';'
	newLineRune      = '\n'
)

// Set of tags that look like <img k1="v1" k2="v2"/> (i.e., no closing </img>).
/* const */ var degenerateTags = map[string]int{
	"br": 1, "hr": 1, "link": 1, "img": 1, "meta": 1,
}

/* const */ var htmlEscapeRuneMap = map[rune]string{
	'<': "&lt;", '>': "&gt;", '&': "&amp;", '\'': "&apos;", '"': "&quot;",
}

func htmlEscapeRune(r rune) string {
	rs, ok := htmlEscapeRuneMap[r]
	if ok {
		return rs
	}
	return string(r)
}

// This does not look correct.  It probably should not gobble up the backslash
// character in some cases.
func backslashUnescapeThenHtmlEscape(r rune) string {
	switch r {
	case 'f':
		return "\f"
	case 'n':
		return "\n"
	case 'r':
		return "\r"
	case 't':
		return "\t"
	case 'v':
		return "\v"
	default:
		return htmlEscapeRune(r) // backslash and doublequote are also covered here.
	}
}

type NodeType int

const (
	ElementNode NodeType = iota // An HTML Element node.
	TextNode                    // Only text (stored in node.tag).
)

type Node struct {
	kind    NodeType // If this is a text node, "tag" will contain the text.
	tag     string
	attr    map[string]string
	content []*Node
}

func NewNode(kind NodeType, tag string) *Node {
	return &Node{
		kind:    kind,
		tag:     tag,
		attr:    map[string]string{},
		content: []*Node{},
	}
}

type contextType int

const (
	contextDefault contextType = iota
	contextTag
	contextAfterTag
	contextAttrKey
	contextAfterAttrKey
	contextAttrValue
	contextContent
)

type ParseState struct {
	context           contextType
	token             string // we keep appending to it.
	key               string
	escapingBackslash bool
	stack             []*Node
}

// eatFn "eats" a rune, reads and possibly alters ParseState and returns the
// eatFn that should consume the next rune.
type eatFn func(r rune, ps *ParseState) eatFn

func (ps *ParseState) error(s string) eatFn {
	ps.token = s
	return nil
}

func (ps *ParseState) currentNode() *Node {
	if len(ps.stack) == 0 {
		return nil
	}
	return ps.stack[len(ps.stack)-1]
}

func (ps *ParseState) flushToken() (s string) {
	s, ps.token = ps.token, ""
	return s
}

func (ps *ParseState) commit() {
	switch ps.context {
	case contextTag:
		ps.currentNode().tag = ps.flushToken()

	case contextAttrKey:
		ps.key = ps.flushToken()

	case contextAttrValue:
		key := ""
		key, ps.key = ps.key, key
		ps.currentNode().attr[key] = ps.flushToken()

	case contextContent:
		node := ps.currentNode()
		node.content = append(node.content, NewNode(TextNode, ps.flushToken()))

	default: // noop
	}
}

func (ps *ParseState) push() eatFn {
	newNode := NewNode(ElementNode, "") // start with an empty tag.
	parent := ps.currentNode()
	if len(ps.stack) >= maxStackDepth {
		return ps.error("tree too deep")
	}
	ps.stack = append(ps.stack, newNode) // push into the stack.
	if parent != nil {
		parent.content = append(parent.content, newNode)
	}
	ps.context = contextTag
	return eatSymbol
}

func (ps *ParseState) pop() eatFn {
	if len(ps.stack) > 1 {
		ps.stack = ps.stack[0 : len(ps.stack)-1]
		ps.context = contextDefault
		return eatAir
	}
	return ps.error("unexpected closing paren")
}

// We have 3 eatFn: eatAir (consuming space between "tokens"), eatSymbol
// (consuming a symbol), eatString (consuming a string).
func eatAir(r rune, ps *ParseState) eatFn {
	switch {
	case r == openParenRune:
		if ps.context == contextAfterAttrKey {
			return ps.error("unexpected open paren")
		}
		return ps.push()

	case r == closeParenRune:
		if ps.context == contextAfterAttrKey {
			return ps.error("unexpected close paren")
		}
		return ps.pop()

	case r == quoteRune:
		if ps.context == contextAfterAttrKey {
			ps.context = contextAttrValue
			return eatString
		}
		ps.context = contextContent // or contextDefault?
		return eatString

	case r == commentStartRune:
		return eatComment

	case r == keywordStartRune:
		if ps.context == contextAfterTag {
			ps.context = contextAttrKey
			return eatSymbol
		}
		return ps.error("unexpect character")

	case r == escapingRune:
		return ps.error("backslash-escaping not allowed here")

	case unicode.IsSpace(r):
		return eatAir

	default:
		ps.token += string(r)
		if ps.context == contextAfterAttrKey {
			ps.context = contextAttrValue
		} else {
			ps.context = contextContent
		}
		return eatSymbol
	}
}

func eatSymbol(r rune, ps *ParseState) eatFn {
	switch {
	case r == openParenRune:
		if ps.context == contextAttrKey {
			return ps.error("unexpected open paren")
		}
		ps.commit()
		return ps.push()

	case r == closeParenRune:
		if ps.context == contextAttrKey {
			return ps.error("unexpected close paren")
		}
		ps.commit()
		return ps.pop()

	case r == quoteRune:
		ps.commit()
		if ps.context == contextAttrKey {
			ps.context = contextAttrValue
		} else {
			ps.context = contextContent
		}
		return eatString

	case r == escapingRune:
		return ps.error("backslash-escaping is not allowed here")

	case unicode.IsSpace(r):
		ps.commit()
		if ps.context == contextAttrKey {
			ps.context = contextAfterAttrKey
		} else {
			ps.context = contextAfterTag
		}
		return eatAir

	default:
		ps.token += string(r)
		return eatSymbol
	}
}

func eatString(r rune, ps *ParseState) eatFn {
	if ps.escapingBackslash {
		ps.escapingBackslash = false
		ps.token += backslashUnescapeThenHtmlEscape(r)
		return eatString
	}

	if r == quoteRune {
		ps.commit()
		if ps.context == contextAttrValue {
			ps.context = contextAfterTag
		} else {
			ps.context = contextDefault
		}
		return eatAir
	}

	if r == escapingRune {
		ps.escapingBackslash = true
		return eatString
	}
	ps.token += htmlEscapeRune(r)
	return eatString
}

func eatComment(r rune, ps *ParseState) eatFn {
	if r == newLineRune {
		return eatAir // Do not touch ps.context
	}
	return eatComment
}

func Parse(rawInput string) (*Node, error) {
	if rawInput == "" {
		return nil, nil
	}
	rootNode := NewNode(ElementNode, "")
	ps := ParseState{
		context:           contextDefault,
		token:             "",
		key:               "",
		escapingBackslash: false,
		stack:             append(make([]*Node, 0, maxStackDepth), rootNode),
	}
	eater := eatAir
	lineNumber := 1
	columnNumber := 0
	for _, r := range rawInput {
		eater = eater(r, &ps)
		if r == newLineRune {
			lineNumber++
			columnNumber = 0
		} else {
			columnNumber++
		}
		if eater == nil {
			return nil, fmt.Errorf(
				"Error processing rune %q (line %d column %d).  %s.",
				r, lineNumber, columnNumber, ps.token)
		}
	}
	if len(ps.stack) > 1 {
		return nil, fmt.Errorf(
			"Parser stack contains more than the root element.  "+
				"Perhaps %d closing parens are missing?", len(ps.stack)-1)
	}
	return ps.stack[0], nil // root node
}

// Sort a list of strings.
type stringSlice []string

func (a stringSlice) Len() int           { return len(a) }
func (a stringSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a stringSlice) Less(i, j int) bool { return a[i] < a[j] }

func (t *Node) String() string {
	if t == nil {
		return ""
	}

	if t.kind == TextNode {
		switch t.tag {
		case "_":
			return "&nbsp;"
		default:
			return t.tag
		}
	}

	if t.kind == ElementNode && t.tag == "" {
		s := ""
		for _, c := range t.content {
			s += c.String()
		}
		return s
	}

	if t.kind == ElementNode {
		s := "<" + t.tag
		attrKeys := []string{}
		for k, _ := range t.attr {
			attrKeys = append(attrKeys, k)
		}
		sort.Sort(stringSlice(attrKeys))
		for _, k := range attrKeys {
			s += " " + k + "=\"" + t.attr[k] + "\""
		}
		if len(t.content) == 0 {
			if _, isDegenerate := degenerateTags[t.tag]; isDegenerate {
				s += "/>"
			} else {
				s += "></" + t.tag + ">"
			}
		} else {
			s += ">"
			for _, c := range t.content {
				s += c.String()
			}
			s += "</" + t.tag + ">"
		}
		return s
	}

	return ""
}
