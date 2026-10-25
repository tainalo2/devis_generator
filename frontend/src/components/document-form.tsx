import { useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { previewTotals } from "@/lib/api"
import {
  DOC_KINDS,
  VAT_RATES,
  emptyLine,
  formatEUR,
  type DocLine,
  type TotalsPreview,
} from "@/lib/types"

export type DocumentFormValue = {
  kind: string
  title: string
  vat_regime: string
  client_id: string
  issue_date: string
  due_date: string
  notes: string
  payment_terms: string
  lines: DocLine[]
}

type Props = {
  value: DocumentFormValue
  onChange: (v: DocumentFormValue) => void
  clients?: { id: string; name: string }[]
  showClientSelect?: boolean
  readOnly?: boolean
}

export function DocumentForm({ value, onChange, clients = [], showClientSelect, readOnly }: Props) {
  const [totals, setTotals] = useState<TotalsPreview | null>(null)
  const [calcError, setCalcError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    const t = window.setTimeout(() => {
      void previewTotals(value.vat_regime, value.lines)
        .then((res) => {
          if (!cancelled) {
            setTotals(res)
            setCalcError(null)
          }
        })
        .catch((err: unknown) => {
          if (!cancelled) {
            setTotals(null)
            setCalcError(err instanceof Error ? err.message : "Calcul impossible")
          }
        })
    }, 200)
    return () => {
      cancelled = true
      window.clearTimeout(t)
    }
  }, [value.vat_regime, value.lines])

  function patch(p: Partial<DocumentFormValue>) {
    onChange({ ...value, ...p })
  }

  function updateLine(i: number, p: Partial<DocLine>) {
    const lines = value.lines.map((l, idx) => (idx === i ? { ...l, ...p } : l))
    patch({ lines })
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-2">
        <label className="space-y-1.5 text-sm">
          <span className="font-medium">Type</span>
          <select
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            value={value.kind}
            disabled={readOnly}
            onChange={(e) => patch({ kind: e.target.value })}
          >
            {DOC_KINDS.map((k) => (
              <option key={k.value} value={k.value}>
                {k.label}
              </option>
            ))}
          </select>
        </label>
        <label className="space-y-1.5 text-sm">
          <span className="font-medium">Régime TVA</span>
          <select
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            value={value.vat_regime}
            disabled={readOnly}
            onChange={(e) => patch({ vat_regime: e.target.value })}
          >
            <option value="franchise_293b">Franchise en base (art. 293 B)</option>
            <option value="standard">TVA normal (taux par ligne)</option>
          </select>
        </label>
        {showClientSelect && (
          <label className="space-y-1.5 text-sm md:col-span-2">
            <span className="font-medium">Client</span>
            <select
              className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={value.client_id}
              disabled={readOnly}
              onChange={(e) => patch({ client_id: e.target.value })}
            >
              <option value="">— Aucun —</option>
              {clients.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </label>
        )}
        <label className="space-y-1.5 text-sm md:col-span-2">
          <span className="font-medium">Titre</span>
          <input
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            value={value.title}
            disabled={readOnly}
            onChange={(e) => patch({ title: e.target.value })}
          />
        </label>
        <label className="space-y-1.5 text-sm">
          <span className="font-medium">Date</span>
          <input
            type="date"
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            value={value.issue_date}
            disabled={readOnly}
            onChange={(e) => patch({ issue_date: e.target.value })}
          />
        </label>
        <label className="space-y-1.5 text-sm">
          <span className="font-medium">Échéance</span>
          <input
            type="date"
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            value={value.due_date}
            disabled={readOnly}
            onChange={(e) => patch({ due_date: e.target.value })}
          />
        </label>
      </div>

      <div className="space-y-3">
        <div className="flex items-center justify-between gap-2">
          <h2 className="text-lg font-medium">Lignes</h2>
          {!readOnly && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => patch({ lines: [...value.lines, emptyLine()] })}
            >
              Ajouter une ligne
            </Button>
          )}
        </div>
        <div className="space-y-3">
          {value.lines.map((line, i) => (
            <div key={i} className="grid gap-2 rounded-md border border-border p-3 md:grid-cols-12">
              <label className="space-y-1 text-sm md:col-span-4">
                <span className="text-muted-foreground">Désignation</span>
                <input
                  className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                  value={line.description}
                  disabled={readOnly}
                  onChange={(e) => updateLine(i, { description: e.target.value })}
                />
              </label>
              <label className="space-y-1 text-sm md:col-span-2">
                <span className="text-muted-foreground">Qté</span>
                <input
                  type="number"
                  step="0.001"
                  min="0.001"
                  className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                  value={(line.quantity_milli / 1000).toString()}
                  disabled={readOnly}
                  onChange={(e) =>
                    updateLine(i, {
                      quantity_milli: Math.round(Number(e.target.value || "0") * 1000),
                    })
                  }
                />
              </label>
              <label className="space-y-1 text-sm md:col-span-2">
                <span className="text-muted-foreground">P.U. HT (€)</span>
                <input
                  type="number"
                  step="0.01"
                  className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                  value={(line.unit_price_cents / 100).toString()}
                  disabled={readOnly}
                  onChange={(e) =>
                    updateLine(i, {
                      unit_price_cents: Math.round(Number(e.target.value || "0") * 100),
                    })
                  }
                />
              </label>
              <label className="space-y-1 text-sm md:col-span-2">
                <span className="text-muted-foreground">TVA</span>
                <select
                  className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
                  value={line.tax_rate_bps}
                  disabled={readOnly || value.vat_regime === "franchise_293b"}
                  onChange={(e) => updateLine(i, { tax_rate_bps: Number(e.target.value) })}
                >
                  {VAT_RATES.map((r) => (
                    <option key={r.bps} value={r.bps}>
                      {r.label}
                    </option>
                  ))}
                </select>
              </label>
              <div className="flex items-end md:col-span-2">
                {!readOnly && value.lines.length > 1 && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => patch({ lines: value.lines.filter((_, idx) => idx !== i) })}
                  >
                    Retirer
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="rounded-md border border-border bg-muted/30 px-4 py-3 text-sm">
        {calcError && <p className="text-destructive">{calcError}</p>}
        {totals && (
          <div className="space-y-1">
            <p>
              Total HT : <strong>{formatEUR(totals.subtotal_cents)}</strong>
            </p>
            <p>
              Total TVA : <strong>{formatEUR(totals.tax_cents)}</strong>
            </p>
            <p>
              Total TTC : <strong>{formatEUR(totals.total_cents)}</strong>
            </p>
          </div>
        )}
      </div>

      <label className="block space-y-1.5 text-sm">
        <span className="font-medium">Notes</span>
        <textarea
          className="min-h-20 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
          value={value.notes}
          disabled={readOnly}
          onChange={(e) => patch({ notes: e.target.value })}
        />
      </label>
      <label className="block space-y-1.5 text-sm">
        <span className="font-medium">Conditions de paiement</span>
        <input
          className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
          value={value.payment_terms}
          disabled={readOnly}
          onChange={(e) => patch({ payment_terms: e.target.value })}
        />
      </label>
    </div>
  )
}
