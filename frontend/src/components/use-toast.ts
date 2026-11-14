import { createContext, useContext } from "react"

export type ToastContextValue = {
  push: (message: string, kind?: "info" | "success" | "error") => void
}

export const ToastContext = createContext<ToastContextValue | null>(null)

export function useToast() {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error("useToast hors ToastProvider")
  return ctx
}
