import { useEffect, useState, type FormEvent } from "react"
import { Navigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { Button } from "@/components/ui/button"
import { getIssuer, saveIssuer } from "@/lib/api"
import { emptyAddress, type IssuerProfile } from "@/lib/types"

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
      <form onSubmit={onSubmit} className="space-y-3">
        <input
          className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
          placeholder="Raison sociale"
          value={form.legal_name}
          onChange={(e) => setForm({ ...form, legal_name: e.target.value })}
        />
        <input
          className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
          placeholder="SIRET"
          value={form.siret}
          onChange={(e) => setForm({ ...form, siret: e.target.value })}
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
