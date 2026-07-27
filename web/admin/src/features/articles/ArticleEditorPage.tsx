import { zodResolver } from "@hookform/resolvers/zod";
import CodeMirror, { type ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { markdown, markdownLanguage } from "@codemirror/lang-markdown";
import { keymap } from "@codemirror/view";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Bold, Code2, Heading2, Image, Italic, Link as LinkIcon, List, ListOrdered, Quote, Save, Send, Table2, Upload, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useNavigate, useParams } from "react-router-dom";
import { z } from "zod";
import { cms } from "../../api/cms";
import { ApiError } from "../../api/http";
import type { Article, ArticleInput, MediaItem, PublishCheck } from "../../api/types";

const articleSchema = z.object({
  locale: z.string().min(1, "Locale is required"),
  title: z.string().min(1, "Title is required"),
  slug: z.string().min(1, "Slug is required").regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/, "Slug must use lowercase letters, numbers and hyphens"),
  summary: z.string(), content: z.string(), content_format: z.literal("markdown"), seo_title: z.string(), seo_description: z.string(), canonical_url: z.string(),
});
type ArticleForm = z.infer<typeof articleSchema>;

const emptyArticle: ArticleForm = { locale: "", title: "", slug: "", summary: "", content: "", content_format: "markdown", seo_title: "", seo_description: "", canonical_url: "" };
const errorMessage = (error: unknown) => error instanceof ApiError ? `${error.code}: ${error.message}` : error instanceof Error ? error.message : "Request failed";

export function ArticleEditorPage() {
  const { id, locale: routeLocale } = useParams();
  const articleID = id ? Number(id) : null;
  const navigate = useNavigate();
  const client = useQueryClient();
  const [publishChecks, setPublishChecks] = useState<PublishCheck[]>([]);
  const [mediaOpen, setMediaOpen] = useState(false);
  const locales = useQuery({ queryKey: ["locales"], queryFn: cms.locales });
  const detail = useQuery({ queryKey: ["article", articleID, routeLocale], queryFn: () => cms.article(articleID!, routeLocale!), enabled: Boolean(articleID && routeLocale) });
  const form = useForm<ArticleForm>({ resolver: zodResolver(articleSchema), defaultValues: { ...emptyArticle, locale: routeLocale ?? "" } });

  useEffect(() => {
    if (detail.data) form.reset({ locale: detail.data.locale, title: detail.data.title, slug: detail.data.slug, summary: detail.data.summary, content: detail.data.content, content_format: "markdown", seo_title: detail.data.seo_title, seo_description: detail.data.seo_description, canonical_url: detail.data.canonical_url });
  }, [detail.data, form]);

  useEffect(() => {
    if (!articleID && !form.getValues("locale")) {
      const defaultLocale = locales.data?.find((item) => item.is_default)?.code;
      if (defaultLocale) form.setValue("locale", defaultLocale);
    }
  }, [articleID, form, locales.data]);

  useEffect(() => {
    const warn = (event: BeforeUnloadEvent) => { if (form.formState.isDirty) event.preventDefault(); };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [form.formState.isDirty]);

  const saveMutation = useMutation({ mutationFn: async (input: ArticleInput) => articleID ? cms.updateArticle(articleID, routeLocale!, input) : cms.createArticle(input) });
  const publishMutation = useMutation({ mutationFn: async (values: ArticleForm) => {
    const saved = await saveMutation.mutateAsync(values);
    const preview = await cms.previewPublish(saved.id, saved.locale);
    setPublishChecks(preview.checks);
    if (!preview.publishable) return { saved, published: false };
    if (!window.confirm("All blocking checks passed. Publish this article now?")) return { saved, published: false };
    await cms.publish(saved.id, saved.locale);
    return { saved, published: true };
  }});

  async function afterSave(article: Article, values: ArticleForm) {
    form.reset(values);
    await client.invalidateQueries({ queryKey: ["articles"] });
    await client.invalidateQueries({ queryKey: ["article", article.id, article.locale] });
    if (!articleID) navigate(`/articles/${article.id}/${article.locale}`, { replace: true });
  }

  const save = form.handleSubmit(async (values) => {
    const article = await saveMutation.mutateAsync(values);
    await afterSave(article, values);
  });
  const publish = form.handleSubmit(async (values) => {
    const result = await publishMutation.mutateAsync(values);
    await afterSave(result.saved, values);
  });

  const content = form.watch("content");
  const debouncedContent = useDebouncedValue(content, 400);
  const preview = useQuery({ queryKey: ["markdown-preview", debouncedContent], queryFn: () => cms.previewMarkdown(debouncedContent), enabled: debouncedContent.trim().length > 0, retry: false });

  if (detail.error) return <section className="page"><ErrorNotice error={detail.error} /></section>;
  return <section className="page article-editor-page">
    <div className="page-heading editor-heading"><div><h1>{articleID ? "Edit article" : "New article"}</h1><p>{form.formState.isDirty ? "Unsaved changes" : detail.data ? `Saved · ${detail.data.status}` : "Create a Markdown article"}</p></div><div className="editor-actions"><button className="secondary" type="button" onClick={() => void save()} disabled={saveMutation.isPending}><Save size={16} />Save draft</button><button type="button" onClick={() => void publish()} disabled={publishMutation.isPending}><Send size={16} />{publishMutation.isPending ? "Checking..." : "Publish"}</button></div></div>
    <form className="article-editor-form" onSubmit={(event) => event.preventDefault()}>
      <div className="article-meta-grid">
        <label>Locale{articleID ? <input readOnly {...form.register("locale")} /> : <select {...form.register("locale")}>{locales.data?.filter((item) => item.is_enabled).map((item) => <option key={item.code} value={item.code}>{item.name}</option>)}</select>}</label>
        <label>Slug<input {...form.register("slug")} /></label>
        <label className="wide">Title<input {...form.register("title")} /></label>
        <label className="wide">Summary<textarea rows={3} {...form.register("summary")} /></label>
      </div>
      <Controller name="content" control={form.control} render={({ field }) => <MarkdownWorkspace value={field.value} onChange={field.onChange} onSave={() => void save()} onMedia={() => setMediaOpen(true)} previewHTML={preview.data?.html ?? ""} previewPending={preview.isFetching} previewError={preview.error} readingMinutes={preview.data?.reading_minutes} />} />
      <details className="seo-fields"><summary>SEO settings</summary><div className="article-meta-grid"><label>SEO title<input {...form.register("seo_title")} /></label><label>Canonical URL<input {...form.register("canonical_url")} /></label><label className="wide">SEO description<textarea rows={3} {...form.register("seo_description")} /></label></div></details>
      {Object.values(form.formState.errors).map((error) => <p className="error" key={error.message}>{error.message}</p>)}
      {(saveMutation.error || publishMutation.error) && <ErrorNotice error={saveMutation.error ?? publishMutation.error} />}
      {publishChecks.length > 0 && <PublishChecklist checks={publishChecks} />}
    </form>
    {mediaOpen && <MediaPicker onClose={() => setMediaOpen(false)} onSelect={(item, alt) => { insertIntoActiveEditor(`![${alt}](${item.public_url})`); setMediaOpen(false); }} />}
  </section>;
}

let activeEditor: ReactCodeMirrorRef | null = null;
function insertIntoActiveEditor(text: string) {
  const view = activeEditor?.view;
  if (!view) return;
  const selection = view.state.selection.main;
  view.dispatch({ changes: { from: selection.from, to: selection.to, insert: text }, selection: { anchor: selection.from + text.length } });
  view.focus();
}

function MarkdownWorkspace({ value, onChange, onSave, onMedia, previewHTML, previewPending, previewError, readingMinutes }: { value: string; onChange(value: string): void; onSave(): void; onMedia(): void; previewHTML: string; previewPending: boolean; previewError: unknown; readingMinutes?: number }) {
  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const extensions = useMemo(() => [markdown({ base: markdownLanguage }), keymap.of([{ key: "Mod-s", run: () => { onSave(); return true; } }])], [onSave]);
  const command = (prefix: string, suffix = prefix, placeholder = "text") => {
    const view = editorRef.current?.view;
    if (!view) return;
    const selection = view.state.selection.main;
    const selected = view.state.sliceDoc(selection.from, selection.to) || placeholder;
    const insert = `${prefix}${selected}${suffix}`;
    view.dispatch({ changes: { from: selection.from, to: selection.to, insert }, selection: { anchor: selection.from + prefix.length, head: selection.from + prefix.length + selected.length } });
    view.focus();
  };
  return <div className="markdown-workspace">
    <div className="markdown-toolbar" role="toolbar" aria-label="Markdown formatting">
      <button type="button" title="Heading" onClick={() => command("## ", "", "Heading")}><Heading2 size={16} /></button><button type="button" title="Bold" onClick={() => command("**", "**")}><Bold size={16} /></button><button type="button" title="Italic" onClick={() => command("_", "_")}><Italic size={16} /></button><button type="button" title="Quote" onClick={() => command("> ", "", "Quote")}><Quote size={16} /></button><button type="button" title="Bulleted list" onClick={() => command("- ", "", "List item")}><List size={16} /></button><button type="button" title="Numbered list" onClick={() => command("1. ", "", "List item")}><ListOrdered size={16} /></button><button type="button" title="Link" onClick={() => command("[", "](https://example.com)", "link text")}><LinkIcon size={16} /></button><button type="button" title="Code block" onClick={() => command("```\n", "\n```", "code")}><Code2 size={16} /></button><button type="button" title="Table" onClick={() => command("| Column | Column |\n| --- | --- |\n| Value | Value |\n", "", "")}><Table2 size={16} /></button><button type="button" title="Choose image" onClick={onMedia}><Image size={16} /></button>
    </div>
    <div className="markdown-columns"><div className="markdown-editor-pane"><div className="pane-title">Markdown</div><CodeMirror ref={(instance) => { editorRef.current = instance; activeEditor = instance; }} value={value} height="560px" extensions={extensions} onChange={onChange} basicSetup={{ lineNumbers: true, foldGutter: true, highlightActiveLine: true }} /></div><div className="markdown-preview-pane"><div className="pane-title">Preview {readingMinutes ? `· ${readingMinutes} min read` : ""}{previewPending ? " · updating" : ""}</div>{previewError ? <ErrorNotice error={previewError} /> : previewHTML ? <article className="markdown-preview" dangerouslySetInnerHTML={{ __html: previewHTML }} /> : <p className="preview-empty">Start writing to see the final rendered article.</p>}</div></div>
    <div className="editor-status"><span>{value.length} characters</span><span>Ctrl/Cmd+S to save</span></div>
  </div>;
}

function PublishChecklist({ checks }: { checks: PublishCheck[] }) {
  return <section className="publish-checklist"><h2>Publication checks</h2>{checks.map((check) => <div className={check.passed ? "check passed" : check.blocking ? "check blocked" : "check warning"} key={check.name}><strong>{check.passed ? "✓" : check.blocking ? "×" : "!"} {check.name.replaceAll("_", " ")}</strong><span>{check.message}</span></div>)}</section>;
}

function MediaPicker({ onClose, onSelect }: { onClose(): void; onSelect(item: MediaItem, alt: string): void }) {
  const [alt, setAlt] = useState("");
  const media = useQuery({ queryKey: ["media"], queryFn: () => cms.media() });
  const client = useQueryClient();
  const upload = useMutation({ mutationFn: async (file: File) => { const request = await cms.requestMediaUpload(file); await cms.uploadMediaObject(request, file); await cms.completeMediaUpload(request.id); }, onSuccess: () => client.invalidateQueries({ queryKey: ["media"] }) });
  return <div className="dialog-backdrop" role="presentation"><section className="media-dialog" role="dialog" aria-modal="true" aria-label="Choose article image"><div className="dialog-heading"><div><h2>Insert image</h2><p>Upload or select a ready media asset.</p></div><button className="icon-button" type="button" onClick={onClose}><X size={18} /></button></div><label>Alternative text<input value={alt} onChange={(event) => setAlt(event.target.value)} placeholder="Describe the image" /></label><label className="media-upload"><Upload size={18} />{upload.isPending ? "Uploading..." : "Upload image"}<input type="file" accept="image/*" disabled={upload.isPending} onChange={(event) => { const file = event.target.files?.[0]; if (file) upload.mutate(file); }} /></label>{upload.error && <ErrorNotice error={upload.error} />}<div className="media-grid">{media.data?.items.filter((item) => item.status === "ready" && item.public_url).map((item) => <button type="button" className="media-card" key={item.id} disabled={!alt.trim()} onClick={() => onSelect(item, alt.trim())}><img src={item.public_url} alt="" /><span>{item.original_filename}</span><small>{item.width}×{item.height}</small></button>)}</div>{media.isLoading && <p>Loading media...</p>}{media.error && <ErrorNotice error={media.error} />}</section></div>;
}

function useDebouncedValue<T>(value: T, delay: number) {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => { const timer = window.setTimeout(() => setDebounced(value), delay); return () => window.clearTimeout(timer); }, [delay, value]);
  return debounced;
}

function ErrorNotice({ error }: { error: unknown }) { return <p className="error" role="alert">{errorMessage(error)}</p>; }
