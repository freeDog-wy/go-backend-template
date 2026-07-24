package sitegen

import "github.com/freeDog-wy/go-backend-template/internal/app/pkg/cmsclient"

// Sitegen consumes the CMS public API models directly. Aliases keep template
// and rendering code independent from the API client's package path.
type Locale = cmsclient.Locale
type Category = cmsclient.Category
type Tag = cmsclient.Tag
type Cover = cmsclient.Cover
type CategoryRef = cmsclient.CategoryRef
type ArticleListItem = cmsclient.ArticleListItem
type ArticleLocale = cmsclient.ArticleLocale
type Article = cmsclient.Article
type SitemapEntry = cmsclient.SitemapEntry
type Redirect = cmsclient.Redirect
type pageMeta = cmsclient.PageMeta
