import { useEffect, useState } from "react"
import { Link } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
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
      <p className="text-muted-foreground">
        Connectez-vous pour gérer vos clients, ou{" "}
        <Link className="underline" to="/documents/nouveau">
          créez un document en invité
        </Link>
        .
      </p>
    )
  }

  return (
    <section className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-3xl font-semibold tracking-tight">Clients</h1>
        <Button render={<Link to="/clients/nouveau" />}>Nouveau client</Button>
      </div>
      {error && <p className="text-sm text-destructive">{error}</p>}
      <ul className="divide-y divide-border rounded-md border border-border">
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
                void deleteClient(c.id).then(() => setClients((prev) => prev.filter((x) => x.id !== c.id)))
              }}
            >
              Supprimer
            </Button>
          </li>
        ))}
        {clients.length === 0 && (
          <li className="px-4 py-6 text-sm text-muted-foreground">Aucun client pour l’instant.</li>
        )}
      </ul>
    </section>
  )
}
