import { useState } from "react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { validateVAT, type VATValidation } from "@/lib/api"

type Props = {
  value: string
  onChange: (vat: string) => void
  label?: string
  placeholder?: string
}

export function VATValidator({ value, onChange, label = "N° de TVA intracommunautaire", placeholder = "FR 00 000000000" }: Props) {
  const [result, setResult] = useState<VATValidation | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  async function check() {
    setPending(true)
    setError(null)
    setResult(null)
    try {
      setResult(await validateVAT(value))
    } catch (err) {
      setError(err instanceof Error ? err.message : "Validation impossible")
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="flex flex-col gap-1.5">
      <label className="flex flex-col gap-1.5 text-sm font-medium">
        <span>{label}</span>
        <span className="flex gap-2">
          <Input
            placeholder={placeholder}
            value={value}
            onChange={(e) => {
              onChange(e.target.value)
              setResult(null)
            }}
          />
          <Button variant="outline" size="lg" disabled={pending || value.trim().length < 4} onClick={() => void check()}>
            Vérifier
          </Button>
        </span>
      </label>
      {error && <p className="text-xs text-destructive">{error}</p>}
      {result ? (
        <p className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
          <Badge variant={result.valid ? "success" : "warning"}>{result.valid ? "Numéro valide (VIES)" : "Numéro invalide (VIES)"}</Badge>
          {result.name && result.name !== "---" ? result.name : null}
          <span>
            {new Date(result.checked_at).toLocaleString("fr-FR")}
            {result.cached ? " · cache local" : ""}
          </span>
        </p>
      ) : (
        !error && (
          <span>
            <Badge variant="muted">Vérification VIES sur demande</Badge>
          </span>
        )
      )}
    </div>
  )
}
