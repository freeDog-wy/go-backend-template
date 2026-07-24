import { createContext, useContext, useEffect, useMemo, useState } from "react";
import * as authApi from "../api/auth";
import { refreshSession, sessionToken } from "../api/http";
import type { User } from "../api/types";

type AuthState = { ready: boolean; user: User | null; signIn(email: string, password: string): Promise<void>; signOut(): Promise<void> };
const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [ready, setReady] = useState(false);
  const [user, setUser] = useState<User | null>(null);
  useEffect(() => {
    refreshSession().then((result) => setUser(result.user ?? null)).catch(() => sessionToken.set(null)).finally(() => setReady(true));
  }, []);
  const value = useMemo<AuthState>(() => ({
    ready, user,
    signIn: async (email, password) => { const result = await authApi.login(email, password); setUser(result.user ?? null); },
    signOut: async () => { await authApi.logout(); setUser(null); },
  }), [ready, user]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
export function useAuth() { const value = useContext(AuthContext); if (!value) throw new Error("AuthProvider is required"); return value; }
