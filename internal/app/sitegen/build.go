package sitegen

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/freeDog-wy/go-backend-template/internal/app/pkg/cmsclient"
)

// App coordinates a complete, repeatable static site build.
type App struct {
	cfg      Config
	public   *cmsclient.PublicClient
	markdown *MarkdownRenderer
	renderer *renderer
}

type BuildStats struct {
	Locales  int
	Articles int
	Pages    int
}

type localeSnapshot struct {
	Locale           Locale
	Categories       []Category
	CategoriesBySlug map[string]Category
	Tags             []Tag
	Articles         []ArticleListItem
	ArticleDetails   map[string]Article
	CategoryArticles map[string][]ArticleListItem
	TagArticles      map[string][]ArticleListItem
	Sitemap          []SitemapEntry
	Redirects        []Redirect
}

func New(cfg Config) *App {
	public, err := cmsclient.NewPublic(cfg.APIBaseURL.String(), &http.Client{Timeout: cfg.HTTPTimeout})
	if err != nil {
		panic(fmt.Sprintf("create sitegen CMS client: %v", err))
	}
	return &App{
		cfg:      cfg,
		public:   public,
		markdown: NewMarkdownRenderer(),
		renderer: newRenderer(),
	}
}

func (a *App) Build(ctx context.Context) (BuildStats, error) {
	locales, err := a.public.ListLocales(ctx)
	if err != nil {
		return BuildStats{}, err
	}
	defaultLocale, localeByCode, err := validateLocales(locales)
	if err != nil {
		return BuildStats{}, err
	}

	snapshots := make(map[string]*localeSnapshot, len(locales))
	for _, locale := range locales {
		if !locale.IsEnabled {
			continue
		}
		snapshot, err := a.loadLocale(ctx, locale)
		if err != nil {
			return BuildStats{}, fmt.Errorf("load locale %s: %w", locale.Code, err)
		}
		snapshots[locale.Code] = snapshot
	}

	writer, err := newStagingWriter(a.cfg.OutputDir)
	if err != nil {
		return BuildStats{}, err
	}
	defer writer.Abort()
	for _, asset := range []string{"theme-init.js", "app.js", "site.css", "tool-base64.js", "tool-json.js"} {
		if err := writer.CopyEmbeddedFile(siteFiles, "assets/"+asset, "assets/"+asset); err != nil {
			return BuildStats{}, err
		}
	}

	stats := BuildStats{Locales: len(snapshots)}
	for _, locale := range locales {
		if !locale.IsEnabled {
			continue
		}
		snapshot := snapshots[locale.Code]
		pages, articles, err := a.renderLocale(writer, snapshot, locales, localeByCode, snapshots, defaultLocale)
		if err != nil {
			return BuildStats{}, fmt.Errorf("render locale %s: %w", locale.Code, err)
		}
		stats.Pages += pages
		stats.Articles += articles
		toolPages, err := a.renderTools(writer, snapshot, locales)
		if err != nil {
			return BuildStats{}, fmt.Errorf("render tools for locale %s: %w", locale.Code, err)
		}
		stats.Pages += toolPages
	}

	defaultSnapshot := snapshots[defaultLocale.Code]
	if err := a.writeNotFound(writer, defaultSnapshot, locales); err != nil {
		return BuildStats{}, err
	}
	stats.Pages++
	if err := a.writeSitemap(writer, snapshots); err != nil {
		return BuildStats{}, err
	}
	if err := writer.WriteFile("robots.txt", []byte("User-agent: *\nAllow: /\nSitemap: "+a.cfg.absoluteURL("/sitemap.xml")+"\n")); err != nil {
		return BuildStats{}, err
	}
	if err := a.writeRedirects(writer, snapshots, defaultLocale.Code); err != nil {
		return BuildStats{}, err
	}
	if err := writer.Commit(); err != nil {
		return BuildStats{}, err
	}
	return stats, nil
}

func validateLocales(locales []Locale) (Locale, map[string]Locale, error) {
	byCode := make(map[string]Locale, len(locales))
	var defaultLocale *Locale
	for _, locale := range locales {
		if strings.TrimSpace(locale.Code) == "" || strings.TrimSpace(locale.Name) == "" || !locale.IsEnabled {
			return Locale{}, nil, fmt.Errorf("public locale list contains an invalid or disabled locale")
		}
		if _, exists := byCode[locale.Code]; exists {
			return Locale{}, nil, fmt.Errorf("duplicate locale %q", locale.Code)
		}
		copy := locale
		byCode[locale.Code] = copy
		if locale.IsDefault {
			if defaultLocale != nil {
				return Locale{}, nil, fmt.Errorf("more than one default locale")
			}
			defaultLocale = &copy
		}
	}
	if defaultLocale == nil {
		return Locale{}, nil, fmt.Errorf("public locale list has no default locale")
	}
	return *defaultLocale, byCode, nil
}

func (a *App) loadLocale(ctx context.Context, locale Locale) (*localeSnapshot, error) {
	categories, err := a.public.ListCategories(ctx, locale.Code)
	if err != nil {
		return nil, err
	}
	tags, err := loadAllPages(ctx, func(ctx context.Context, page int) ([]Tag, *pageMeta, error) {
		return a.public.ListTags(ctx, locale.Code, page, a.cfg.PerPage)
	})
	if err != nil {
		return nil, err
	}
	articles, err := loadAllPages(ctx, func(ctx context.Context, page int) ([]ArticleListItem, *pageMeta, error) {
		return a.public.ListArticles(ctx, locale.Code, page, a.cfg.PerPage)
	})
	if err != nil {
		return nil, err
	}
	sitemap, err := loadAllPages(ctx, func(ctx context.Context, page int) ([]SitemapEntry, *pageMeta, error) {
		return a.public.ListSitemapEntries(ctx, locale.Code, page, a.cfg.PerPage)
	})
	if err != nil {
		return nil, err
	}
	redirects, err := loadAllPages(ctx, func(ctx context.Context, page int) ([]Redirect, *pageMeta, error) {
		return a.public.ListRedirects(ctx, locale.Code, page, a.cfg.PerPage)
	})
	if err != nil {
		return nil, err
	}
	details, err := a.loadArticleDetails(ctx, locale.Code, articles)
	if err != nil {
		return nil, err
	}

	categoryArticles := make(map[string][]ArticleListItem)
	categoriesBySlug := make(map[string]Category)
	for _, category := range flattenCategories(categories) {
		if _, exists := categoriesBySlug[category.Slug]; exists {
			return nil, fmt.Errorf("duplicate category slug %q", category.Slug)
		}
		categoriesBySlug[category.Slug] = category
		items, err := loadAllPages(ctx, func(ctx context.Context, page int) ([]ArticleListItem, *pageMeta, error) {
			return a.public.ListCategoryArticles(ctx, locale.Code, category.Slug, page, a.cfg.PerPage)
		})
		if err != nil {
			return nil, fmt.Errorf("read category %s articles: %w", category.Slug, err)
		}
		categoryArticles[category.Slug] = items
	}
	tagArticles := make(map[string][]ArticleListItem)
	for _, tag := range tags {
		items, err := loadAllPages(ctx, func(ctx context.Context, page int) ([]ArticleListItem, *pageMeta, error) {
			return a.public.ListTagArticles(ctx, locale.Code, tag.Slug, page, a.cfg.PerPage)
		})
		if err != nil {
			return nil, fmt.Errorf("read tag %s articles: %w", tag.Slug, err)
		}
		tagArticles[tag.Slug] = items
	}

	return &localeSnapshot{
		Locale: locale, Categories: categories, CategoriesBySlug: categoriesBySlug, Tags: tags, Articles: articles,
		ArticleDetails: details, CategoryArticles: categoryArticles, TagArticles: tagArticles, Sitemap: sitemap, Redirects: redirects,
	}, nil
}

func (a *App) loadArticleDetails(ctx context.Context, locale string, items []ArticleListItem) (map[string]Article, error) {
	type result struct {
		item    ArticleListItem
		article Article
		err     error
	}
	jobs := make(chan ArticleListItem)
	results := make(chan result, len(items))
	var workers sync.WaitGroup
	for range a.cfg.Concurrency {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for item := range jobs {
				article, err := a.public.GetArticle(ctx, locale, item.Slug)
				results <- result{item: item, article: article, err: err}
			}
		}()
	}
	go func() {
		for _, item := range items {
			jobs <- item
		}
		close(jobs)
		workers.Wait()
		close(results)
	}()

	details := make(map[string]Article, len(items))
	for result := range results {
		if result.err != nil {
			return nil, fmt.Errorf("read article %s: %w", result.item.Slug, result.err)
		}
		if result.article.Locale != locale || result.article.Slug != result.item.Slug || result.article.ContentFormat != "markdown" {
			return nil, fmt.Errorf("article %s has an unexpected locale, slug, or content format", result.item.Slug)
		}
		details[result.item.Slug] = result.article
	}
	if len(details) != len(items) {
		return nil, fmt.Errorf("article detail response count does not match article list")
	}
	return details, nil
}

func (a *App) renderLocale(writer *stagingWriter, snapshot *localeSnapshot, locales []Locale, localeByCode map[string]Locale, snapshots map[string]*localeSnapshot, defaultLocale Locale) (int, int, error) {
	pages, articles := 0, 0
	base := func(head headView) pageBaseView {
		return a.baseView(snapshot, locales, head)
	}

	latest := limitCards(a.cards(snapshot.Locale.Code, snapshot.Articles), 6)
	home := homeView{
		pageBaseView: base(a.standardHead(snapshot.Locale, snapshot.Locale.Name, "", localeRoute(snapshot.Locale.Code))),
		Heading:      snapshot.Locale.Name, Categories: a.categoryNavs(snapshot.Locale.Code, snapshot.Categories), Articles: latest,
	}
	if err := a.writeTemplate(writer, "home.html", outputPath(localeRoute(snapshot.Locale.Code)), home); err != nil {
		return 0, 0, err
	}
	pages++

	for page, items := range paginate(snapshot.Articles, a.cfg.PerPage) {
		route := articlesRoute(snapshot.Locale.Code, page+1)
		view := listingView{
			pageBaseView: base(a.standardHead(snapshot.Locale, a.label(snapshot.Locale.Code, "articles"), "", route)),
			Heading:      a.label(snapshot.Locale.Code, "articles"), Articles: a.cards(snapshot.Locale.Code, items),
			Pagination: a.pagination(snapshot.Locale.Code, page+1, len(snapshot.Articles), func(p int) string { return articlesRoute(snapshot.Locale.Code, p) }),
		}
		if err := a.writeTemplate(writer, "listing.html", outputPath(route), view); err != nil {
			return 0, 0, err
		}
		pages++
	}

	for _, category := range flattenCategories(snapshot.Categories) {
		items := snapshot.CategoryArticles[category.Slug]
		for page, group := range paginate(items, a.cfg.PerPage) {
			route := categoryRoute(snapshot.Locale.Code, category.Slug, page+1)
			view := categoryView{listingView: listingView{
				pageBaseView: base(a.standardHead(snapshot.Locale, category.Name, category.Description, route)),
				Heading:      category.Name, Description: category.Description, Articles: a.cards(snapshot.Locale.Code, group),
				Pagination: a.pagination(snapshot.Locale.Code, page+1, len(items), func(p int) string { return categoryRoute(snapshot.Locale.Code, category.Slug, p) }),
			}, Children: a.categoryNavs(snapshot.Locale.Code, category.Children)}
			if err := a.writeTemplate(writer, "category.html", outputPath(route), view); err != nil {
				return 0, 0, err
			}
			pages++
		}
	}

	for _, tag := range snapshot.Tags {
		items := snapshot.TagArticles[tag.Slug]
		for page, group := range paginate(items, a.cfg.PerPage) {
			route := tagRoute(snapshot.Locale.Code, tag.Slug, page+1)
			view := listingView{
				pageBaseView: base(a.standardHead(snapshot.Locale, tag.Name, "", route)),
				Heading:      tag.Name, Articles: a.cards(snapshot.Locale.Code, group),
				Pagination: a.pagination(snapshot.Locale.Code, page+1, len(items), func(p int) string { return tagRoute(snapshot.Locale.Code, tag.Slug, p) }),
			}
			if err := a.writeTemplate(writer, "listing.html", outputPath(route), view); err != nil {
				return 0, 0, err
			}
			pages++
		}
	}

	for _, item := range snapshot.Articles {
		article := snapshot.ArticleDetails[item.Slug]
		rendered, err := a.markdown.Render(article.Content)
		if err != nil {
			return 0, 0, fmt.Errorf("render article %s: %w", article.Slug, err)
		}
		route := articleRoute(snapshot.Locale.Code, article.Slug)
		view := articleView{
			pageBaseView: base(a.articleHead(article, route, localeByCode, defaultLocale.Code)),
			Article:      article, Body: rendered.HTML, TOC: rendered.TOC, ReadingMinutes: rendered.ReadingMinutes,
			Languages: articleLanguages(article, localeByCode), ShowLanguageMenu: len(article.AvailableLocales) >= 2,
		}
		if err := a.writeTemplate(writer, "article.html", outputPath(route), view); err != nil {
			return 0, 0, err
		}
		pages++
		articles++
	}
	_ = snapshots // Snapshots are loaded before rendering to guarantee language targets exist.
	return pages, articles, nil
}

func (a *App) baseView(snapshot *localeSnapshot, locales []Locale, head headView) pageBaseView {
	return a.baseViewWithLocaleRoute(snapshot, locales, head, func(locale Locale) string {
		return localeRoute(locale.Code)
	})
}

func (a *App) baseViewWithLocaleRoute(snapshot *localeSnapshot, locales []Locale, head headView, routeForLocale func(Locale) string) pageBaseView {
	options := make([]localeOptionView, 0, len(locales))
	for _, locale := range locales {
		options = append(options, localeOptionView{Code: locale.Code, Name: locale.Name, URL: routeForLocale(locale), Current: locale.Code == snapshot.Locale.Code})
	}
	return pageBaseView{
		SiteName:      a.cfg.SiteName,
		CurrentLocale: localeOptionView{Code: snapshot.Locale.Code, Name: snapshot.Locale.Name, URL: routeForLocale(snapshot.Locale), Current: true},
		Locales:       options, Navigation: a.categoryNavs(snapshot.Locale.Code, snapshot.Categories), Labels: localizedLabels(snapshot.Locale.Code), Head: head, HomeURL: localeRoute(snapshot.Locale.Code),
	}
}

func (a *App) renderTools(writer *stagingWriter, snapshot *localeSnapshot, locales []Locale) (int, error) {
	pages := 0
	indexRoute := toolsRoute(snapshot.Locale.Code)
	indexRouteForLocale := func(locale Locale) string {
		return toolsRoute(locale.Code)
	}
	indexBase := a.baseViewWithLocaleRoute(snapshot, locales, a.toolsHead(indexRoute, a.label(snapshot.Locale.Code, "tools"), a.toolsDescription(snapshot.Locale.Code), locales, indexRouteForLocale), indexRouteForLocale)
	indexView := toolsIndexView{pageBaseView: indexBase, Heading: a.label(snapshot.Locale.Code, "tools"), Description: a.toolsDescription(snapshot.Locale.Code), Tools: staticTools(snapshot.Locale.Code)}
	if err := a.writeTemplate(writer, "tools-index.html", outputPath(indexRoute), indexView); err != nil {
		return 0, err
	}
	pages++

	for _, definition := range toolDefinitions {
		copy := definition.copyFor(snapshot.Locale.Code)
		route := toolRoute(snapshot.Locale.Code, definition.ID)
		routeForLocale := func(locale Locale) string {
			return toolRoute(locale.Code, definition.ID)
		}
		base := a.baseViewWithLocaleRoute(snapshot, locales, a.toolsHead(route, copy.Title, copy.Description, locales, routeForLocale), routeForLocale)
		base.Scripts = []string{definition.Script}
		view := toolPageView{pageBaseView: base, Tool: toolCardView{ID: definition.ID, Title: copy.Title, Description: copy.Description, URL: route, Icon: definition.Icon}, Copy: copy}
		if err := a.writeTemplate(writer, definition.Template, outputPath(route), view); err != nil {
			return 0, err
		}
		pages++
	}
	return pages, nil
}

func (a *App) standardHead(locale Locale, title, description, route string) headView {
	if description == "" {
		description = a.cfg.SiteName
	}
	return headView{Title: joinTitle(title, a.cfg.SiteName), Description: summaryDescription(description), Canonical: a.cfg.absoluteURL(route), Kind: "website"}
}

func (a *App) toolsHead(route, title, description string, locales []Locale, routeForLocale func(Locale) string) headView {
	head := headView{Title: joinTitle(title, a.cfg.SiteName), Description: summaryDescription(description), Canonical: a.cfg.absoluteURL(route), Kind: "website"}
	for _, alternate := range locales {
		head.Hreflangs = append(head.Hreflangs, hreflangView{Locale: alternate.Code, URL: a.cfg.absoluteURL(routeForLocale(alternate))})
	}
	return head
}

func (a *App) articleHead(article Article, route string, localeByCode map[string]Locale, defaultLocale string) headView {
	title := article.SEOTitle
	if title == "" {
		title = article.Title
	}
	description := article.SEODescription
	if description == "" {
		description = article.Summary
	}
	canonical := a.cfg.absoluteURL(route)
	if approvedCanonical(article.CanonicalURL, a.cfg.SiteURL) {
		canonical = article.CanonicalURL
	}
	hreflangs := make([]hreflangView, 0, len(article.AvailableLocales)+1)
	for _, ref := range article.AvailableLocales {
		if _, ok := localeByCode[ref.Locale]; !ok {
			continue
		}
		url := a.cfg.absoluteURL(articleRoute(ref.Locale, ref.Slug))
		hreflangs = append(hreflangs, hreflangView{Locale: ref.Locale, URL: url})
		if ref.Locale == defaultLocale {
			hreflangs = append(hreflangs, hreflangView{URL: url, Default: true})
		}
	}
	image := ""
	if article.Cover != nil {
		image = article.Cover.URL
	}
	return headView{Title: joinTitle(title, a.cfg.SiteName), Description: summaryDescription(description), Canonical: canonical, Hreflangs: hreflangs, Image: image, Kind: "article"}
}

func approvedCanonical(value string, siteURL *url.URL) bool {
	u, err := url.Parse(strings.TrimSpace(value))
	return err == nil && u.Scheme == siteURL.Scheme && u.Host == siteURL.Host && u.RawQuery == "" && u.Fragment == "" && u.Path != ""
}

func (a *App) cards(locale string, items []ArticleListItem) []articleCardView {
	cards := make([]articleCardView, 0, len(items))
	for _, item := range items {
		var category *categoryNavView
		if item.PrimaryCategory != nil {
			category = &categoryNavView{Name: item.PrimaryCategory.Name, URL: categoryRoute(locale, item.PrimaryCategory.Slug, 1)}
		}
		cards = append(cards, articleCardView{Title: item.Title, Summary: item.Summary, URL: articleRoute(locale, item.Slug), PublishedAt: item.PublishedAt, Category: category, Cover: item.Cover})
	}
	return cards
}

func (a *App) categoryNavs(locale string, categories []Category) []categoryNavView {
	result := make([]categoryNavView, 0, len(categories))
	for _, category := range categories {
		result = append(result, categoryNavView{Name: category.Name, URL: categoryRoute(locale, category.Slug, 1)})
	}
	return result
}

func (a *App) pagination(locale string, current, totalItems int, route func(int) string) paginationView {
	total := pageCount(totalItems, a.cfg.PerPage)
	labels := localizedLabels(locale)
	view := paginationView{Current: current, Total: total, PreviousLabel: labels.Previous, NextLabel: labels.Next, PageLabel: labels.Page}
	if current > 1 {
		view.Previous = route(current - 1)
	}
	if current < total {
		view.Next = route(current + 1)
	}
	return view
}

func (a *App) writeTemplate(writer *stagingWriter, templateName, path string, data any) error {
	rendered, err := a.renderer.Render(templateName, data)
	if err != nil {
		return err
	}
	return writer.WriteFile(path, rendered)
}

func (a *App) writeNotFound(writer *stagingWriter, defaultSnapshot *localeSnapshot, locales []Locale) error {
	base := a.baseView(defaultSnapshot, locales, a.standardHead(defaultSnapshot.Locale, "404", "", "/404/"))
	return a.writeTemplate(writer, "404.html", "404.html", notFoundView{pageBaseView: base})
}

func (a *App) writeSitemap(writer *stagingWriter, snapshots map[string]*localeSnapshot) error {
	type sitemapURL struct {
		Location string `xml:"loc"`
		LastMod  string `xml:"lastmod"`
	}
	type sitemap struct {
		XMLName xml.Name     `xml:"urlset"`
		XMLNS   string       `xml:"xmlns,attr"`
		URLs    []sitemapURL `xml:"url"`
	}
	urls := make([]sitemapURL, 0)
	for _, snapshot := range snapshots {
		for _, entry := range snapshot.Sitemap {
			route, ok := canonicalContentRoute(entry.URL)
			if !ok {
				return fmt.Errorf("invalid sitemap path %q", entry.URL)
			}
			urls = append(urls, sitemapURL{Location: a.cfg.absoluteURL(route), LastMod: entry.LastModified.Format("2006-01-02")})
		}
		urls = append(urls, sitemapURL{Location: a.cfg.absoluteURL(toolsRoute(snapshot.Locale.Code))})
		for _, definition := range toolDefinitions {
			urls = append(urls, sitemapURL{Location: a.cfg.absoluteURL(toolRoute(snapshot.Locale.Code, definition.ID))})
		}
	}
	sort.Slice(urls, func(i, j int) bool { return urls[i].Location < urls[j].Location })
	data, err := xml.MarshalIndent(sitemap{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: urls}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sitemap: %w", err)
	}
	return writer.WriteFile("sitemap.xml", append([]byte(xml.Header), append(data, '\n')...))
}

func (a *App) writeRedirects(writer *stagingWriter, snapshots map[string]*localeSnapshot, defaultLocale string) error {
	lines := make(map[string]string)
	add := func(source, target string, status int) error {
		if !strings.HasPrefix(source, "/") || strings.HasPrefix(source, "//") || !strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") {
			return fmt.Errorf("redirect must use site-root paths")
		}
		if status != 301 && status != 308 {
			return fmt.Errorf("unsupported redirect status %d", status)
		}
		line := target + " " + fmt.Sprintf("%d", status)
		if existing, ok := lines[source]; ok && existing != line {
			return fmt.Errorf("conflicting redirect source %q", source)
		}
		lines[source] = line
		return nil
	}
	if err := add("/", localeRoute(defaultLocale), 301); err != nil {
		return err
	}
	for _, snapshot := range snapshots {
		for _, redirect := range snapshot.Redirects {
			target := redirect.TargetPath
			if normalized, ok := canonicalContentRoute(target); ok {
				target = normalized
			}
			if err := add(redirect.SourcePath, target, redirect.StatusCode); err != nil {
				return err
			}
			if normalized, ok := canonicalContentRoute(redirect.SourcePath); ok && normalized != redirect.SourcePath {
				if err := add(normalized, target, redirect.StatusCode); err != nil {
					return err
				}
			}
		}
	}
	sources := make([]string, 0, len(lines))
	for source := range lines {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	var output strings.Builder
	for _, source := range sources {
		output.WriteString(source)
		output.WriteByte(' ')
		output.WriteString(lines[source])
		output.WriteByte('\n')
	}
	return writer.WriteFile("_redirects", []byte(output.String()))
}

func articleLanguages(article Article, locales map[string]Locale) []articleLanguageView {
	result := make([]articleLanguageView, 0, len(article.AvailableLocales))
	for _, ref := range article.AvailableLocales {
		locale, ok := locales[ref.Locale]
		if !ok {
			continue
		}
		result = append(result, articleLanguageView{Name: locale.Name, Code: locale.Code, URL: articleRoute(ref.Locale, ref.Slug), Current: ref.Locale == article.Locale})
	}
	return result
}

func flattenCategories(categories []Category) []Category {
	result := make([]Category, 0)
	var visit func([]Category)
	visit = func(items []Category) {
		for _, item := range items {
			result = append(result, item)
			visit(item.Children)
		}
	}
	visit(categories)
	return result
}

func paginate[T any](items []T, size int) [][]T {
	if len(items) == 0 {
		return [][]T{{}}
	}
	pages := make([][]T, 0, pageCount(len(items), size))
	for start := 0; start < len(items); start += size {
		end := min(start+size, len(items))
		pages = append(pages, items[start:end])
	}
	return pages
}

func pageCount(total, size int) int {
	if total == 0 {
		return 1
	}
	return (total + size - 1) / size
}

func limitCards(cards []articleCardView, maxCards int) []articleCardView {
	if len(cards) <= maxCards {
		return cards
	}
	return cards[:maxCards]
}

func joinTitle(title, siteName string) string {
	if strings.TrimSpace(title) == "" {
		return siteName
	}
	return title + " | " + siteName
}

func (a *App) label(locale, key string) string {
	labels := localizedLabels(locale)
	switch key {
	case "articles":
		return labels.Articles
	case "tools":
		return labels.Tools
	default:
		return key
	}
}

func (a *App) toolsDescription(locale string) string {
	if strings.HasPrefix(strings.ToLower(locale), "zh") {
		return "无需上传数据的浏览器本地工具。"
	}
	return "Browser-based tools that keep your data on this device."
}
