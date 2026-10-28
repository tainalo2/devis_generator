import { cn } from "@/lib/utils"

type Props<T extends string> = {
  label: string
  value: T
  options: { value: T; label: string }[]
  onChange: (v: T) => void
  className?: string
  /** Boutons de largeur égale (formulaire) plutôt qu'ajustés au texte (barre d'outils). */
  stretch?: boolean
}

/** Groupe de boutons à bascule (`aria-pressed`) sur fond `muted`. */
export function Segmented<T extends string>({ label, value, options, onChange, className, stretch }: Props<T>) {
  return (
    <div role="group" aria-label={label} className={cn("inline-flex gap-0.5 rounded-lg bg-muted p-0.5", stretch && "flex", className)}>
      {options.map((o) => {
        const on = o.value === value
        return (
          <button
            key={o.value}
            type="button"
            aria-pressed={on}
            onClick={() => onChange(o.value)}
            className={cn(
              "rounded-md px-3 text-[13px] font-medium whitespace-nowrap transition-colors focus-visible:ring-offset-0",
              stretch ? "h-8 flex-1" : "h-7",
              on ? "bg-card text-foreground shadow-[0_1px_2px_rgb(15_40_60/0.1)]" : "text-muted-foreground hover:text-foreground",
            )}
          >
            {o.label}
          </button>
        )
      })}
    </div>
  )
}
