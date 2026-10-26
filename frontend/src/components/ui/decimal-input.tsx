import { useState, type ComponentProps } from "react"

import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

type Props = Omit<ComponentProps<"input">, "value" | "onChange" | "type"> & {
  /** Valeur en unités entières (centimes, millièmes…). */
  value: number
  onValueChange: (v: number) => void
  /** Nombre d'unités entières par unité affichée (100 pour des centimes). */
  scale: number
  /** Décimales affichées hors saisie. */
  decimals?: number
  suffix?: string
}

function format(v: number, scale: number, decimals: number, suffix?: string) {
  const s = (v / scale).toLocaleString("fr-FR", { minimumFractionDigits: decimals, maximumFractionDigits: Math.max(decimals, 3) })
  return suffix ? `${s} ${suffix}` : s
}

/** Champ numérique à la française : virgule décimale, formaté à la sortie du champ. */
export function DecimalInput({ value, onValueChange, scale, decimals = 0, suffix, className, onFocus, onBlur, ...props }: Props) {
  const [draft, setDraft] = useState<string | null>(null)

  return (
    <Input
      {...props}
      type="text"
      inputMode="decimal"
      className={cn("text-right tabular-nums", className)}
      value={draft ?? format(value, scale, decimals, suffix)}
      onFocus={(e) => {
        setDraft(value === 0 ? "" : (value / scale).toLocaleString("fr-FR", { maximumFractionDigits: 3, useGrouping: false }))
        onFocus?.(e)
      }}
      onChange={(e) => {
        setDraft(e.target.value)
        const n = Number(e.target.value.replace(/\s/g, "").replace(",", ".").replace(/[^\d.-]/g, ""))
        if (Number.isFinite(n)) onValueChange(Math.round(n * scale))
      }}
      onBlur={(e) => {
        setDraft(null)
        onBlur?.(e)
      }}
    />
  )
}
