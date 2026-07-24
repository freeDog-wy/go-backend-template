import { request, write } from "./http";
import type { Article, ArticleDetail, ArticleInput, Category, Locale, Tag } from "./types";

const query = (items: Record<string, string | number | boolean | undefined>) => `?${new URLSearchParams(Object.entries(items).filter(([, v]) => v !== undefined).map(([k, v]) => [k, String(v)])).toString()}`;
export const cms = {
  locales: () => request<Locale[]>("/api/v1/admin/cms/locales"),
  createLocale: (input: Omit<Locale, "is_default">) => write<Locale>("POST", "/api/v1/admin/cms/locales", input),
  updateLocale: (code: string, input: Omit<Locale, "code">) => write<Locale>("PATCH", `/api/v1/admin/cms/locales/${encodeURIComponent(code)}`, input),
  categories: (locale: string) => request<Category[]>(`/api/v1/admin/cms/categories${query({ locale })}`),
  createCategory: (input: { locale: string; name: string; slug: string; description: string; parent_id?: number | null; sort_order: number }) => write<Category>("POST", "/api/v1/admin/cms/categories", input),
  updateCategory: (id: number, input: { is_enabled: boolean; sort_order: number }) => write<Category>("PATCH", `/api/v1/admin/cms/categories/${id}`, input),
  tags: (locale: string, page = 1) => request<Tag[]>(`/api/v1/admin/cms/tags${query({ locale, page, per_page: 100 })}`),
  createTag: (input: { locale: string; name: string; slug: string }) => write<Tag>("POST", "/api/v1/admin/cms/tags", input),
  articles: (locale: string, status?: string, page = 1) => {
    const path = `/api/v1/admin/cms/articles${query({ locale, status, page, per_page: 20 })}`;
    return request<Article[]>(path);
  },
  article: (id: number, locale: string) => request<ArticleDetail>(`/api/v1/admin/cms/articles/${id}/translations/${encodeURIComponent(locale)}`),
  createArticle: (input: ArticleInput) => write<Article>("POST", "/api/v1/admin/cms/articles", input),
  updateArticle: (id: number, locale: string, input: ArticleInput) => write<Article>("PUT", `/api/v1/admin/cms/articles/${id}/translations/${encodeURIComponent(locale)}`, input),
  publish: (id: number, locale: string) => write<Article>("POST", `/api/v1/admin/cms/articles/${id}/translations/${encodeURIComponent(locale)}/publish`),
  archive: (id: number, locale: string) => write<Article>("POST", `/api/v1/admin/cms/articles/${id}/translations/${encodeURIComponent(locale)}/archive`),
  restore: (id: number) => write<Article>("POST", `/api/v1/admin/cms/articles/${id}/restore`),
};
