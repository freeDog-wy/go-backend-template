package markdown

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type TOCEntry struct {
	Level int    `json:"level"`
	ID    string `json:"id"`
	Text  string `json:"text"`
}

type Rendered struct {
	HTML           template.HTML `json:"html"`
	TOC            []TOCEntry    `json:"toc"`
	ReadingMinutes int           `json:"reading_minutes"`
}

type Renderer struct {
	markdown goldmark.Markdown
	policy   *bluemonday.Policy
}

func NewRenderer() *Renderer {
	return &Renderer{
		markdown: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		),
		policy: articleHTMLPolicy(),
	}
}

func (r *Renderer) Render(source string) (Rendered, error) {
	sourceBytes := []byte(source)
	root := r.markdown.Parser().Parse(text.NewReader(sourceBytes))
	toc := make([]TOCEntry, 0)
	if err := ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch current := node.(type) {
		case *ast.Link:
			if !isAllowedLinkURL(string(current.Destination)) {
				return ast.WalkStop, fmt.Errorf("Markdown link URL %q uses an unsupported scheme", current.Destination)
			}
		case *ast.Image:
			if !isAllowedImageURL(string(current.Destination)) {
				return ast.WalkStop, fmt.Errorf("Markdown image URL %q must be an absolute HTTP(S) URL or a site-root path", current.Destination)
			}
		case *ast.Heading:
			if current.Level < 2 || current.Level > 4 {
				return ast.WalkContinue, nil
			}
			id, ok := current.AttributeString("id")
			if !ok {
				return ast.WalkStop, fmt.Errorf("Markdown heading is missing an ID")
			}
			idBytes, ok := id.([]byte)
			if !ok {
				return ast.WalkStop, fmt.Errorf("Markdown heading has an invalid ID")
			}
			toc = append(toc, TOCEntry{Level: current.Level, ID: string(idBytes), Text: normalizeText(string(current.Text(sourceBytes)))})
		}
		return ast.WalkContinue, nil
	}); err != nil {
		return Rendered{}, err
	}

	var rendered bytes.Buffer
	if err := r.markdown.Convert(sourceBytes, &rendered); err != nil {
		return Rendered{}, fmt.Errorf("render Markdown: %w", err)
	}
	safeHTML := r.policy.SanitizeBytes(rendered.Bytes())
	return Rendered{HTML: template.HTML(safeHTML), TOC: toc, ReadingMinutes: readingMinutes(source)}, nil
}

func articleHTMLPolicy() *bluemonday.Policy {
	policy := bluemonday.NewPolicy()
	policy.AllowElements("p", "br", "strong", "em", "del", "blockquote", "hr", "ul", "ol", "li", "pre", "code", "table", "thead", "tbody", "tr", "th", "td")
	policy.AllowElements("h2", "h3", "h4", "a", "img", "input")
	policy.AllowAttrs("id").OnElements("h2", "h3", "h4")
	policy.AllowAttrs("href", "title").OnElements("a")
	policy.AllowAttrs("src", "alt", "title", "width", "height").OnElements("img")
	policy.AllowAttrs("class").Matching(regexp.MustCompile(`^language-[a-zA-Z0-9+_-]+$`)).OnElements("code")
	policy.AllowAttrs("type", "checked", "disabled").OnElements("input")
	policy.AllowURLSchemes("http", "https", "mailto")
	policy.AllowRelativeURLs(true)
	policy.RequireNoReferrerOnLinks(true)
	return policy
}

func isAllowedImageURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" && parsed.Host != "" {
		return false
	}
	if strings.HasPrefix(value, "/") {
		return !strings.HasPrefix(value, "//")
	}
	return parsed.IsAbs() && (parsed.Scheme == "https" || parsed.Scheme == "http")
}

func isAllowedLinkURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" && parsed.Host != "" {
		return false
	}
	if parsed.Scheme == "" {
		return true
	}
	return parsed.Scheme == "https" || parsed.Scheme == "http" || parsed.Scheme == "mailto"
}

func readingMinutes(source string) int {
	chineseChars, words := 0, 0
	inWord := false
	for _, character := range source {
		if unicode.Is(unicode.Han, character) {
			chineseChars++
		}
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			if !inWord {
				words++
				inWord = true
			}
		} else {
			inWord = false
		}
	}
	minutesByChinese := (chineseChars + 399) / 400
	minutesByWords := (words + 199) / 200
	if minutesByChinese > minutesByWords {
		return max(1, minutesByChinese)
	}
	return max(1, minutesByWords)
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
