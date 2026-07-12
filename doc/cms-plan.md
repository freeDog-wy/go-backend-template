# 单站点多语言 Headless CMS 方案

## 1. 目标与边界

本项目是单站点、多语言、带后台管理的 Headless CMS。后端负责内容管理、发布、媒体、权限、公共内容 API 和 SEO 元数据；前端负责页面渲染、浏览器路由和最终 HTML SEO 输出。

第一版不实现：

- 多站点、多租户或工作空间隔离。
- 用户自定义内容模型（Content Type Builder）。
- 复杂审批流、协作编辑、评论系统或全文搜索集群。
- 数据库 Blob 文件存储。

这些能力在真实需求出现后以独立模块演进，不能预先污染基础内容模型。

## 2. 核心原则

- 站点结构在所有语言间共享；内容翻译、slug、SEO 元数据和发布状态按语言独立管理。
- 一种语言可先于其他语言发布。同一逻辑文章的中文与英文不要求同时上线。
- 所有面向前端的 API 无论成功或失败均返回 HTTP `200`，由 `success` 和 `error.code` 表达结果。
- 数据结构变更只能通过版本化 SQL migration 完成，不使用运行时 `AutoMigrate`。
- 媒体文件存对象存储，PostgreSQL 仅保存对象 key、元数据与访问策略。
- 内容发布、下线、重定向更新和缓存失效通过 Outbox 发送领域事件。

## 3. 语言模型

支持语言使用 BCP 47 locale，例如 `zh-CN`、`en-US`。应有一个默认语言，并明确允许发布的语言集合。

建议使用 `locales` 表管理语言：

```text
locales
  code                 primary key，例如 zh-CN
  name
  is_default
  is_enabled
  sort_order
```

对于缺失翻译的公开请求，默认返回 `CONTENT_TRANSLATION_NOT_FOUND`，不自动回退到其他语言。是否允许前端显式回退由产品策略决定；第一版保持严格，避免不同语言 URL 返回错误内容。

## 4. 内容模型

### 4.1 文章与翻译

```text
articles
  id
  author_user_id
  cover_media_id
  created_at
  updated_at
  deleted_at

article_translations
  id
  article_id
  locale
  title
  slug
  summary
  content
  content_format
  status               draft | published | archived
  published_at
  seo_title
  seo_description
  canonical_url
  created_at
  updated_at
```

约束：

- `UNIQUE(article_id, locale)`：每篇逻辑文章每种语言最多一份翻译。
- `UNIQUE(locale, slug)`：公开文章 URL 在同一语言内稳定且唯一。
- `status` 和 `published_at` 放在翻译表中，使每个语言版本可独立发布。
- 公开 API 只读取 `status = published` 且 `published_at <= now()` 的翻译。

### 4.2 分类树

分类使用邻接表模型：

```text
categories
  id
  parent_id            nullable，根分类为 NULL
  sort_order
  is_enabled
  created_at
  updated_at

category_translations
  id
  category_id
  locale
  name
  slug
  description
  seo_title
  seo_description
```

规则：

- 最大层级限制为 3 或 4 层。
- 禁止将自身或子节点设为父节点，防止循环。
- 有子分类或文章关联的分类不可直接删除；应迁移内容或先停用。
- 同一语言内分类 slug 使用 `UNIQUE(locale, slug)`，第一版不把分类路径作为文章永久 URL。
- 后台一次读取全部分类后在应用层组装树；公共导航树使用 Redis 缓存。

### 4.3 标签与文章分类

```text
tags
  id
  created_at
  updated_at

tag_translations
  id
  tag_id
  locale
  name
  slug

article_categories
  article_id
  category_id
  is_primary

article_tags
  article_id
  tag_id
```

分类用于稳定栏目、导航和面包屑；标签用于扁平、细粒度的内容关联。文章可关联多个分类，但每种文章只允许一个主分类，用于面包屑和 canonical 语义。

### 4.4 媒体与版本

```text
media_assets
  id
  uploader_user_id
  object_key
  original_filename
  mime_type
  size_bytes
  width
  height
  created_at

article_revisions
  id
  article_translation_id
  title
  summary
  content
  revision_number
  created_by_user_id
  created_at

url_redirects
  id
  locale
  source_path
  target_path
  status_code
  created_at
```

媒体对象由对象存储适配器处理；上传后可通过异步事件生成缩略图或提取元数据。文章 slug 变更时记录 `url_redirects`，由前端或边缘层对浏览器请求返回 `301`；后端公共 API 仍遵守 HTTP 200 响应约定。

## 5. URL 与 API

浏览器侧的稳定文章 URL：

```text
/{locale}/articles/{slug}

例如：
/zh-CN/articles/go-cms-design
/en-US/articles/go-cms-design
```

文章 URL 不包含分类路径。文章移动分类不应改变永久 URL，以避免 SEO 死链和重定向维护成本。

分类 URL：

```text
/{locale}/categories/{slug}
```

公共 API：

```text
GET /api/v1/public/{locale}/articles
GET /api/v1/public/{locale}/articles/{slug}
GET /api/v1/public/{locale}/categories
GET /api/v1/public/{locale}/categories/{slug}/articles
GET /api/v1/public/{locale}/tags/{slug}/articles
```

后台 API：

```text
/api/v1/admin/cms/locales
/api/v1/admin/cms/articles
/api/v1/admin/cms/categories
/api/v1/admin/cms/tags
/api/v1/admin/cms/media
```

后台 API 支持草稿编辑、翻译编辑、发布、下线、分类排序、媒体管理和版本查看。

## 6. 后台权限与审计

在既有 RBAC 基础上新增最小权限集合：

```text
cms.article.create
cms.article.read
cms.article.update
cms.article.publish
cms.article.archive
cms.category.manage
cms.tag.manage
cms.media.upload
cms.media.delete
cms.locale.manage
```

发布、下线、删除、恢复、slug 变更、分类移动、媒体删除和权限变更必须写入审计日志。

## 7. SEO 与公开内容策略

每个文章和分类翻译维护独立 SEO title、description 和 canonical URL。公共 API 还应提供：

- 当前语言及可用翻译语言列表，用于前端生成 `hreflang`。
- 主分类和完整分类祖先链，用于面包屑。
- 发布时间、更新时间、封面媒体及其本地化 alt 文本。
- 分页信息，用于分类、标签和文章列表页。

后续增加：

- 按语言生成 XML Sitemap。
- `robots.txt` 配置。
- Article、BreadcrumbList 等 JSON-LD 所需字段。
- 内容缓存、缓存失效和 CDN 刷新事件。

## 8. 异步与缓存

文章翻译发布、下线或删除后，事务内写入 Outbox。建议事件：

```text
cms.article_translation.published
cms.article_translation.archived
cms.article_translation.slug_changed
cms.category.changed
cms.media.uploaded
```

消费者负责失效 Redis 内容缓存、重建 Sitemap、提交搜索索引或触发 CDN 刷新。第一版可先实现 Redis 缓存失效，搜索和 CDN 由后续项目需求决定。

缓存 key 必须包含 locale，例如：

```text
cms:article:{locale}:{slug}
cms:category-tree:{locale}
cms:category-articles:{locale}:{slug}:{page}
```

## 9. 实施阶段

### 阶段 1：内容核心

1. 创建下一版本 migration：`locales`、文章、翻译、分类、分类翻译、文章分类、标签和标签翻译表。
2. 实现 Domain Entity、Repository、Usecase 和 PostgreSQL 集成测试。
3. 实现后台文章/翻译/分类/标签管理与 RBAC 权限。
4. 实现公开文章和分类 API，仅返回已发布翻译。
5. 实现文章发布、下线和审计日志。

### 阶段 2：媒体与 SEO

1. 接入对象存储，增加媒体上传与访问控制。
2. 增加封面图、本地化 alt 文本、SEO 字段和面包屑数据。
3. 增加 slug 变更重定向记录、Sitemap 数据接口和缓存。
4. 使用 Outbox 失效公开内容缓存。

### 阶段 3：版本与异步增强

1. 增加文章修订版本、恢复版本和定时发布。
2. 为发布事件增加搜索索引、Sitemap 生成和 CDN 刷新消费者。
3. 补齐 Kafka 重试、DLQ、回放和发布失败场景测试。

## 10. 验收标准

- 每个支持语言可独立创建、编辑和发布文章翻译。
- 未发布、已归档或缺失翻译的内容不会通过公共 API 暴露。
- 分类树不会出现循环，且移动分类不会改变文章永久 URL。
- 文章和分类 slug 在同一 locale 内唯一；slug 变更可查询重定向记录。
- 后台操作经过 RBAC 校验并写审计日志。
- 所有新表通过 migration 创建，Repository 有 PostgreSQL 集成测试，Usecase 和 Handler 有对应单元或路由测试。
- 发布事件通过 Outbox 可靠投递，缓存失效可重复执行。
