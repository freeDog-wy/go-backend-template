package markdown

import (
	"strings"
	"testing"
)

func TestRendererRejectsRelativeImagesAndSanitizesRawHTML(t *testing.T) {
	renderer := NewRenderer()
	if _, err := renderer.Render("![cover](images/cover.png)"); err == nil {
		t.Fatal("relative Markdown image was accepted")
	}
	rendered, err := renderer.Render("## Safe\n\n<script>alert(1)</script>\n\n**content**")
	if err != nil {
		t.Fatalf("render Markdown: %v", err)
	}
	if strings.Contains(string(rendered.HTML), "script") {
		t.Fatal("raw HTML reached rendered output")
	}
	if len(rendered.TOC) != 1 || rendered.TOC[0].Text != "Safe" {
		t.Fatalf("unexpected table of contents: %#v", rendered.TOC)
	}
}
