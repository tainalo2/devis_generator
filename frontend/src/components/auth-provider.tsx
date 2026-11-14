import {
  useCallback,
  useEffect,
  useRef,
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
import { AuthContext, type AuthContextValue } from "./use-auth"

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const initRef = useRef(false)

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
    if (!initRef.current) {
      initRef.current = true
      void refresh()
    }
  }, [refresh])

  const login = useCallback(async (email: string, password: string) => {
    const u = await apiLogin(email, password)
    setUser(u)
  }, [])

  const register = useCallback(async (email: string, password: string, displayName: string) => {
    const u = await apiRegister(email, password, displayName)
    setUser(u)
  }, [])

  const logout = useCallback(async () => {
    await apiLogout()
    setUser(null)
  }, [])

  const exportData = useCallback(async () => {
    const blob = await exportAccount()
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = "devisgenerator-vibed-export.json"
    a.click()
    URL.revokeObjectURL(url)
  }, [])

  const removeAccount = useCallback(async (password: string) => {
    await deleteAccount(password)
    setUser(null)
  }, [])

  const value: AuthContextValue = {
    user,
    loading,
    refresh,
    login,
    register,
    logout,
    exportData,
    removeAccount,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
