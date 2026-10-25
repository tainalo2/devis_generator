import { useEffect, useState, type FormEvent } from "react"
import { Link, Navigate, useNavigate, useParams } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { CompanyAutocomplete } from "@/components/company-autocomplete"
import { Button } from "@/components/ui/button"
import { VATValidator } from "@/components/vat-validator"
import { listClients, saveClient, type CompanyHit } from "@/lib/api"
import { emptyAddress, type Client } from "@/lib/types"

export function ClientEditPage() {
  const { id } = useParams()
  const isNew = !id || id === "nouveau"
  const { user, loading } = useAuth()
  const navigate = useNavigate()
  const [form, setForm] = useState({
    name: "",
    email: "",
    siren: "",
    siret: "",
    vat_number: "",
    iban: "",
    notes: "",
    address: emptyAddress(),
  })
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  useEffect(() => {
    if (isNew || !user) return
    void listClients().then((list) => {
      const c = list.find((x) => x.id === id)
      if (!c) return
      setForm({
        name: c.name,
        email: c.email,
        siren: c.siren,
        siret: c.siret,
        vat_number: c.vat_number,
        iban: c.iban ?? "",
        notes: c.notes,
        address: c.address ?? emptyAddress(),
      })
    })
  }, [id, isNew, user])

  if (!loading && !user) return <Navigate to="/connexion" replace />

  function applyHit(hit: CompanyHit) {
    setForm((f) => ({
      ...f,
      name: hit.name || f.name,
      siren: hit.siren || f.siren,
      siret: hit.siret || f.siret,
      vat_number: hit.vat_number || f.vat_number,
      address: {
        ...f.address,
        line1: hit.address || f.address.line1,
        postal_code: hit.postal_code || f.address.postal_code,
        city: hit.city || f.address.city,
      },
    }))
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setPending(true)
    setError(null)
    try {
      const saved: Client = await saveClient(form, isNew ? undefined : id)
      navigate(`/clients/${saved.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur")
    } finally {
      setPending(false)
    }
  }

  return (
    <section className="mx-auto max-w-xl space-y-6">
      <h1 className="text-3xl font-semibold tracking-tight">{isNew ? "Nouveau client" : "Client"}</h1>
      <CompanyAutocomplete onPick={applyHit} />
      <form onSubmit={onSubmit} className="space-y-4">
        <label className="block space-y-1 text-sm">
          <span className="font-medium">Nom</span>
          <input
            required
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
        </label>
        <label className="block space-y-1 text-sm">
          <span className="font-medium">E-mail</span>
          <input
            type="email"
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            value={form.email}
            onChange={(e) => setForm({ ...form, email: e.target.value })}
          />
        </label>
        <label className="block space-y-1 text-sm">
          <span className="font-medium">Adresse</span>
          <input
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            value={form.address.line1}
            onChange={(e) => setForm({ ...form, address: { ...form.address, line1: e.target.value } })}
          />
        </label>
        <div className="grid gap-3 sm:grid-cols-2">
          <input
            placeholder="Code postal"
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            value={form.address.postal_code}
            onChange={(e) =>
              setForm({ ...form, address: { ...form.address, postal_code: e.target.value } })
            }
          />
          <input
            placeholder="Ville"
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            value={form.address.city}
            onChange={(e) => setForm({ ...form, address: { ...form.address, city: e.target.value } })}
          />
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <input
            placeholder="SIREN"
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            value={form.siren}
            onChange={(e) => setForm({ ...form, siren: e.target.value })}
          />
          <input
            placeholder="SIRET"
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            value={form.siret}
            onChange={(e) => setForm({ ...form, siret: e.target.value })}
          />
        </div>
        <VATValidator
          value={form.vat_number}
          onChange={(vat_number) => setForm({ ...form, vat_number })}
        />
        <label className="block space-y-1 text-sm">
          <span className="font-medium">IBAN (chiffré)</span>
          <input
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
            value={form.iban}
            onChange={(e) => setForm({ ...form, iban: e.target.value })}
          />
        </label>
        {error && <p className="text-sm text-destructive">{error}</p>}
        <div className="flex gap-2">
          <Button type="submit" disabled={pending}>
            Enregistrer
          </Button>
          <Button variant="ghost" render={<Link to="/clients" />}>
            Retour
          </Button>
        </div>
      </form>
    </section>
  )
}
