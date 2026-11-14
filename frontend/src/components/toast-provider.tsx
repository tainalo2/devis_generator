import {
  useCallback,
  useMemo,
  useState,
  type ReactNode,
} from "react"

import { cn } from "@/lib/utils"
import { ToastContext, type ToastContextValue, useToast } from "./use-toast"

export { useToast }

type ToastKind = "info" | "success" | "error"

type Toast = {
  id: number
  message: string
  kind: ToastKind
}

let seq = 0

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<Toast[]>([])

  const push = useCallback((message: string, kind: ToastKind = "info") => {
    const id = ++seq
    setItems((prev) => [...prev, { id, message, kind }])
    window.setTimeout(() => {
      setItems((prev) => prev.filter((t) => t.id !== id))
    }, 4200)
  }, [])

  const value: ToastContextValue = useMemo(() => ({ push }), [push])

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div
        className="pointer-events-none fixed inset-x-0 bottom-4 z-50 flex flex-col items-center gap-2 px-4"
        aria-live="polite"
        aria-relevant="additions"
      >
        {items.map((t) => (
          <div
            key={t.id}
            role="status"
            className={cn(
              "pointer-events-auto max-w-md rounded-lg border px-4 py-2 text-sm shadow-lg animate-fade-up",
              t.kind === "error" && "border-destructive/40 bg-destructive/10 text-destructive",
              t.kind === "success" && "border-brand/30 bg-card text-foreground",
              t.kind === "info" && "border-border bg-card text-foreground",
            )}
          >
            {t.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}
