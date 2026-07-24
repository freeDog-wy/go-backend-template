import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { AppRouter } from "./app/router";
import "./styles.css";

const savedTheme = localStorage.getItem("admin-theme");
if (savedTheme === "light" || savedTheme === "dark") document.documentElement.dataset.theme = savedTheme;
createRoot(document.getElementById("root")!).render(<StrictMode><AppRouter /></StrictMode>);
