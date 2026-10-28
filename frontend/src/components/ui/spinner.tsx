import { cn } from "@/lib/utils"

export function Spinner({ className, label = "Chargement" }: { className?: string; label?: string }) {
  return (
    <span
      role="status"
      aria-live="polite"
      className={cn("inline-flex items-center gap-2 text-sm text-muted-foreground", className)}
    >
      <span
        className="size-4 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-brand"
        aria-hidden
      />
      <span className="sr-only">{label}</span>
    </span>
  )
}
