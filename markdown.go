package goBrowser

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

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
