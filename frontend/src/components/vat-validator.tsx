import { useState } from "react"

import { Button } from "@/components/ui/button"
import { validateVAT, type VATValidation } from "@/lib/api"

type Props = {
  value: string
  onChange: (vat: string) => void
}

export function VATValidator({ value, onChange }: Props) {
  const [result, setResult] = useState<VATValidation | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  async function check() {
    setPending(true)
    setError(null)
    setResult(null)
    try {
      const res = await validateVAT(value)
      setResult(res)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Validation impossible")
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="space-y-2">
      <label className="block space-y-1 text-sm">
        <span className="font-medium">N° TVA intracommunautaire</span>
        <div className="flex flex-col gap-2 sm:flex-row">
          <input
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            placeholder="FR…"
            value={value}
            onChange={(e) => {
              onChange(e.target.value)
              setResult(null)
            }}
          />
          <Button type="button" variant="outline" disabled={pending || value.trim().length < 4} onClick={() => void check()}>
            Vérifier (VIES)
          </Button>
        </div>
      </label>
      {error && <p className="text-xs text-destructive">{error}</p>}
      {result && (
        <p className={`text-xs ${result.valid ? "text-foreground" : "text-destructive"}`}>
          {result.valid ? "Numéro valide selon VIES" : "Numéro invalide selon VIES"}
          {result.name && result.name !== "---" ? ` — ${result.name}` : ""}
          {result.cached ? " (cache local)" : ""}
          {" · "}
          {new Date(result.checked_at).toLocaleString("fr-FR")}
        </p>
      )}
    </div>
  )
}
