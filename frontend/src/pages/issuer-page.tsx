import { useEffect, useState, type FormEvent } from "react"
import { Navigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { CompanyAutocomplete } from "@/components/company-autocomplete"
import { Button } from "@/components/ui/button"
import { VATValidator } from "@/components/vat-validator"
import { getIssuer, saveIssuer, type CompanyHit } from "@/lib/api"
import { emptyAddress, type IssuerProfile } from "@/lib/types"

function applyCompanyHit(form: IssuerProfile, hit: CompanyHit): IssuerProfile {
  const address = { ...form.address }
  if (hit.address) address.line1 = hit.address
  if (hit.postal_code) address.postal_code = hit.postal_code
  if (hit.city) address.city = hit.city
  return {
    ...form,
    legal_name: hit.name || form.legal_name,
    trade_name: hit.trade_name || form.trade_name,
    siren: hit.siren || form.siren,
    siret: hit.siret || form.siret,
    vat_number: hit.vat_number || form.vat_number,
    ape_code: hit.naf || form.ape_code,
    address,
  }
}

export function IssuerPage() {
  const { user, loading } = useAuth()
  const [form, setForm] = useState<IssuerProfile | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  useEffect(() => {
    if (!user) return
    void getIssuer()
      .then((p) => setForm({ ...p, address: p.address ?? emptyAddress() }))
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erreur"))
  }, [user])

  if (!loading && !user) return <Navigate to="/connexion" replace />
  if (!form) return <p className="text-muted-foreground">{error ?? "Chargement…"}</p>

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!form) return
    setError(null)
    try {
      const saved = await saveIssuer(form)
      setForm(saved)
      setMessage("Profil émetteur enregistré.")
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur")
    }
  }

  return (
    <section className="mx-auto max-w-xl space-y-6">
      <h1 className="text-3xl font-semibold tracking-tight">Mon entreprise</h1>
      <p className="text-muted-foreground">
        Identité figée dans les documents émis (SIRET, adresse chiffrée, régime TVA par défaut).
      </p>
      <CompanyAutocomplete
        onPick={(hit) => setForm((f) => (f ? applyCompanyHit(f, hit) : f))}
      />
      <form onSubmit={onSubmit} className="space-y-3">
        <input
          className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
          placeholder="Raison sociale"
          value={form.legal_name}
          onChange={(e) => setForm({ ...form, legal_name: e.target.value })}
        />
        <div className="grid gap-3 sm:grid-cols-2">
          <input
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            placeholder="SIREN"
            value={form.siren}
            onChange={(e) => setForm({ ...form, siren: e.target.value })}
          />
          <input
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            placeholder="SIRET"
            value={form.siret}
            onChange={(e) => setForm({ ...form, siret: e.target.value })}
          />
        </div>
        <VATValidator
          value={form.vat_number}
          onChange={(vat_number) => setForm({ ...form, vat_number })}
        />
        <input
          className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
          placeholder="Adresse"
          value={form.address.line1}
          onChange={(e) => setForm({ ...form, address: { ...form.address, line1: e.target.value } })}
        />
        <select
          className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
          value={form.default_vat_regime}
          onChange={(e) => setForm({ ...form, default_vat_regime: e.target.value })}
        >
          <option value="franchise_293b">Franchise 293 B</option>
          <option value="standard">TVA normale</option>
        </select>
        <input
          className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
          placeholder="IBAN (chiffré)"
          value={form.iban ?? ""}
          onChange={(e) => setForm({ ...form, iban: e.target.value })}
        />
        {error && <p className="text-sm text-destructive">{error}</p>}
        {message && <p className="text-sm text-muted-foreground">{message}</p>}
        <Button type="submit">Enregistrer</Button>
      </form>
    </section>
  )
}
