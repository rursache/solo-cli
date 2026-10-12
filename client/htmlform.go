package client

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// htmlDoc is a parsed server-rendered page with just enough helpers to drive
// the cont.solo.ro login forms
type htmlDoc struct {
	root  *html.Node
	forms []*htmlForm
}

type htmlForm struct {
	attrs    map[string]string
	controls []*htmlControl
}

type htmlControl struct {
	tag   string
	attrs map[string]string
	text  string
}

type formField struct {
	name, value string
}

func parseHTMLDoc(src string) (*htmlDoc, error) {
	root, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return nil, err
	}
	doc := &htmlDoc{root: root}
	walkNodes(root, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "form" {
			form := &htmlForm{attrs: nodeAttrs(n)}
			walkNodes(n, func(c *html.Node) bool {
				if c == n || c.Type != html.ElementNode {
					return true
				}
				switch c.Data {
				case "input", "button", "textarea":
					ctl := &htmlControl{tag: c.Data, attrs: nodeAttrs(c)}
					if c.Data != "input" {
						ctl.text = nodeText(c)
					}
					form.controls = append(form.controls, ctl)
				}
				return true
			})
			doc.forms = append(doc.forms, form)
			return false
		}
		return true
	})
	return doc, nil
}

// walkNodes visits n and its descendants in document order, fn returning
// false skips the children of that node
func walkNodes(n *html.Node, fn func(*html.Node) bool) {
	if !fn(n) {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkNodes(c, fn)
	}
}

// nodeAttrs lowercases names and keeps the first occurrence like a browser
func nodeAttrs(n *html.Node) map[string]string {
	attrs := make(map[string]string, len(n.Attr))
	for _, a := range n.Attr {
		key := strings.ToLower(a.Key)
		if _, ok := attrs[key]; !ok {
			attrs[key] = a.Val
		}
	}
	return attrs
}

// nodeText is the whitespace-collapsed text of n
func nodeText(n *html.Node) string {
	var parts []string
	walkNodes(n, func(c *html.Node) bool {
		if c.Type == html.TextNode {
			parts = append(parts, c.Data)
		}
		return true
	})
	return strings.Join(strings.Fields(strings.Join(parts, " ")), " ")
}

func hasClass(attrs map[string]string, class string) bool {
	for _, c := range strings.Fields(attrs["class"]) {
		if c == class {
			return true
		}
	}
	return false
}

// firstElement returns the attributes and text of the first element matching pred
func (d *htmlDoc) firstElement(pred func(tag string, attrs map[string]string) bool) (map[string]string, string, bool) {
	var found *html.Node
	walkNodes(d.root, func(n *html.Node) bool {
		if found != nil {
			return false
		}
		if n.Type == html.ElementNode && pred(n.Data, nodeAttrs(n)) {
			found = n
			return false
		}
		return true
	})
	if found == nil {
		return nil, "", false
	}
	return nodeAttrs(found), nodeText(found), true
}

// firstText is the text of the first element matching pred, empty when that
// element has none
func (d *htmlDoc) firstText(pred func(attrs map[string]string) bool) string {
	_, text, _ := d.firstElement(func(_ string, attrs map[string]string) bool { return pred(attrs) })
	return text
}

func (d *htmlDoc) formByID(id string) *htmlForm {
	for _, f := range d.forms {
		if f.attrs["id"] == id {
			return f
		}
	}
	return nil
}

func (d *htmlDoc) formWithAction(fragment string) *htmlForm {
	fragment = strings.ToLower(fragment)
	for _, f := range d.forms {
		if strings.Contains(strings.ToLower(f.attrs["action"]), fragment) {
			return f
		}
	}
	return nil
}

func (c *htmlControl) name() string { return c.attrs["name"] }

func (c *htmlControl) typ() string {
	if t := strings.ToLower(c.attrs["type"]); t != "" {
		return t
	}
	if c.tag == "button" {
		return "submit"
	}
	return "text"
}

func (c *htmlControl) value() string {
	if c.tag == "textarea" {
		return c.text
	}
	if v, ok := c.attrs["value"]; ok {
		return v
	}
	if t := c.typ(); t == "checkbox" || t == "radio" {
		return "on"
	}
	return ""
}

func (c *htmlControl) isSubmitButton() bool {
	switch c.tag {
	case "button":
		return c.typ() == "submit"
	case "input":
		return c.typ() == "submit" || c.typ() == "image"
	}
	return false
}

func (f *htmlForm) method() string {
	if m := strings.ToLower(f.attrs["method"]); m != "" {
		return m
	}
	return "get"
}

func (f *htmlForm) controlByID(id string) *htmlControl {
	for _, c := range f.controls {
		if c.attrs["id"] == id {
			return c
		}
	}
	return nil
}

func (f *htmlForm) emailControl() *htmlControl {
	for _, c := range f.controls {
		if c.typ() == "email" && c.name() != "" {
			return c
		}
	}
	for _, c := range f.controls {
		if strings.HasSuffix(strings.ToLower(c.name()), "email") {
			return c
		}
	}
	return nil
}

func (f *htmlForm) passwordControl() *htmlControl {
	for _, c := range f.controls {
		if c.typ() == "password" && c.name() != "" {
			return c
		}
	}
	return nil
}

// fields builds what a browser submits: named enabled controls, checked boxes
// only and buttons only when they are the submitter. values overrides by
// name and checking ticks boxes
func (f *htmlForm) fields(values map[string]string, checking map[string]bool, submitter *htmlControl) []formField {
	var out []formField
	for _, c := range f.controls {
		name := c.name()
		if name == "" {
			continue
		}
		if _, disabled := c.attrs["disabled"]; disabled {
			continue
		}
		t := c.typ()
		if c.tag == "button" || t == "submit" || t == "image" || t == "button" || t == "reset" {
			if c == submitter {
				out = append(out, formField{name, c.value()})
			}
			continue
		}
		switch t {
		case "file":
			continue
		case "checkbox", "radio":
			if _, checked := c.attrs["checked"]; checked || checking[name] {
				out = append(out, formField{name, c.value()})
			}
		default:
			v, ok := values[name]
			if !ok {
				v = c.value()
			}
			out = append(out, formField{name, v})
		}
	}
	return out
}

// actionURL resolves the form (or submitter formaction) against the page.
// Page query items such as returnTo that the action leaves out are carried
// over when both point at the same path, as the live flow expects
func (f *htmlForm) actionURL(pageURL *url.URL, submitter *htmlControl) (*url.URL, error) {
	raw := f.attrs["action"]
	if submitter != nil {
		if fa, ok := submitter.attrs["formaction"]; ok {
			raw = fa
		}
	}
	resolved, err := pageURL.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("bad form action %q: %w", raw, err)
	}
	resolved.Fragment = ""
	if resolved.Host == pageURL.Host && resolved.Path == pageURL.Path && pageURL.RawQuery != "" {
		var items []string
		existing := map[string]bool{}
		if resolved.RawQuery != "" {
			items = strings.Split(resolved.RawQuery, "&")
			for _, it := range items {
				existing[strings.ToLower(strings.SplitN(it, "=", 2)[0])] = true
			}
		}
		for _, it := range strings.Split(pageURL.RawQuery, "&") {
			if !existing[strings.ToLower(strings.SplitN(it, "=", 2)[0])] {
				items = append(items, it)
			}
		}
		resolved.RawQuery = strings.Join(items, "&")
	}
	return resolved, nil
}

// formEncode follows the HTML application/x-www-form-urlencoded rules: space
// becomes +, everything but alphanumerics and *-._ is percent escaped
func formEncode(fields []formField) string {
	enc := func(s string) string {
		var b strings.Builder
		for i := 0; i < len(s); i++ {
			ch := s[i]
			switch {
			case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9',
				ch == '*', ch == '-', ch == '.', ch == '_':
				b.WriteByte(ch)
			case ch == ' ':
				b.WriteByte('+')
			default:
				fmt.Fprintf(&b, "%%%02X", ch)
			}
		}
		return b.String()
	}
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = enc(f.name) + "=" + enc(f.value)
	}
	return strings.Join(parts, "&")
}
