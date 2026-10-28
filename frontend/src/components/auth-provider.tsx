import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react"

import {
  deleteAccount,
  exportAccount,
  fetchMe,
  login as apiLogin,
  logout as apiLogout,
  register as apiRegister,
  type User,
} from "@/lib/api"

type AuthContextValue = {
  user: User | null
  loading: boolean
  refresh: () => Promise<void>
  login: (email: string, password: string) => Promise<void>
  register: (email: string, password: string, displayName: string) => Promise<void>
  logout: () => Promise<void>
  exportData: () => Promise<void>
  removeAccount: (password: string) => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    try {
      const me = await fetchMe()
      setUser(me)
    } catch {
      setUser(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const value: AuthContextValue = {
    user,
    loading,
    refresh,
    login: async (email, password) => {
      const u = await apiLogin(email, password)
      setUser(u)
    },
    register: async (email, password, displayName) => {
      const u = await apiRegister(email, password, displayName)
      setUser(u)
    },
    logout: async () => {
      await apiLogout()
      setUser(null)
    },
    exportData: async () => {
      const blob = await exportAccount()
      const url = URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = "devisgenerator-vibed-export.json"
      a.click()
      URL.revokeObjectURL(url)
    },
    removeAccount: async (password) => {
      await deleteAccount(password)
      setUser(null)
    },
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth doit être utilisé dans AuthProvider")
  return ctx
}
