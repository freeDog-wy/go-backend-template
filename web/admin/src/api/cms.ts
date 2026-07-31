import { request, write } from "./http";
import type { Article, ArticleDetail, ArticleInput, Category, Locale, MarkdownPreview, MediaList, MediaUpload, PublishPreview, Tag } from "./types";

const query = (items: Record<string, string | number | boolean | undefined>) => `?${new URLSearchParams(Object.entries(items).filter(([, v]) => v !== undefined).map(([k, v]) => [k, String(v)])).toString()}`;
export const cms = {
  locales: () => request<Locale[]>("/api/v1/admin/cms/locales"),
  createLocale: (input: Omit<Locale, "is_default">) => write<Locale>("POST", "/api/v1/admin/cms/locales", input),
  updateLocale: (code: string, input: Omit<Locale, "code">) => write<Locale>("PATCH", `/api/v1/admin/cms/locales/${encodeURIComponent(code)}`, input),
  categories: (locale: string) => request<Category[]>(`/api/v1/admin/cms/categories${query({ locale })}`),
  createCategory: (input: { locale: string; name: string; slug: string; description: string; parent_id?: number | null; sort_order: number }) => write<Category>("POST", "/api/v1/admin/cms/categories", input),
  updateCategory: (id: number, input: { is_enabled: boolean; sort_order: number }) => write<Category>("PATCH", `/api/v1/admin/cms/categories/${id}`, input),
  renameCategory: (id: number, locale: string, input: { name: string }) => write<Category>("PATCH", `/api/v1/admin/cms/categories/${id}/translations/${encodeURIComponent(locale)}`, input),
  moveCategory: (id: number, input: { parent_id: number | null; sort_order: number }) => write<{ id: number }>("PATCH", `/api/v1/admin/cms/categories/${id}/move`, input),
  deleteCategory: (id: number) => write<{ id: number }>("DELETE", `/api/v1/admin/cms/categories/${id}`),
  tags: (locale: string, page = 1) => request<Tag[]>(`/api/v1/admin/cms/tags${query({ locale, page, per_page: 100 })}`),
  createTag: (input: { locale: string; name: string; slug: string }) => write<Tag>("POST", "/api/v1/admin/cms/tags", input),
  updateTag: (id: number, input: { is_enabled: boolean }) => write<{ id: number; is_enabled: boolean }>("PATCH", `/api/v1/admin/cms/tags/${id}`, input),
  renameTag: (id: number, locale: string, input: { name: string }) => write<Tag>("PATCH", `/api/v1/admin/cms/tags/${id}/translations/${encodeURIComponent(locale)}`, input),
  articles: (locale: string, status?: string, page = 1, options: { includeDeleted?: boolean; deletedOnly?: boolean } = {}) => {
    const path = `/api/v1/admin/cms/articles${query({ locale, status, page, per_page: 20, include_deleted: options.includeDeleted, deleted_only: options.deletedOnly })}`;
    return request<Article[]>(path);
  },
  article: (id: number, locale: string) => request<ArticleDetail>(`/api/v1/admin/cms/articles/${id}/translations/${encodeURIComponent(locale)}`),
  previewMarkdown: (content: string) => request<MarkdownPreview>("/api/v1/admin/cms/markdown/preview", { method: "POST", body: { content } }),
  previewPublish: (id: number, locale: string) => request<PublishPreview>(`/api/v1/admin/cms/articles/${id}/translations/${encodeURIComponent(locale)}/publish-preview`),
  createArticle: (input: ArticleInput) => write<Article>("POST", "/api/v1/admin/cms/articles", input),
  updateArticle: (id: number, locale: string, input: ArticleInput) => write<Article>("PUT", `/api/v1/admin/cms/articles/${id}/translations/${encodeURIComponent(locale)}`, input),
  replaceArticleCategories: (id: number, categoryIDs: number[], primaryCategoryID: number | null) => write<{ id: number; category_ids: number[]; primary_category_id: number | null }>("PUT", `/api/v1/admin/cms/articles/${id}/categories`, { category_ids: categoryIDs, primary_category_id: primaryCategoryID }),
  replaceArticleTags: (id: number, tagIDs: number[]) => write<{ id: number; tag_ids: number[] }>("PUT", `/api/v1/admin/cms/articles/${id}/tags`, { tag_ids: tagIDs }),
  publish: (id: number, locale: string) => write<Article>("POST", `/api/v1/admin/cms/articles/${id}/translations/${encodeURIComponent(locale)}/publish`),
  archive: (id: number, locale: string) => write<Article>("POST", `/api/v1/admin/cms/articles/${id}/translations/${encodeURIComponent(locale)}/archive`),
  deleteArticle: (id: number) => write<{ id: number }>("DELETE", `/api/v1/admin/cms/articles/${id}`),
  restore: (id: number) => write<{ id: number }>("POST", `/api/v1/admin/cms/articles/${id}/restore`),
  media: (page = 1) => request<MediaList>(`/api/v1/admin/cms/media${query({ page, per_page: 40 })}`),
  requestMediaUpload: (file: File) => write<MediaUpload>("POST", "/api/v1/admin/cms/media/upload-requests", { filename: file.name, content_type: file.type, size_bytes: file.size }),
  uploadMediaObject: async (upload: MediaUpload, file: File) => {
    const response = await fetch(upload.upload_url, { method: "PUT", headers: upload.headers, body: file });
    if (!response.ok) throw new Error(`Object upload failed with HTTP ${response.status}`);
  },
  completeMediaUpload: (id: number) => write<{ id: number; status: string }>("POST", `/api/v1/admin/cms/media/${id}/complete`),
};
