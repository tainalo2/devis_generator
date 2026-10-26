import type { ComponentProps, ReactNode } from "react"

import { initials } from "@/lib/documents"
import { cn } from "@/lib/utils"

export function PageHeader({
  title,
  description,
  actions,
  className,
}: {
  title: ReactNode
  description?: ReactNode
  actions?: ReactNode
  className?: string
}) {
  return (
    <div className={cn("flex flex-wrap items-end justify-between gap-x-6 gap-y-3", className)}>
      <div className="flex min-w-0 flex-col gap-1">
        <h1 className="font-heading text-[30px] leading-9 font-semibold tracking-tight">{title}</h1>
        {description && <p className="text-sm text-muted-foreground">{description}</p>}
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </div>
  )
}

export function Card({ className, ...props }: ComponentProps<"div">) {
  return <div className={cn("min-w-0 rounded-xl border border-border bg-card", className)} {...props} />
}

export function CardSection({ className, ...props }: ComponentProps<"section">) {
  return <section className={cn("min-w-0 rounded-xl border border-border bg-card", className)} {...props} />
}

export function CardTitle({ className, ...props }: ComponentProps<"h2">) {
  return <h2 className={cn("font-heading text-lg leading-[26px] font-semibold tracking-tight", className)} {...props} />
}

/** Libellé de statistique en capitales (style `overline`). */
export function Overline({ className, ...props }: ComponentProps<"span">) {
  return (
    <span
      className={cn("text-[11px] leading-4 font-semibold tracking-[0.06em] text-muted-foreground uppercase", className)}
      {...props}
    />
  )
}

export type TabItem<T extends string> = { value: T; label: string; count?: number }

export function Tabs<T extends string>({
  items,
  value,
  onChange,
  label,
}: {
  items: TabItem<T>[]
  value: T
  onChange: (v: T) => void
  label: string
}) {
  return (
    <div role="tablist" aria-label={label} className="flex gap-1 overflow-x-auto border-b border-border">
      {items.map((t) => {
        const selected = t.value === value
        return (
          <button
            key={t.value}
            type="button"
            role="tab"
            aria-selected={selected}
            onClick={() => onChange(t.value)}
            className={cn(
              "-mb-px inline-flex h-10 shrink-0 items-center gap-1.5 border-b-2 px-3 text-sm font-medium whitespace-nowrap transition-colors focus-visible:ring-0 focus-visible:ring-offset-0 focus-visible:[box-shadow:inset_0_0_0_2px_var(--ring)]",
              selected ? "border-primary text-foreground" : "border-transparent text-muted-foreground hover:text-foreground",
            )}
          >
            {t.label}
            {t.count !== undefined && (
              <span className="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-muted px-1.5 text-[11px] text-muted-foreground">
                {t.count}
              </span>
            )}
          </button>
        )
      })}
    </div>
  )
}

export function DashedEmpty({ title, description, action }: { title: string; description?: string; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-2 rounded-xl border-[1.5px] border-dashed border-border px-6 py-10 text-center">
      <span className="font-heading text-lg leading-[26px] font-semibold">{title}</span>
      {description && <span className="max-w-md text-sm text-muted-foreground">{description}</span>}
      {action && <div className="mt-2">{action}</div>}
    </div>
  )
}

export function Avatar({ name, className }: { name: string; className?: string }) {
  return (
    <span
      aria-hidden
      className={cn(
        "inline-flex size-8 shrink-0 items-center justify-center rounded-full bg-secondary text-xs font-semibold text-secondary-foreground",
        className,
      )}
    >
      {initials(name)}
    </span>
  )
}
