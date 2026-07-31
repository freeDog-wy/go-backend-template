import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Fragment, useEffect, useState } from "react";
import { AlertTriangle, Archive, ChevronDown, ChevronRight, Folder, FolderInput, FolderOpen, Pencil, Plus, RotateCcw, Star, Trash2, X } from "lucide-react";
import { useForm } from "react-hook-form";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { ApiError } from "../api/http";
import { cms } from "../api/cms";
import type { Article, Category, Locale, Tag } from "../api/types";
import { useAuth } from "../app/auth";

const message = (error: unknown) => error instanceof ApiError ? `${error.code}: ${error.message}` : "Request failed";

export function LoginPage() {
  const { signIn } = useAuth(); const navigate = useNavigate(); const form = useForm({ defaultValues: { email: "", password: "" } });
  const mutation = useMutation({ mutationFn: ({ email, password }: { email: string; password: string }) => signIn(email, password), onSuccess: () => navigate("/articles") });
  return <main className="login-page"><form className="login-form" onSubmit={form.handleSubmit((values) => mutation.mutate(values))}><h1>Elseif CMS</h1><p>Administrator access</p><label>Email<input type="email" autoComplete="email" required {...form.register("email")} /></label><label>Password<input type="password" autoComplete="current-password" required minLength={6} {...form.register("password")} /></label>{mutation.error && <p className="error" role="alert">{message(mutation.error)}</p>}<button disabled={mutation.isPending}>{mutation.isPending ? "Signing in..." : "Sign in"}</button></form></main>;
}

function LocaleSelect({ value, onChange }: { value: string; onChange(value: string): void }) { const query = useQuery({ queryKey: ["locales"], queryFn: cms.locales }); return <select value={value} onChange={(e) => onChange(e.target.value)}>{query.data?.filter((item) => item.is_enabled).map((item) => <option key={item.code} value={item.code}>{item.name}</option>)}</select>; }
export function ArticlesPage() {
  const [params, setParams] = useSearchParams();
  const locale = params.get("locale") ?? "";
  const status = params.get("status") ?? "";
  const lifecycle = params.get("lifecycle") ?? "active";
  const isDeletedView = lifecycle === "deleted";
  const [action, setAction] = useState<ArticleAction>(null);
  const locales = useQuery({ queryKey: ["locales"], queryFn: cms.locales });
  const selected = locale || locales.data?.find((item) => item.is_default)?.code || "";
  const articles = useQuery({ queryKey: ["articles", selected, status, lifecycle], queryFn: () => cms.articles(selected, status || undefined, 1, { deletedOnly: isDeletedView || undefined }), enabled: Boolean(selected) });
  const client = useQueryClient();
  const onActionSuccess = () => { setAction(null); client.invalidateQueries({ queryKey: ["articles"] }); client.invalidateQueries({ queryKey: ["article"] }); };
  const archive = useMutation({ mutationFn: (article: Article) => cms.archive(article.id, article.locale), onSuccess: onActionSuccess });
  const remove = useMutation({ mutationFn: (article: Article) => cms.deleteArticle(article.id), onSuccess: onActionSuccess });
  const restore = useMutation({ mutationFn: (article: Article) => cms.restore(article.id), onSuccess: onActionSuccess });
  const pending = archive.isPending || remove.isPending || restore.isPending;
  const actionError = action?.kind === "archive" ? archive.error : action?.kind === "delete" ? remove.error : restore.error;
  const setFilters = (next: { locale?: string; status?: string; lifecycle?: string }) => setParams({ locale: next.locale ?? selected, status: next.status ?? status, lifecycle: next.lifecycle ?? lifecycle });
  const confirmAction = () => { if (!action) return; if (action.kind === "archive") archive.mutate(action.article); else if (action.kind === "delete") remove.mutate(action.article); else restore.mutate(action.article); };
  return <section className="page"><div className="page-heading"><div><h1>Articles</h1><p>Manage translations and recoverable article lifecycle changes.</p></div><Link className="button" to="/articles/new"><Plus size={16} />New article</Link></div><div className="filters"><LocaleSelect value={selected} onChange={(value) => setFilters({ locale: value })} /><select value={status} onChange={(e) => setFilters({ status: e.target.value })}><option value="">All statuses</option><option value="draft">Draft</option><option value="published">Published</option><option value="archived">Archived</option></select><select aria-label="Article lifecycle" value={lifecycle} onChange={(e) => setFilters({ lifecycle: e.target.value })}><option value="active">Active articles</option><option value="deleted">Deleted articles</option></select></div>{articles.error && <ErrorNotice error={articles.error} />}{articles.isLoading ? <p>Loading articles...</p> : <table className="articles-table"><thead><tr><th>Title</th><th>Locale</th><th>Status</th><th>Published</th><th>Actions</th></tr></thead><tbody>{articles.data?.map((item) => <tr key={`${item.id}-${item.locale}`}><td>{isDeletedView ? item.title : <Link to={`/articles/${item.id}/${item.locale}`}>{item.title}</Link>}</td><td>{item.locale}</td><td><span className={`status ${item.status}`}>{item.status}</span></td><td>{item.published_at ? new Date(item.published_at).toLocaleDateString() : "-"}</td><td><div className="article-actions">{isDeletedView ? <button className="icon-button" type="button" title="Restore article" aria-label={`Restore ${item.title}`} disabled={pending} onClick={() => setAction({ kind: "restore", article: item })}><RotateCcw size={16} /></button> : <><button className="icon-button" type="button" title="Archive translation" aria-label={`Archive ${item.title} translation`} disabled={pending || item.status === "archived"} onClick={() => setAction({ kind: "archive", article: item })}><Archive size={16} /></button><button className="icon-button article-delete-button" type="button" title="Delete article" aria-label={`Delete ${item.title}`} disabled={pending} onClick={() => setAction({ kind: "delete", article: item })}><Trash2 size={16} /></button></>}</div></td></tr>)}</tbody></table>}{action && <ArticleActionDialog action={action} pending={pending} error={actionError} onCancel={() => setAction(null)} onConfirm={confirmAction} />}</section>;
}

type ArticleAction = { kind: "archive" | "delete" | "restore"; article: Article } | null;

function ArticleActionDialog({ action, pending, error, onCancel, onConfirm }: { action: Exclude<ArticleAction, null>; pending: boolean; error: unknown; onCancel(): void; onConfirm(): void }) {
  const copy = action.kind === "archive" ? { title: "Archive translation", description: `Archive the ${action.article.locale} translation of \"${action.article.title}\"? Other language versions remain unchanged.`, confirm: "Archive" } : action.kind === "delete" ? { title: "Delete article", description: `Soft-delete \"${action.article.title}\" and all of its language versions? It can be restored later.`, confirm: "Delete" } : { title: "Restore article", description: `Restore \"${action.article.title}\" and make all of its language versions available to manage again?`, confirm: "Restore" };
  return <div className="dialog-backdrop" role="presentation"><section className="confirmation-dialog" role="dialog" aria-modal="true" aria-labelledby="article-action-title"><div className="dialog-heading"><div><h2 id="article-action-title">{copy.title}</h2><p>{copy.description}</p></div><button className="icon-button" type="button" title="Close" aria-label="Close" disabled={pending} onClick={onCancel}><X size={18} /></button></div>{Boolean(error) && <ErrorNotice error={error} />}<div className="dialog-actions"><button className="secondary" type="button" disabled={pending} onClick={onCancel}>Cancel</button><button className={action.kind === "delete" ? "danger-button" : ""} type="button" disabled={pending} onClick={onConfirm}>{pending ? "Working..." : copy.confirm}</button></div></section></div>;
}

type CategoryForm = { name: string; slug: string; description: string; parent_id: string };
type CategoryOption = { category: Category; depth: number };
type CategoryAction = { kind: "rename" | "move" | "delete"; category: Category } | null;

function flattenCategories(categories: Category[], depth = 0): CategoryOption[] {
  return categories.flatMap((category) => [{ category, depth }, ...flattenCategories(category.children, depth + 1)]);
}

function CategoryTreeRows({ categories, expanded, onToggle, onEnabledChange, onAction, updatingID, depth = 0 }: { categories: Category[]; expanded: Set<number>; onToggle(id: number): void; onEnabledChange(category: Category): void; onAction(action: Exclude<CategoryAction, null>): void; updatingID?: number; depth?: number }) {
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
        <td><label className="category-enabled-control"><input type="checkbox" role="switch" checked={category.is_enabled} disabled={updatingID === category.id} onChange={() => onEnabledChange(category)} /><span>{category.is_enabled ? "Enabled" : "Disabled"}</span></label></td>
        <td><div className="article-actions"><button className="icon-button" type="button" title="Rename category" aria-label={`Rename ${category.name}`} onClick={() => onAction({ kind: "rename", category })}><Pencil size={16} /></button><button className="icon-button" type="button" title="Move category" aria-label={`Move ${category.name}`} onClick={() => onAction({ kind: "move", category })}><FolderInput size={16} /></button><button className="icon-button article-delete-button" type="button" title="Delete category" aria-label={`Delete ${category.name}`} onClick={() => onAction({ kind: "delete", category })}><Trash2 size={16} /></button></div></td>
      </tr>
      {hasChildren && isExpanded && <CategoryTreeRows categories={category.children} expanded={expanded} onToggle={onToggle} onEnabledChange={onEnabledChange} onAction={onAction} updatingID={updatingID} depth={depth + 1} />}
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
  const [action, setAction] = useState<CategoryAction>(null);
  const form = useForm<CategoryForm>({ defaultValues: { name: "", slug: "", description: "", parent_id: "" } });
  const client = useQueryClient();
  const create = useMutation({
    mutationFn: (value: CategoryForm) => cms.createCategory({ ...value, locale: selected, parent_id: value.parent_id ? Number(value.parent_id) : null, sort_order: 0 }),
    onSuccess: () => { form.reset(); client.invalidateQueries({ queryKey: ["categories", selected] }); },
  });
  const update = useMutation({
    mutationFn: (category: Category) => cms.updateCategory(category.id, { is_enabled: !category.is_enabled, sort_order: category.sort_order }),
    onSuccess: () => client.invalidateQueries({ queryKey: ["categories", selected] }),
  });
  const rename = useMutation({ mutationFn: ({ category, name }: { category: Category; name: string }) => cms.renameCategory(category.id, selected, { name }), onSuccess: () => { setAction(null); client.invalidateQueries({ queryKey: ["categories", selected] }); } });
  const move = useMutation({ mutationFn: ({ category, parentID, sortOrder }: { category: Category; parentID: number | null; sortOrder: number }) => cms.moveCategory(category.id, { parent_id: parentID, sort_order: sortOrder }), onSuccess: () => { setAction(null); client.invalidateQueries({ queryKey: ["categories", selected] }); } });
  const remove = useMutation({ mutationFn: (category: Category) => cms.deleteCategory(category.id), onSuccess: () => { setAction(null); client.invalidateQueries({ queryKey: ["categories", selected] }); } });

  useEffect(() => { setExpanded(new Set(options.map(({ category }) => category.id))); }, [selected, categoryIDs]);
  const toggleCategory = (id: number) => setExpanded((current) => {
    const next = new Set(current);
    if (next.has(id)) next.delete(id); else next.add(id);
    return next;
  });

  const actionError = action?.kind === "rename" ? rename.error : action?.kind === "move" ? move.error : remove.error;
  const pending = rename.isPending || move.isPending || remove.isPending;
  return <section className="page"><PageTitle title="Categories" description="Maintain localized taxonomy." /><div className="filters"><LocaleSelect value={selected} onChange={(value) => setParams({ locale: value })} /></div><form className="inline-form category-create-form" onSubmit={form.handleSubmit((value) => create.mutate(value))}><input placeholder="Name" required {...form.register("name")} /><input placeholder="slug" required {...form.register("slug")} /><input placeholder="Description" {...form.register("description")} /><select aria-label="Parent category" {...form.register("parent_id")}><option value="">Root category</option>{options.map(({ category, depth }) => <option key={category.id} value={category.id}>{`${"-- ".repeat(depth)}${category.name}`}</option>)}</select><button disabled={create.isPending}><Plus size={16} />Add</button></form>{create.error && <ErrorNotice error={create.error} />}{update.error && <ErrorNotice error={update.error} />}{categories.error && <ErrorNotice error={categories.error} />}{categories.isLoading ? <p>Loading categories...</p> : <table className="categories-table"><thead><tr><th>Name</th><th>Slug</th><th>Order</th><th>Status</th><th>Actions</th></tr></thead><tbody><CategoryTreeRows categories={categories.data ?? []} expanded={expanded} onToggle={toggleCategory} onEnabledChange={(category) => update.mutate(category)} onAction={setAction} updatingID={update.isPending ? update.variables?.id : undefined} /></tbody></table>}{action && <CategoryActionDialog action={action} options={options} pending={pending} error={actionError} onCancel={() => setAction(null)} onRename={(name) => rename.mutate({ category: action.category, name })} onMove={(parentID, sortOrder) => move.mutate({ category: action.category, parentID, sortOrder })} onDelete={() => remove.mutate(action.category)} />}</section>;
}

function CategoryActionDialog({ action, options, pending, error, onCancel, onRename, onMove, onDelete }: { action: Exclude<CategoryAction, null>; options: CategoryOption[]; pending: boolean; error: unknown; onCancel(): void; onRename(name: string): void; onMove(parentID: number | null, sortOrder: number): void; onDelete(): void }) {
  const form = useForm({ defaultValues: { name: action.category.name, parent_id: action.category.parent_id?.toString() ?? "", sort_order: action.category.sort_order } });
  useEffect(() => form.reset({ name: action.category.name, parent_id: action.category.parent_id?.toString() ?? "", sort_order: action.category.sort_order }), [action, form]);
  const blockedParents = new Set(flattenCategories([action.category]).map(({ category }) => category.id));
  const title = action.kind === "rename" ? "Rename category" : action.kind === "move" ? "Move category" : "Delete category";
  const confirm = action.kind === "rename" ? "Save" : action.kind === "move" ? "Move" : "Delete";
  const submit = form.handleSubmit((value) => { if (action.kind === "rename") onRename(value.name); else if (action.kind === "move") onMove(value.parent_id ? Number(value.parent_id) : null, value.sort_order); else onDelete(); });
  return <div className="dialog-backdrop" role="presentation"><section className="confirmation-dialog" role="dialog" aria-modal="true" aria-labelledby="category-action-title"><div className="dialog-heading"><div><h2 id="category-action-title">{title}</h2><p>{action.kind === "rename" ? `The URL slug remains ${action.category.slug}.` : action.kind === "move" ? "Moving changes the taxonomy and article breadcrumbs, but not category URLs or article assignments." : `Delete ${action.category.name} permanently. Categories with articles or child categories cannot be deleted.`}</p></div><button className="icon-button" type="button" title="Close" aria-label="Close" disabled={pending} onClick={onCancel}><X size={18} /></button></div><form onSubmit={submit}>{action.kind === "rename" && <label>Name<input required {...form.register("name")} /></label>}{action.kind === "move" && <><label>Parent category<select {...form.register("parent_id")}><option value="">Root category</option>{options.filter(({ category }) => !blockedParents.has(category.id)).map(({ category, depth }) => <option key={category.id} value={category.id}>{`${"-- ".repeat(depth)}${category.name}`}</option>)}</select></label><label>Sort order<input type="number" {...form.register("sort_order", { valueAsNumber: true })} /></label></>}{Boolean(error) && <ErrorNotice error={error} />}<div className="dialog-actions"><button className="secondary" type="button" disabled={pending} onClick={onCancel}>Cancel</button><button className={action.kind === "delete" ? "danger-button" : ""} disabled={pending}>{pending ? "Working..." : confirm}</button></div></form></section></div>;
}

export function TagsPage() {
  const locales = useQuery({ queryKey: ["locales"], queryFn: cms.locales });
  const [params, setParams] = useSearchParams();
  const selected = params.get("locale") || locales.data?.find((item) => item.is_default)?.code || "";
  const tags = useQuery({ queryKey: ["tags", selected], queryFn: () => cms.tags(selected), enabled: Boolean(selected) });
  const form = useForm({ defaultValues: { name: "", slug: "" } });
  const [renaming, setRenaming] = useState<Tag | null>(null);
  const client = useQueryClient();
  const create = useMutation({ mutationFn: (v: { name: string; slug: string }) => cms.createTag({ ...v, locale: selected }), onSuccess: () => { form.reset(); client.invalidateQueries({ queryKey: ["tags", selected] }); } });
  const update = useMutation({ mutationFn: (tag: Tag) => cms.updateTag(tag.id, { is_enabled: !tag.is_enabled }), onSuccess: () => client.invalidateQueries({ queryKey: ["tags", selected] }) });
  const rename = useMutation({ mutationFn: ({ tag, name }: { tag: Tag; name: string }) => cms.renameTag(tag.id, selected, { name }), onSuccess: () => { setRenaming(null); client.invalidateQueries({ queryKey: ["tags", selected] }); } });
  return <section className="page"><PageTitle title="Tags" description="Maintain localized tags." /><div className="filters"><LocaleSelect value={selected} onChange={(value) => setParams({ locale: value })} /></div><form className="inline-form" onSubmit={form.handleSubmit((v) => create.mutate(v))}><input placeholder="Name" required {...form.register("name")} /><input placeholder="slug" required {...form.register("slug")} /><button disabled={create.isPending}><Plus size={16} />Add</button></form>{create.error && <ErrorNotice error={create.error} />}{update.error && <ErrorNotice error={update.error} />}{tags.error && <ErrorNotice error={tags.error} />}<table className="tags-table"><thead><tr><th>Name</th><th>Slug</th><th>Status</th><th>Actions</th></tr></thead><tbody>{tags.data?.map((item) => <tr key={item.id}><td>{item.name}</td><td>{item.slug}</td><td><label className="tag-enabled-control"><input type="checkbox" role="switch" checked={item.is_enabled} disabled={update.isPending && update.variables?.id === item.id} onChange={() => update.mutate(item)} /><span>{item.is_enabled ? "Enabled" : "Disabled"}</span></label></td><td><button className="icon-button" type="button" title="Rename tag" aria-label={`Rename ${item.name}`} onClick={() => setRenaming(item)}><Pencil size={16} /></button></td></tr>)}</tbody></table>{renaming && <TagRenameDialog tag={renaming} pending={rename.isPending} error={rename.error} onCancel={() => setRenaming(null)} onRename={(name) => rename.mutate({ tag: renaming, name })} />}</section>;
}

function TagRenameDialog({ tag, pending, error, onCancel, onRename }: { tag: Tag; pending: boolean; error: unknown; onCancel(): void; onRename(name: string): void }) {
  const form = useForm({ defaultValues: { name: tag.name } });
  useEffect(() => form.reset({ name: tag.name }), [tag, form]);
  return <div className="dialog-backdrop" role="presentation"><section className="confirmation-dialog" role="dialog" aria-modal="true" aria-labelledby="tag-rename-title"><div className="dialog-heading"><div><h2 id="tag-rename-title">Rename tag</h2><p>The URL slug remains {tag.slug}.</p></div><button className="icon-button" type="button" title="Close" aria-label="Close" disabled={pending} onClick={onCancel}><X size={18} /></button></div><form onSubmit={form.handleSubmit((value) => onRename(value.name))}><label>Name<input required {...form.register("name")} /></label>{Boolean(error) && <ErrorNotice error={error} />}<div className="dialog-actions"><button className="secondary" type="button" disabled={pending} onClick={onCancel}>Cancel</button><button disabled={pending}>{pending ? "Working..." : "Save"}</button></div></form></section></div>;
}
function PageTitle({ title, description }: { title: string; description: string }) { return <div className="page-heading"><div><h1>{title}</h1><p>{description}</p></div></div>; }

export function LocalesPage() {
  const query = useQuery({ queryKey: ["locales"], queryFn: cms.locales });
  const form = useForm({ defaultValues: { code: "", name: "", sort_order: 0, is_enabled: true } });
  const client = useQueryClient();
  const refreshLocales = () => client.invalidateQueries({ queryKey: ["locales"] });
  const create = useMutation({
    mutationFn: (value: { code: string; name: string; sort_order: number; is_enabled: boolean }) => cms.createLocale(value),
    onSuccess: () => { form.reset(); refreshLocales(); },
  });
  const update = useMutation({
    mutationFn: ({ locale, isEnabled, isDefault }: { locale: Locale; isEnabled: boolean; isDefault: boolean }) => cms.updateLocale(locale.code, { name: locale.name, is_enabled: isEnabled, sort_order: locale.sort_order, is_default: isDefault }),
    onSuccess: refreshLocales,
  });

  return <section className="page">
    <div className="page-heading"><div><h1>Locales</h1><p>Enabled locales are available for CMS translations.</p></div></div>
    <form className="inline-form" onSubmit={form.handleSubmit((v) => create.mutate(v))}>
      <input placeholder="en-US" required {...form.register("code")} />
      <input placeholder="English" required {...form.register("name")} />
      <input type="number" aria-label="Sort order" {...form.register("sort_order", { valueAsNumber: true })} />
      <label className="checkbox"><input type="checkbox" {...form.register("is_enabled")} />Enabled</label>
      <button disabled={create.isPending}><Plus size={16} />Add</button>
    </form>
    {create.error && <ErrorNotice error={create.error} />}
    {update.error && <ErrorNotice error={update.error} />}
    {query.error && <ErrorNotice error={query.error} />}
    <table className="locales-table">
      <thead><tr><th>Code</th><th>Name</th><th>Status</th><th>Default</th><th>Actions</th></tr></thead>
      <tbody>{query.data?.map((item) => {
        const rowPending = update.isPending && update.variables?.locale.code === item.code;
        return <tr key={item.code}>
          <td>{item.code}</td>
          <td>{item.name}</td>
          <td><label className="locale-enabled-control"><input type="checkbox" role="switch" checked={item.is_enabled} disabled={item.is_default || rowPending} onChange={() => update.mutate({ locale: item, isEnabled: !item.is_enabled, isDefault: item.is_default })} /><span>{item.is_enabled ? "Enabled" : "Disabled"}</span></label></td>
          <td>{item.is_default ? <span className="default-locale">Default</span> : ""}</td>
          <td>{!item.is_default && <button className="icon-button" type="button" title={item.is_enabled ? "Set as default locale" : "Enable this locale before setting it as default"} aria-label={`Set ${item.name} as default locale`} disabled={!item.is_enabled || rowPending} onClick={() => update.mutate({ locale: item, isEnabled: item.is_enabled, isDefault: true })}><Star size={16} /></button>}</td>
        </tr>;
      })}</tbody>
    </table>
  </section>;
}
function ErrorNotice({ error }: { error: unknown }) { return <p className="error" role="alert"><AlertTriangle size={16} />{message(error)}</p>; }
