import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Fragment, useEffect, useState } from "react";
import { AlertTriangle, ChevronDown, ChevronRight, Folder, FolderOpen, Plus } from "lucide-react";
import { useForm } from "react-hook-form";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { ApiError } from "../api/http";
import { cms } from "../api/cms";
import type { Category, Locale } from "../api/types";
import { useAuth } from "../app/auth";

const message = (error: unknown) => error instanceof ApiError ? `${error.code}: ${error.message}` : "Request failed";

export function LoginPage() {
  const { signIn } = useAuth(); const navigate = useNavigate(); const form = useForm({ defaultValues: { email: "", password: "" } });
  const mutation = useMutation({ mutationFn: ({ email, password }: { email: string; password: string }) => signIn(email, password), onSuccess: () => navigate("/articles") });
  return <main className="login-page"><form className="login-form" onSubmit={form.handleSubmit((values) => mutation.mutate(values))}><h1>Elseif CMS</h1><p>Administrator access</p><label>Email<input type="email" autoComplete="email" required {...form.register("email")} /></label><label>Password<input type="password" autoComplete="current-password" required minLength={6} {...form.register("password")} /></label>{mutation.error && <p className="error" role="alert">{message(mutation.error)}</p>}<button disabled={mutation.isPending}>{mutation.isPending ? "Signing in..." : "Sign in"}</button></form></main>;
}

function LocaleSelect({ value, onChange }: { value: string; onChange(value: string): void }) { const query = useQuery({ queryKey: ["locales"], queryFn: cms.locales }); return <select value={value} onChange={(e) => onChange(e.target.value)}>{query.data?.filter((item) => item.is_enabled).map((item) => <option key={item.code} value={item.code}>{item.name}</option>)}</select>; }
export function ArticlesPage() {
  const [params, setParams] = useSearchParams(); const locale = params.get("locale") ?? ""; const status = params.get("status") ?? "";
  const locales = useQuery({ queryKey: ["locales"], queryFn: cms.locales }); const selected = locale || locales.data?.find((item) => item.is_default)?.code || "";
  const articles = useQuery({ queryKey: ["articles", selected, status], queryFn: () => cms.articles(selected, status || undefined), enabled: Boolean(selected) });
  return <section className="page"><div className="page-heading"><div><h1>Articles</h1><p>Drafts and published translations.</p></div><Link className="button" to="/articles/new"><Plus size={16} />New article</Link></div><div className="filters"><LocaleSelect value={selected} onChange={(value) => setParams({ locale: value, status })} /><select value={status} onChange={(e) => setParams({ locale: selected, status: e.target.value })}><option value="">All statuses</option><option value="draft">Draft</option><option value="published">Published</option><option value="archived">Archived</option></select></div>{articles.error && <ErrorNotice error={articles.error} />}{articles.isLoading ? <p>Loading articles...</p> : <table><thead><tr><th>Title</th><th>Locale</th><th>Status</th><th>Published</th></tr></thead><tbody>{articles.data?.map((item) => <tr key={`${item.id}-${item.locale}`}><td><Link to={`/articles/${item.id}/${item.locale}`}>{item.title}</Link></td><td>{item.locale}</td><td><span className={`status ${item.status}`}>{item.status}</span></td><td>{item.published_at ? new Date(item.published_at).toLocaleDateString() : "-"}</td></tr>)}</tbody></table>}</section>;
}

type CategoryForm = { name: string; slug: string; description: string; parent_id: string };
type CategoryOption = { category: Category; depth: number };

function flattenCategories(categories: Category[], depth = 0): CategoryOption[] {
  return categories.flatMap((category) => [{ category, depth }, ...flattenCategories(category.children, depth + 1)]);
}

function CategoryTreeRows({ categories, expanded, onToggle, depth = 0 }: { categories: Category[]; expanded: Set<number>; onToggle(id: number): void; depth?: number }) {
  return <>{categories.map((category) => {
    const hasChildren = category.children.length > 0;
    const isExpanded = expanded.has(category.id);
    return <Fragment key={category.id}>
      <tr>
        <td>
          <div className="category-tree-item" style={{ paddingInlineStart: `${depth * 22}px` }}>
            {hasChildren ? <button className="category-tree-toggle" type="button" aria-expanded={isExpanded} aria-label={`${isExpanded ? "Collapse" : "Expand"} ${category.name}`} title={isExpanded ? "Collapse" : "Expand"} onClick={() => onToggle(category.id)}>{isExpanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}</button> : <span className="category-tree-spacer" aria-hidden="true" />}
            {hasChildren && isExpanded ? <FolderOpen className="category-tree-icon" size={16} aria-hidden="true" /> : <Folder className="category-tree-icon" size={16} aria-hidden="true" />}
            <span>{category.name}</span>
          </div>
        </td>
        <td>{category.slug}</td>
        <td>{category.sort_order}</td>
      </tr>
      {hasChildren && isExpanded && <CategoryTreeRows categories={category.children} expanded={expanded} onToggle={onToggle} depth={depth + 1} />}
    </Fragment>;
  })}</>;
}

export function CategoriesPage() {
  const locales = useQuery({ queryKey: ["locales"], queryFn: cms.locales });
  const [params, setParams] = useSearchParams();
  const selected = params.get("locale") || locales.data?.find((item) => item.is_default)?.code || "";
  const categories = useQuery({ queryKey: ["categories", selected], queryFn: () => cms.categories(selected), enabled: Boolean(selected) });
  const options = flattenCategories(categories.data ?? []);
  const categoryIDs = options.map(({ category }) => category.id).join(",");
  const [expanded, setExpanded] = useState<Set<number>>(new Set());
  const form = useForm<CategoryForm>({ defaultValues: { name: "", slug: "", description: "", parent_id: "" } });
  const client = useQueryClient();
  const create = useMutation({
    mutationFn: (value: CategoryForm) => cms.createCategory({ ...value, locale: selected, parent_id: value.parent_id ? Number(value.parent_id) : null, sort_order: 0 }),
    onSuccess: () => { form.reset(); client.invalidateQueries({ queryKey: ["categories", selected] }); },
  });

  useEffect(() => { setExpanded(new Set(options.map(({ category }) => category.id))); }, [selected, categoryIDs]);
  const toggleCategory = (id: number) => setExpanded((current) => {
    const next = new Set(current);
    if (next.has(id)) next.delete(id); else next.add(id);
    return next;
  });

  return <section className="page"><PageTitle title="Categories" description="Maintain localized taxonomy." /><div className="filters"><LocaleSelect value={selected} onChange={(value) => setParams({ locale: value })} /></div><form className="inline-form category-create-form" onSubmit={form.handleSubmit((value) => create.mutate(value))}><input placeholder="Name" required {...form.register("name")} /><input placeholder="slug" required {...form.register("slug")} /><input placeholder="Description" {...form.register("description")} /><select aria-label="Parent category" {...form.register("parent_id")}><option value="">Root category</option>{options.map(({ category, depth }) => <option key={category.id} value={category.id}>{`${"-- ".repeat(depth)}${category.name}`}</option>)}</select><button disabled={create.isPending}><Plus size={16} />Add</button></form>{create.error && <ErrorNotice error={create.error} />}{categories.error && <ErrorNotice error={categories.error} />}{categories.isLoading ? <p>Loading categories...</p> : <table><thead><tr><th>Name</th><th>Slug</th><th>Order</th></tr></thead><tbody><CategoryTreeRows categories={categories.data ?? []} expanded={expanded} onToggle={toggleCategory} /></tbody></table>}</section>;
}
export function TagsPage() {
  const locales = useQuery({ queryKey: ["locales"], queryFn: cms.locales }); const [params, setParams] = useSearchParams(); const selected = params.get("locale") || locales.data?.find((item) => item.is_default)?.code || "";
  const tags = useQuery({ queryKey: ["tags", selected], queryFn: () => cms.tags(selected), enabled: Boolean(selected) }); const form = useForm({ defaultValues: { name: "", slug: "" } }); const client = useQueryClient(); const create = useMutation({ mutationFn: (v: { name: string; slug: string }) => cms.createTag({ ...v, locale: selected }), onSuccess: () => { form.reset(); client.invalidateQueries({ queryKey: ["tags", selected] }); } });
  return <section className="page"><PageTitle title="Tags" description="Maintain localized tags." /><div className="filters"><LocaleSelect value={selected} onChange={(value) => setParams({ locale: value })} /></div><form className="inline-form" onSubmit={form.handleSubmit((v) => create.mutate(v))}><input placeholder="Name" required {...form.register("name")} /><input placeholder="slug" required {...form.register("slug")} /><button disabled={create.isPending}><Plus size={16} />Add</button></form>{create.error && <ErrorNotice error={create.error} />}<table><thead><tr><th>Name</th><th>Slug</th></tr></thead><tbody>{tags.data?.map((item) => <tr key={item.id}><td>{item.name}</td><td>{item.slug}</td></tr>)}</tbody></table></section>;
}
function PageTitle({ title, description }: { title: string; description: string }) { return <div className="page-heading"><div><h1>{title}</h1><p>{description}</p></div></div>; }

export function LocalesPage() { const query = useQuery({ queryKey: ["locales"], queryFn: cms.locales }); const form = useForm({ defaultValues: { code: "", name: "", sort_order: 0, is_enabled: true } }); const client = useQueryClient(); const create = useMutation({ mutationFn: (value: { code: string; name: string; sort_order: number; is_enabled: boolean }) => cms.createLocale(value), onSuccess: () => { form.reset(); client.invalidateQueries({ queryKey: ["locales"] }); } }); return <section className="page"><div className="page-heading"><div><h1>Locales</h1><p>Enabled locales are available for CMS translations.</p></div></div><form className="inline-form" onSubmit={form.handleSubmit((v) => create.mutate(v))}><input placeholder="en-US" required {...form.register("code")} /><input placeholder="English" required {...form.register("name")} /><input type="number" aria-label="Sort order" {...form.register("sort_order", { valueAsNumber: true })} /><label className="checkbox"><input type="checkbox" {...form.register("is_enabled")} />Enabled</label><button disabled={create.isPending}><Plus size={16} />Add</button></form>{create.error && <ErrorNotice error={create.error} />}<table><thead><tr><th>Code</th><th>Name</th><th>Enabled</th><th>Default</th></tr></thead><tbody>{query.data?.map((item: Locale) => <tr key={item.code}><td>{item.code}</td><td>{item.name}</td><td>{item.is_enabled ? "Yes" : "No"}</td><td>{item.is_default ? "Yes" : ""}</td></tr>)}</tbody></table></section>; }
function ErrorNotice({ error }: { error: unknown }) { return <p className="error" role="alert"><AlertTriangle size={16} />{message(error)}</p>; }
