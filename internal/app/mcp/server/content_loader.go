package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const maxArticleContentBytes = 10 << 20

// contentLoader confines MCP-provided paths to the locally configured staging
// directory. It is intentionally an MCP adapter concern, not a CMS API concern.
type contentLoader struct {
	root string
}

func newContentLoader(root string) contentLoader {
	return contentLoader{root: strings.TrimSpace(root)}
}

func (l contentLoader) load(name string) (string, string, error) {
	if l.root == "" {
		return "", "", fmt.Errorf("CMS_CONTENT_ROOT is required when content_file is set")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", fmt.Errorf("content_file is required")
	}
	if filepath.IsAbs(name) {
		return "", "", fmt.Errorf("content_file must be relative to CMS_CONTENT_ROOT")
	}
	cleanName := filepath.Clean(name)
	if cleanName == "." || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("content_file must remain below CMS_CONTENT_ROOT")
	}

	root, err := filepath.EvalSymlinks(l.root)
	if err != nil {
		return "", "", fmt.Errorf("resolve CMS_CONTENT_ROOT: %w", err)
	}
	rootInfo, err := os.Stat(root)
	if err != nil {
		return "", "", fmt.Errorf("stat CMS_CONTENT_ROOT: %w", err)
	}
	if !rootInfo.IsDir() {
		return "", "", fmt.Errorf("CMS_CONTENT_ROOT must be a directory")
	}

	path, err := filepath.EvalSymlinks(filepath.Join(root, cleanName))
	if err != nil {
		return "", "", fmt.Errorf("resolve content_file: %w", err)
	}
	if !pathWithin(root, path) {
		return "", "", fmt.Errorf("content_file must remain below CMS_CONTENT_ROOT")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", "", fmt.Errorf("stat content_file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("content_file must be a regular file")
	}

	file, err := os.Open(path)
	if err != nil {
		return "", "", fmt.Errorf("open content_file: %w", err)
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, maxArticleContentBytes+1))
	if err != nil {
		return "", "", fmt.Errorf("read content_file: %w", err)
	}
	if len(body) > maxArticleContentBytes {
		return "", "", fmt.Errorf("content_file exceeds %d byte limit", maxArticleContentBytes)
	}
	if !utf8.Valid(body) {
		return "", "", fmt.Errorf("content_file must be valid UTF-8")
	}
	sum := sha256.Sum256(body)
	return string(body), hex.EncodeToString(sum[:]), nil
}

func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

type resolvedArticleWrite struct {
	input         articleWriteInput
	contentDigest string
}

func resolveArticleWrite(input articleWriteInput, loader contentLoader) (resolvedArticleWrite, error) {
	if strings.TrimSpace(input.ContentFile) == "" {
		return resolvedArticleWrite{input: input}, nil
	}
	content, digest, err := loader.load(input.ContentFile)
	if err != nil {
		return resolvedArticleWrite{}, err
	}
	input.Content = content
	input.ContentFile = ""
	return resolvedArticleWrite{input: input, contentDigest: digest}, nil
}

func (r resolvedArticleWrite) operationInput() any {
	if r.contentDigest == "" {
		return r.input
	}
	return struct {
		ArticleID      uint   `json:"article_id,omitempty"`
		Locale         string `json:"locale"`
		Title          string `json:"title"`
		Slug           string `json:"slug"`
		Summary        string `json:"summary,omitempty"`
		ContentDigest  string `json:"content_digest"`
		ContentFormat  string `json:"content_format,omitempty"`
		SEOTitle       string `json:"seo_title,omitempty"`
		SEODescription string `json:"seo_description,omitempty"`
		CanonicalURL   string `json:"canonical_url,omitempty"`
	}{
		ArticleID: r.input.ArticleID, Locale: r.input.Locale, Title: r.input.Title, Slug: r.input.Slug,
		Summary: r.input.Summary, ContentDigest: r.contentDigest, ContentFormat: r.input.ContentFormat,
		SEOTitle: r.input.SEOTitle, SEODescription: r.input.SEODescription, CanonicalURL: r.input.CanonicalURL,
	}
}
