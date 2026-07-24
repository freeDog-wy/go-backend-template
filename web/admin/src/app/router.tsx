import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createHashRouter, Navigate, Outlet, RouterProvider } from "react-router-dom";
import { AuthProvider, useAuth } from "./auth";
import { AdminLayout } from "./layout";
import { ArticlesPage, ArticleEditorPage, CategoriesPage, LocalesPage, LoginPage, TagsPage } from "../features/pages";

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 1, staleTime: 15_000 } } });
function Protected() { const { ready, user } = useAuth(); if (!ready) return <div className="loading">Restoring session...</div>; return user ? <Outlet /> : <Navigate to="/login" replace />; }
function LoginRoute() { const { ready, user } = useAuth(); if (!ready) return <div className="loading">Restoring session...</div>; return user ? <Navigate to="/articles" replace /> : <LoginPage />; }
const router = createHashRouter([{ path: "/login", element: <LoginRoute /> }, { element: <Protected />, children: [{ element: <AdminLayout />, children: [{ index: true, element: <Navigate to="/articles" replace /> }, { path: "articles", element: <ArticlesPage /> }, { path: "articles/new", element: <ArticleEditorPage /> }, { path: "articles/:id/:locale", element: <ArticleEditorPage /> }, { path: "categories", element: <CategoriesPage /> }, { path: "tags", element: <TagsPage /> }, { path: "locales", element: <LocalesPage /> }] }] }, { path: "*", element: <Navigate to="/articles" replace /> }]);
export function AppRouter() { return <QueryClientProvider client={queryClient}><AuthProvider><RouterProvider router={router} /></AuthProvider></QueryClientProvider>; }
