import { createContext, useContext } from "react"

import { type User } from "@/lib/api"

export type AuthContextValue = {
  user: User | null
  loading: boolean
  refresh: () => Promise<void>
  login: (email: string, password: string) => Promise<void>
  register: (email: string, password: string, displayName: string) => Promise<void>
  logout: () => Promise<void>
  exportData: () => Promise<void>
  removeAccount: (password: string) => Promise<void>
}

export const AuthContext = createContext<AuthContextValue | null>(null)

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth doit être utilisé dans AuthProvider")
  return ctx
}
