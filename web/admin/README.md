# Elseif CMS Admin

React 管理端的静态构建项目，部署目标为 `admin.elseif.site`。

## Local development

1. 保持后端在 `http://localhost:8080` 运行。
2. 确认后端配置中的 `server.cors_allowed_origins` 与 `auth.admin_origin` 均包含 `http://localhost:5173`。
3. 运行：

```bash
npm ci
npm run dev
```

本地运行时配置在 `public/runtime-config.js`。它只能包含公开的 API 基地址，不能放入 access token、refresh token、用户凭据或任何密钥。

## Quality checks

```bash
npm run typecheck
npm run lint
npm run test
npm run build
```

`dist/` 是唯一的部署产物。生产环境将 `public/runtime-config.js` 替换为 `https://api.elseif.site` 的配置，并通过静态托管层设置 CSP、缓存和 HTTPS。
