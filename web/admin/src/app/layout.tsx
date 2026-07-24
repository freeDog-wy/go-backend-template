import { FileText, FolderTree, Languages, LogOut, Moon, Sun, Tags } from "lucide-react";
import { NavLink, Outlet } from "react-router-dom";
import { useAuth } from "./auth";

const items = [
  ["/articles", "Articles", FileText], ["/categories", "Categories", FolderTree], ["/tags", "Tags", Tags], ["/locales", "Locales", Languages],
] as const;
export function AdminLayout() {
  const { user, signOut } = useAuth();
  const toggleTheme = () => {
    const theme = document.documentElement.dataset.theme === "dark" ? "light" : "dark";
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("admin-theme", theme);
  };
  return <div className="app-shell">
    <aside className="sidebar"><a className="brand" href="#/">Elseif CMS</a><nav>{items.map(([to, label, Icon]) => <NavLink key={to} to={to}><Icon size={17} />{label}</NavLink>)}</nav></aside>
    <main className="workspace"><header className="topbar"><span>{user?.email ?? "Administrator"}</span><div className="tools"><button className="icon-button" aria-label="Toggle theme" title="Toggle theme" onClick={toggleTheme}><Sun size={17} /><Moon size={17} /></button><button className="icon-button" aria-label="Sign out" title="Sign out" onClick={() => void signOut()}><LogOut size={17} /></button></div></header><Outlet /></main>
  </div>;
}
