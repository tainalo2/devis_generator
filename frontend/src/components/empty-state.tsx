import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

type Props = {
  title: string
  description?: string
  action?: ReactNode
  className?: string
}

export function EmptyState({ title, description, action, className }: Props) {
  return (
    <div
      className={cn(
        "flex flex-col items-start gap-3 rounded-xl border border-dashed border-border bg-card/60 px-5 py-8",
        className,
      )}
    >
      <h3 className="font-heading text-lg font-semibold tracking-tight">{title}</h3>
      {description && <p className="max-w-md text-sm text-muted-foreground">{description}</p>}
      {action}
    </div>
  )
}
