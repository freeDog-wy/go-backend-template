# React 静态后台管理站方案

## 1. 决策与目标

本方案在当前后端仓库中新增一个独立的 React 管理端，源码位于 `web/admin/`，构建后以纯静态文件部署到 `https://admin.elseif.site`。管理端通过浏览器调用 `https://api.elseif.site`，不增加 Go 服务端渲染、BFF、SSR 或 Node.js 常驻服务。

这里的“静态后台”是指交付和托管方式为静态资源；文章、分类、标签、语言、登录态及权限始终由 API 动态提供。它不是 `internal/app/sitegen` 的一部分，也不使用 sitegen 的构建流程。

目标：

- 仅面向内容管理员，优先保证操作正确性、安全性和可恢复性。
- 界面简洁、紧凑、低干扰；不按营销站或复杂 SaaS 仪表盘设计。
- 通过 React 管理多页面状态、表单、异步请求和动态路由，避免手写 DOM 更新带来的状态错误。
- 延用后端已有 Bearer access token、HttpOnly refresh cookie、RBAC、CORS 白名单和幂等键能力。
- 最终部署产物可由 Nginx、对象存储 CDN 或静态托管服务直接托管。

第一版不实现：

- SSR、服务端渲染、Next.js、服务端 API 代理或 BFF。
- 面向匿名访问者的页面、SEO、公开内容展示或搜索。
- 离线缓存、Service Worker、乐观更新、多人协作、实时推送和复杂数据可视化。
- 直接访问数据库、Redis、Kafka、对象存储或 Go 内部 Usecase。

## 2. 代码边界与目录

React 项目不放入 `internal/app/`。该目录是 Go 内部应用实现的边界，存放 Node/Vite 项目会混淆语言运行时和构建职责。

```text
go-backend-template/
  cmd/
  configs/
  doc/
  internal/
    app/
      sitegen/                 # Go 静态公开内容站
      pkg/cmsclient/           # Go 调用 CMS API 的 SDK
  web/
    admin/                     # 本方案的 React 项目
      package.json
      package-lock.json
      vite.config.ts
      tsconfig.json
      index.html
      public/
        runtime-config.js
      src/
        main.tsx
        app/
          router.tsx
          layout.tsx
          providers.tsx
        api/
          http.ts
          auth.ts
          cms.ts
        features/
          auth/
          articles/
          categories/
          tags/
          locales/
        components/
          ui/
          layout/
        styles/
          tokens.css
          global.css
          admin.css
      tests/
```

`node_modules/`、`dist/`、本地 `.env` 不提交。`package-lock.json` 必须提交，以锁定可复现依赖。后端 Go 构建不进入 `web/admin/`；前端 CI 在该目录独立执行。

虽然浏览器最终运行 JavaScript，源码使用 TypeScript。它没有运行时成本，但可校验 CMS 表单、路由参数和 API 响应，适合以可靠性优先的管理端。

## 3. 技术决策

| 范围 | 决策 | 原因 |
| --- | --- | --- |
| 构建 | Vite | 快速开发与纯静态产物，部署不依赖 Node 服务。 |
| UI | React + TypeScript | 管理多表单、路由、会话和异步状态。 |
| 路由 | React Router Hash Router | 使用 `/#/articles` 等路径，不要求静态服务器配置 SPA fallback。 |
| 服务端数据 | TanStack Query | 统一加载、失效重取、错误和请求取消；不使用自定义全局缓存。 |
| 表单 | React Hook Form + Zod | 文章与翻译表单的校验、脏状态、提交状态可预测。 |
| 图标 | lucide-react | 使用成熟可访问图标，不手写 SVG。 |
| 样式 | 本地 CSS + CSS Variables | 保持 UI 克制，避免大型 UI 库、CDN 可用性和主题覆盖复杂度。 |
| 状态管理 | React Context 仅管理会话和主题 | 第一版不使用 Redux；服务端状态由 TanStack Query 管理。 |

不得在生产环境通过 CDN 加载 React、路由或编辑器依赖。Vite 将依赖打包为本站带哈希的静态资源，以便施加 CSP、缓存和版本控制。

## 4. 运行时配置与部署

构建时不写入密钥。API 基地址是公开配置，使用运行时文件支持同一前端产物在不同环境部署：

```js
// public/runtime-config.js
window.__ADMIN_CONFIG__ = {
  apiBaseURL: "https://api.elseif.site",
};
```

`index.html` 在模块入口前加载此文件。应用启动时必须校验它是无查询参数、无片段的绝对 HTTPS URL；开发环境可允许明确配置的 `http://localhost`。

发布流程：

```text
web/admin/
  npm ci
  npm run typecheck
  npm run test
  npm run build
       |
       v
  dist/ -> admin.elseif.site
```

静态托管配置：

- `index.html` 返回 `Cache-Control: no-cache`，防止用户长期使用旧入口文件。
- Vite 带哈希的 JS、CSS 和图片资源使用长期不可变缓存。
- 不注册 Service Worker，避免后台操作使用陈旧资源或缓存响应。
- Hash Router 不需要 Nginx `try_files` SPA 回退规则。
- 生产站必须启用 HTTPS，并将 HTTP 重定向到 HTTPS。

建议静态站响应头：

```text
Content-Security-Policy:
  default-src 'self';
  script-src 'self';
  style-src 'self';
  connect-src 'self' https://api.elseif.site;
  img-src 'self' https: data:;
  object-src 'none';
  base-uri 'self';
  frame-ancestors 'none'
X-Content-Type-Options: nosniff
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: camera=(), microphone=(), geolocation=()
```

如 Markdown 预览引入本地图片或受控第三方域名，必须显式补充 `img-src`，不能为了兼容而放宽为 `*`。

## 5. 路由与首期功能

首期采用以下 Hash 路由。未登录访问任一受保护路由时，应用先尝试恢复会话，失败后跳转至 `/#/login`。

| 路由 | 功能 |
| --- | --- |
| `/#/login` | 管理员登录。 |
| `/#/` | 简洁概览：会话状态、文章数量和最近操作入口；不引入图表。 |
| `/#/articles` | 按语言、状态筛选的文章列表和分页。 |
| `/#/articles/new` | 创建文章草稿。 |
| `/#/articles/:articleID/:locale` | 编辑文章翻译、Markdown、SEO、分类、标签、封面和发布状态。 |
| `/#/categories` | 分类创建、状态、排序、移动和翻译。 |
| `/#/tags` | 标签创建和翻译。 |
| `/#/locales` | 语言创建、排序、启用、默认语言设置。 |
| `/#/status` | API 连通性与当前会话诊断。 |

文章编辑必须提供：

- 当前语言及已有翻译版本切换；不可把同一文章的不同语言内容混为一个表单。
- 草稿保存、发布预览、发布、归档、恢复等明确操作；破坏性或影响公开可见性的操作需要确认对话框。
- Markdown 使用 `textarea` 编辑。首期可不提供富文本编辑器，避免编辑器状态、XSS 和依赖复杂度。
- 浏览器预览必须使用受限 Markdown 渲染和净化；不能将 CMS 文本直接赋给 `dangerouslySetInnerHTML`。
- 离开脏表单时询问确认；提交期间保持按钮尺寸并禁用重复提交。

媒体上传或媒体库仅在后端已有稳定管理 API 后加入。不得在前端虚构尚不存在的工作流。

## 6. UI 规范

管理端是个人内容操作工具。视觉和交互应安静、密集、可扫描，而不是卡片化营销页面。

- 桌面端使用固定窄侧栏、顶部会话操作区和单一主工作区；移动端侧栏折叠为抽屉。
- 文章、分类、标签使用紧凑表格或列表；表单使用连续区块和清晰分隔线，避免卡片嵌套。
- 主操作使用文本加图标按钮；撤销、刷新、关闭和主题等工具操作使用带 tooltip 的图标按钮。
- 危险操作使用明确的红色语义，不仅靠颜色区分。成功、失败和保存状态通过可访问的 `aria-live` 区域通知。
- 表格在小屏幕上优先保留标题、语言、状态和更新时间，其余字段进入详情页，避免横向文本重叠。
- 默认跟随系统浅/深色主题，用户可切换；主题偏好可保存到 `localStorage`，但认证信息不得写入任何浏览器持久化存储。
- CSS 以中性色画布、清晰文字、绿色操作强调和红色危险强调为主；不使用渐变、玻璃拟态、装饰性插画或大尺寸 Hero。

设计令牌定义于 `styles/tokens.css`。颜色、间距、边框、字体和控件高度必须使用令牌，组件不得散落硬编码值。

## 7. 浏览器 API 客户端

页面组件不得直接调用 `fetch`。所有浏览器 API 调用集中在 `src/api/`：

```text
api/http.ts
  运行时配置校验
  JSON 编解码
  credentials: "include"
  Bearer access token 注入
  401 单飞刷新、重试一次
  统一 HTTP / 业务错误
  AbortSignal 和超时

api/auth.ts
  login
  refresh
  logout

api/cms.ts
  articles / article translations
  categories / category translations
  tags / tag translations
  locales
  health
```

`http.ts` 的规则：

1. 登录、刷新和登出使用 `credentials: "include"`，从而发送并接收 API 域的 refresh cookie。
2. access token 只存在于 React 会话状态和内存变量，不写入 Cookie、`localStorage`、`sessionStorage`、URL 或日志。
3. 管理 API 请求附加 `Authorization: Bearer <access-token>`，同时保留 `credentials: "include"`。
4. 发生 401 时，仅允许一个 refresh 请求在途；其他请求等待同一结果。刷新成功后每个原请求只重试一次，失败则清空内存会话并跳转登录。
5. HTTP 非 2xx、`success: false`、网络错误、超时和 JSON 解码错误必须转换为可处理的结构化错误；页面不解析原始响应字符串。
6. 每个写操作生成一次 `crypto.randomUUID()`，作为 `X-Correlation-ID` 和 `Idempotency-Key`。用户重试同一次不确定结果时复用原键；新操作必须生成新键。
7. 列表请求使用 `AbortController`，在筛选条件或路由变化时取消不再需要的请求，避免旧响应覆盖新页面。

发布、归档、恢复、默认语言设置等请求成功后使相关 TanStack Query 缓存失效并重新读取服务端状态；第一版不使用乐观更新。

## 8. 认证、CORS 与权限

管理端使用现有 API 认证机制，而不是新增一套认证方案：

```text
admin.elseif.site                     api.elseif.site
       |                                      |
       | POST /api/v1/admin/auth/login        |
       | credentials: include                 |
       |------------------------------------->|
       |  access token 响应 + HttpOnly        |
       |  refresh cookie (Set-Cookie)         |
       |<-------------------------------------|
       |
       | 管理 API: Authorization Bearer       |
       | refresh / logout: refresh cookie     |
       |------------------------------------->|
```

生产配置必须使用精确 Origin，不可使用 `*`：

```yaml
server:
  cors_allowed_origins:
    - "https://admin.elseif.site"

auth:
  admin_origin: "https://admin.elseif.site"
  admin_refresh_cookie_name: "admin_refresh_token"
  admin_refresh_cookie_secure: true
```

开发环境可使用 `http://localhost:<port>`，但该 Origin 必须同时出现在 `server.cors_allowed_origins` 和 `auth.admin_origin`。现有配置校验要求 `admin_origin` 包含在 CORS 白名单中，不应移除该约束。

CORS 中间件必须：

- 仅对精确白名单 Origin 返回 `Access-Control-Allow-Origin`。
- 响应 `Access-Control-Allow-Credentials: true`；启用凭据时不能返回通配 Origin。
- 允许 `Authorization`、`Content-Type`、`X-Correlation-ID`、`Idempotency-Key` 等所需请求头。
- 正确处理 `OPTIONS` 预检请求。

CORS 不是 CSRF 防护。Cookie 参与 refresh 和 logout；对 Cookie 可认证的状态变更接口，后端应在后续安全加固中验证 `Origin`，或采用明确的 CSRF token 机制。管理 API 的最终授权必须继续由后端 Bearer JWT 和 RBAC 判断，前端路由守卫只用于体验。

## 9. CMS API 契约

第一版使用当前已有认证和管理端点，浏览器客户端的路径和请求字段必须与后端保持一致：

```text
POST  /api/v1/admin/auth/login
POST  /api/v1/auth/refresh
POST  /api/v1/auth/logout

GET   /healthz
GET   /readyz

GET   /api/v1/admin/cms/locales
POST  /api/v1/admin/cms/locales
PATCH /api/v1/admin/cms/locales/:code

GET   /api/v1/admin/cms/articles?locale=&status=&page=&per_page=
GET   /api/v1/admin/cms/articles/:id/translations/:locale
POST  /api/v1/admin/cms/articles
POST  /api/v1/admin/cms/articles/:id/translations
PUT   /api/v1/admin/cms/articles/:id/translations/:locale
PUT   /api/v1/admin/cms/articles/:id/categories
PUT   /api/v1/admin/cms/articles/:id/tags
GET   /api/v1/admin/cms/articles/:id/translations/:locale/publish-preview
POST  /api/v1/admin/cms/articles/:id/translations/:locale/publish
POST  /api/v1/admin/cms/articles/:id/translations/:locale/archive
POST  /api/v1/admin/cms/articles/:id/restore
PUT   /api/v1/admin/cms/articles/:id/cover

GET   /api/v1/admin/cms/categories?locale=
POST  /api/v1/admin/cms/categories
PATCH /api/v1/admin/cms/categories/:id
PATCH /api/v1/admin/cms/categories/:id/move
PUT   /api/v1/admin/cms/categories/:id/translations/:locale

GET   /api/v1/admin/cms/tags?locale=&page=&per_page=
POST  /api/v1/admin/cms/tags
PUT   /api/v1/admin/cms/tags/:id/translations/:locale
```

Go 的 `internal/app/pkg/cmsclient` 不能在浏览器中直接复用。它仍是 Go 调用方的 SDK；React 需要自己的 `api/cms.ts`。为防止两套客户端漂移，后续应将 OpenAPI 规范确立为跨语言唯一 API 契约：

1. 后端新增或修改接口时，先同步更新 OpenAPI。
2. Go `cmsclient` 与浏览器 TypeScript 类型以该规范校验或生成。
3. CI 校验规范、后端路由和两个客户端的契约测试。

在 OpenAPI 引入前，任何管理 API 变更必须同时修改后端 handler、Go `cmsclient`、React `api/cms.ts` 和相关测试，不能只修改任一客户端。

## 10. 测试与质量门禁

前端质量门禁：

```text
npm run typecheck
npm run lint
npm run test
npm run build
```

测试分层：

- 单元测试：运行时配置、HTTP 错误映射、401 单飞刷新、幂等键、Zod 校验和路由参数。
- 组件测试：登录失败、筛选分页、脏表单离开提示、提交禁用、发布确认、403 与会话过期状态。
- API 契约测试：CMS 请求方法、路径、请求体、请求头和响应信封与后端一致。
- Playwright 端到端测试：登录、刷新后恢复会话、创建草稿、编辑翻译、发布、归档、退出；同时覆盖 CORS 预检与 Cookie 会话。
- 无障碍检查：键盘导航、焦点可见、表单 label、错误文本关联、对话框焦点陷阱和 `aria-live` 状态通知。

CI 必须使用 `npm ci` 而非无锁安装。依赖升级需经过 lockfile 变更审查与测试，不自动接受不受控的大版本升级。

## 11. 实施阶段与验收

### 阶段一：项目壳和会话

- 初始化 Vite、React、TypeScript、Hash Router 和本地样式令牌。
- 实现运行时配置、登录、刷新、退出、路由守卫和统一 HTTP 客户端。
- 实现浅深色主题、基础布局、全局错误边界和状态页。

验收：刷新页面能够恢复登录；access token 不出现在浏览器持久化存储；CORS、Cookie 和 Bearer 请求可在本地开发环境联通。

### 阶段二：文章工作流

- 实现文章列表、语言和状态筛选、分页、创建草稿和翻译编辑。
- 实现 Markdown、SEO、分类、标签、发布预览、发布、归档和恢复。
- 实现脏表单提示、确认对话框、幂等写请求和服务器状态重取。

验收：同一操作重复点击不会产生重复写入；发布失败保留编辑内容；401、403、超时和业务错误均有明确可恢复反馈。

### 阶段三：分类、标签和语言

- 实现分类树、移动、启用状态和翻译。
- 实现标签和翻译管理。
- 实现语言创建、启用、排序和默认语言设置。

验收：各操作仅通过现有管理 API 执行；无权限时后端拒绝且前端不伪造成功状态。

### 阶段四：交付与安全加固

- 配置生产域名、HTTPS、CSP、缓存头、CORS 白名单与安全 Cookie。
- 补齐 Playwright、API 契约测试和 CI 构建产物发布。
- 规划并引入 OpenAPI 作为后端、Go SDK 和浏览器客户端的契约来源。

验收：生产构建不含密钥；静态产物可独立部署；核心登录和文章发布流程经过端到端测试；后端、Go SDK 与 React API 层的接口变更有明确维护流程。
