import { useEffect, useState } from "react"
import { Link } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { EmptyState } from "@/components/empty-state"
import { Seo } from "@/components/seo"
import { Button } from "@/components/ui/button"
import { deleteClient, listClients } from "@/lib/api"
import type { Client } from "@/lib/types"

export function ClientsPage() {
  const { user, loading } = useAuth()
  const [clients, setClients] = useState<Client[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!user) return
    void listClients()
      .then(setClients)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erreur"))
  }, [user])

  if (!loading && !user) {
    return (
      <EmptyState
        title="Clients réservés au compte"
        description="Connectez-vous pour enregistrer vos clients, ou créez un document en mode invité."
        action={
          <div className="flex flex-wrap gap-2">
            <Button render={<Link to="/connexion" />}>Connexion</Button>
            <Button variant="outline" render={<Link to="/documents/nouveau" />}>
              Mode invité
            </Button>
          </div>
        }
      />
    )
  }

  return (
    <section className="space-y-6">
      <Seo title="Clients" path="/clients" noIndex />
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="font-heading text-3xl font-semibold tracking-tight">Clients</h1>
        <Button render={<Link to="/clients/nouveau" />}>Nouveau client</Button>
      </div>
      {error && <p className="text-sm text-destructive">{error}</p>}
      {clients.length === 0 ? (
        <EmptyState
          title="Aucun client"
          description="Ajoutez un client pour le sélectionner dans vos documents."
          action={<Button render={<Link to="/clients/nouveau" />}>Nouveau client</Button>}
        />
      ) : (
        <ul className="divide-y divide-border rounded-xl border border-border bg-card/60">
          {clients.map((c) => (
            <li key={c.id} className="flex flex-wrap items-center justify-between gap-2 px-4 py-3">
              <div>
                <Link className="font-medium underline-offset-4 hover:underline" to={`/clients/${c.id}`}>
                  {c.name}
                </Link>
                {c.email && <p className="text-sm text-muted-foreground">{c.email}</p>}
              </div>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => {
                  void deleteClient(c.id).then(() =>
                    setClients((prev) => prev.filter((x) => x.id !== c.id)),
                  )
                }}
              >
                Supprimer
              </Button>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
