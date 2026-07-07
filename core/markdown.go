package goBrowser

import (
	"fmt"
	"hash/fnv"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func Merge(snapshots []string) (string, error) {
	switch len(snapshots) {
	case 0:
		return "", fmt.Errorf("no snapshots")
	case 1:
		return snapshots[0], nil
	}

	base, err := html.Parse(strings.NewReader(snapshots[0]))
	if err != nil {
		return "", fmt.Errorf("html.Parse: %w", err)
	}
	baseBody := findBody(base)
	if baseBody == nil {
		return snapshots[0], nil
	}

	for _, snap := range snapshots[1:] {
		doc, err := html.Parse(strings.NewReader(snap))
		if err != nil {
			continue
		}
		body := findBody(doc)
		if body == nil {
			continue
		}
		for c := body.FirstChild; c != nil; {
			next := c.NextSibling
			body.RemoveChild(c)
			baseBody.AppendChild(c)
			c = next
		}
	}

	var buf strings.Builder
	if err := html.Render(&buf, base); err != nil {
		return "", fmt.Errorf("html.Render: %w", err)
	}
	return buf.String(), nil
}

func findBody(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && strings.ToLower(n.Data) == "body" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if b := findBody(c); b != nil {
			return b
		}
	}
	return nil
}

func DedupTree(nodes []*Node) []*Node {
	seen := map[uint64]bool{}
	return dedupTreeWalk(nodes, seen)
}

func dedupTreeWalk(nodes []*Node, seen map[uint64]bool) []*Node {
	out := make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		if isDedupCandidate(n) {
			h := nodeTextHash(n)
			if h != 0 {
				if seen[h] {
					continue
				}
				seen[h] = true
			}
		}
		n.Children = dedupTreeWalk(n.Children, seen)
		if n.Type != "linebreak" && n.Text == "" && len(n.Children) == 0 {
			continue
		}
		out = append(out, n)
	}
	return out
}

func isDedupCandidate(n *Node) bool {
	switch n.Type {
	case "paragraph", "heading", "list_item", "blockquote", "code_block", "image":
		return true
	}
	return false
}

func nodeTextHash(n *Node) uint64 {
	if n == nil {
		return 0
	}
	var sb strings.Builder
	var walk func(*Node)
	walk = func(x *Node) {
		if x.Text != "" {
			sb.WriteString(x.Text)
			sb.WriteByte('\x1f')
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(n)
	if sb.Len() == 0 {
		return 0
	}
	h := fnv.New64a()
	h.Write([]byte(sb.String()))
	return h.Sum64()
}

func DedupMarkdownParagraphs(md string) string {
	lines := strings.Split(md, "\n")
	seen := map[string]bool{}
	var out []string
	var para []string

	flush := func() {
		if len(para) == 0 {
			return
		}
		key := strings.TrimSpace(strings.Join(para, "\n"))
		if key != "" && !seen[key] {
			seen[key] = true
			out = append(out, para...)
		}
		para = nil
	}

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			flush()
			if len(out) > 0 && out[len(out)-1] != "" {
				out = append(out, "")
			}
		} else {
			para = append(para, line)
		}
	}
	flush()

	return strings.TrimSpace(strings.Join(out, "\n"))
}

func InlineTimeElements(htmlSrc string) (string, error) {
	if !strings.Contains(htmlSrc, "<time") {
		return htmlSrc, nil
	}
	doc, err := html.Parse(strings.NewReader(htmlSrc))
	if err != nil {
		return "", fmt.Errorf("html.Parse: %w", err)
	}

	var collectText func(*html.Node, *strings.Builder)
	collectText = func(n *html.Node, b *strings.Builder) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			collectText(c, b)
		}
	}

	var timeNodes []*html.Node
	var collect func(*html.Node)
	collect = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.ToLower(n.Data) == "time" {
			timeNodes = append(timeNodes, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			collect(c)
		}
	}
	collect(doc)

	for _, t := range timeNodes {
		dt := ""
		for _, a := range t.Attr {
			if a.Key == "datetime" {
				dt = a.Val
				break
			}
		}
		var inner strings.Builder
		collectText(t, &inner)
		innerText := strings.TrimSpace(inner.String())

		var text string
		switch {
		case dt != "" && innerText != "":
			text = " [" + dt + "] " + innerText + " "
		case dt != "":
			text = " [" + dt + "] "
		case innerText != "":
			text = " " + innerText + " "
		}

		parent := t.Parent
		if parent == nil {
			continue
		}

		if text == "" {
			parent.RemoveChild(t)
			continue
		}

		textNode := &html.Node{Type: html.TextNode, Data: text}

		if parent.Type == html.ElementNode && strings.ToLower(parent.Data) == "a" {
			onlyChild := true
			for sib := parent.FirstChild; sib != nil; sib = sib.NextSibling {
				if sib == t {
					continue
				}
				if sib.Type == html.TextNode && strings.TrimSpace(sib.Data) == "" {
					continue
				}
				onlyChild = false
				break
			}
			if onlyChild {
				if grand := parent.Parent; grand != nil {
					grand.InsertBefore(textNode, parent)
					grand.RemoveChild(parent)
					continue
				}
			}
		}

		parent.InsertBefore(textNode, t)
		parent.RemoveChild(t)
	}

	var buf strings.Builder
	if err := html.Render(&buf, doc); err != nil {
		return "", fmt.Errorf("html.Render: %w", err)
	}
	return buf.String(), nil
}

type Node struct {
	Type     string  `json:"type"`
	Text     string  `json:"text,omitempty"`
	Level    int     `json:"level,omitempty"`
	Datetime string  `json:"datetime,omitempty"`
	Href     string  `json:"href,omitempty"`
	Src      string  `json:"src,omitempty"`
	Alt      string  `json:"alt,omitempty"`
	Children []*Node `json:"children,omitempty"`
}

func HTMLToNode(content, baseURL string, keepLinks bool) ([]*Node, error) {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("html.Parse: %w", err)
	}

	var walkChildren func(*html.Node) []*Node
	var walk func(*html.Node) []*Node

	walkChildren = func(n *html.Node) []*Node {
		var out []*Node
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			out = append(out, walk(c)...)
		}
		return out
	}

	walk = func(n *html.Node) []*Node {
		if n.Type == html.TextNode {
			t := strings.TrimSpace(n.Data)
			if t == "" {
				return nil
			}
			return []*Node{{Type: "text", Text: t}}
		}
		if n.Type != html.ElementNode {
			return walkChildren(n)
		}

		tag := strings.ToLower(n.Data)

		switch tag {
		case "script", "style", "noscript", "iframe", "form", "button",
			"input", "select", "textarea", "svg", "canvas", "video", "audio":
			return nil

		case "nav", "header", "footer", "aside":
			if !keepLinks {
				return nil
			}
			ch := walkChildren(n)
			if len(ch) == 0 {
				return nil
			}
			return []*Node{{Type: tag, Children: ch}}

		case "img":
			if !keepLinks {
				return nil
			}
			alt, src := extractImage(n, baseURL)
			if src == "" {
				return nil
			}
			return []*Node{{Type: "image", Src: src, Alt: alt}}

		case "h1", "h2", "h3", "h4", "h5", "h6":
			ch := walkChildren(n)
			if len(ch) == 0 {
				return nil
			}
			return []*Node{{Type: "heading", Level: int(tag[1] - '0'), Children: ch}}

		case "p":
			ch := walkChildren(n)
			if len(ch) == 0 {
				return nil
			}
			return []*Node{{Type: "paragraph", Children: ch}}

		case "br":
			return []*Node{{Type: "linebreak"}}

		case "li":
			ch := walkChildren(n)
			if len(ch) == 0 {
				return nil
			}
			return []*Node{{Type: "list_item", Children: ch}}

		case "ul":
			ch := walkChildren(n)
			if len(ch) == 0 {
				return nil
			}
			return []*Node{{Type: "list", Children: ch}}

		case "ol":
			ch := walkChildren(n)
			if len(ch) == 0 {
				return nil
			}
			return []*Node{{Type: "ordered_list", Children: ch}}

		case "strong", "b":
			ch := walkChildren(n)
			if len(ch) == 0 {
				return nil
			}
			return []*Node{{Type: "bold", Children: ch}}

		case "em", "i":
			ch := walkChildren(n)
			if len(ch) == 0 {
				return nil
			}
			return []*Node{{Type: "italic", Children: ch}}

		case "time":
			dt := attrMap(n)["datetime"]
			ch := walkChildren(n)
			if dt == "" && len(ch) == 0 {
				return nil
			}
			return []*Node{{Type: "time", Datetime: dt, Children: ch}}

		case "a":
			ch := walkChildren(n)
			if !keepLinks {
				return ch
			}
			href := resolveLink(attrMap(n)["href"], baseURL)
			if len(ch) == 0 {
				return nil
			}
			return []*Node{{Type: "link", Href: href, Children: ch}}

		case "blockquote":
			ch := walkChildren(n)
			if len(ch) == 0 {
				return nil
			}
			return []*Node{{Type: "blockquote", Children: ch}}

		case "code":
			ch := walkChildren(n)
			if len(ch) == 0 {
				return nil
			}
			return []*Node{{Type: "code", Children: ch}}

		case "pre":
			ch := walkChildren(n)
			if len(ch) == 0 {
				return nil
			}
			return []*Node{{Type: "code_block", Children: ch}}

		default:
			ch := walkChildren(n)
			if len(ch) == 0 {
				return nil
			}
			if isBlock(tag) {
				return []*Node{{Type: "container", Children: ch}}
			}
			return ch
		}
	}

	children := walkChildren(doc)
	children = DedupTree(children)
	return flattenChildren(children), nil
}

func flattenChildren(nodes []*Node) []*Node {
	out := make([]*Node, 0, len(nodes))
	for _, c := range nodes {
		c.Children = flattenChildren(c.Children)
		if len(c.Children) == 1 {
			out = append(out, c.Children[0])
			continue
		}
		out = append(out, c)
	}
	return out
}

func HTMLToMarkdown(content, baseURL string, keepLinks bool) (string, error) {
	node, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("html.Parse: %w", err)
	}

	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
			return
		}
		if n.Type != html.ElementNode {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			return
		}

		tag := strings.ToLower(n.Data)

		switch tag {
		case "script", "style", "noscript", "iframe", "form", "button",
			"input", "select", "textarea", "svg", "canvas", "video", "audio":
			return

		case "nav", "header", "footer", "aside":
			if !keepLinks {
				return
			}
			sb.WriteString("\n")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			sb.WriteString("\n")
			return

		case "img":
			if !keepLinks {
				return
			}
			alt, imgURL := extractImage(n, baseURL)
			if imgURL != "" {
				fmt.Fprintf(&sb, "\n![%s](%s)\n", alt, imgURL)
			}
			return

		case "h1", "h2", "h3", "h4", "h5", "h6":
			level := int(tag[1] - '0')
			sb.WriteString("\n")
			sb.WriteString(strings.Repeat("#", level))
			sb.WriteString(" ")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			sb.WriteString("\n")
			return

		case "p":
			sb.WriteString("\n")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			sb.WriteString("\n")
			return

		case "br":
			sb.WriteString("\n")
			return

		case "li":
			sb.WriteString("\n- ")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			return

		case "strong", "b":
			sb.WriteString("**")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			sb.WriteString("**")
			return

		case "em", "i":
			sb.WriteString("*")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			sb.WriteString("*")
			return

		case "time":
			if dt := attrMap(n)["datetime"]; dt != "" {
				sb.WriteString(" [")
				sb.WriteString(dt)
				sb.WriteString("] ")
				return
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			return

		case "a":
			if !keepLinks {
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				return
			}
			linkHref := attrMap(n)["href"]
			resolved := resolveLink(linkHref, baseURL)
			var text strings.Builder
			orig := sb
			sb = text
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			sb = orig
			t := strings.TrimSpace(text.String())
			if t != "" && resolved != "" {
				fmt.Fprintf(&sb, "[%s](%s)", t, resolved)
			} else if t != "" {
				sb.WriteString(t)
			}
			return

		case "blockquote":
			sb.WriteString("\n> ")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			sb.WriteString("\n")
			return

		case "code":
			sb.WriteString("`")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			sb.WriteString("`")
			return

		case "pre":
			sb.WriteString("\n```\n")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			sb.WriteString("\n```\n")
			return

		default:
			block := isBlock(tag)
			if block {
				sb.WriteString("\n")
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
			if block {
				sb.WriteString("\n")
			}
		}
	}

	walk(node)
	return collapse(sb.String()), nil
}

func resolveLink(linkHref, baseURL string) string {
	if linkHref == "" {
		return ""
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return linkHref
	}
	ref, err := url.Parse(linkHref)
	if err != nil {
		return linkHref
	}
	return base.ResolveReference(ref).String()
}

func isBlock(tag string) bool {
	switch tag {
	case "div", "section", "article", "main", "ul", "ol",
		"table", "thead", "tbody", "tr", "td", "th",
		"figure", "figcaption", "header", "footer", "nav", "aside":
		return true
	}
	return false
}

func extractImage(node *html.Node, baseURL string) (alt, src string) {
	attrs := attrMap(node)
	newSrc := attrs["data-src"]
	if newSrc == "" {
		newSrc = attrs["src"]
	}
	if newSrc == "" || strings.HasPrefix(newSrc, "data:") {
		return "", ""
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", ""
	}
	ref, err := url.Parse(newSrc)
	if err != nil {
		return "", ""
	}
	return attrs["alt"], base.ResolveReference(ref).String()
}

func attrMap(node *html.Node) map[string]string {
	m := make(map[string]string, len(node.Attr))
	for _, a := range node.Attr {
		m[a.Key] = a.Val
	}
	return m
}

func collapse(s string) string {
	var out []string
	blanks := 0
	for line := range strings.SplitSeq(s, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" {
			blanks++
			if blanks <= 1 {
				out = append(out, "")
			}
		} else if canSkipped(trim) {
			continue
		} else {
			blanks = 0
			out = append(out, trim)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func canSkipped(s string) bool {
	for _, r := range s {
		if r != '-' && r != '#' && r != '|' && r != '_' && r != '*' && r != ' ' && r != '\t' {
			return false
		}
	}
	return true
}
