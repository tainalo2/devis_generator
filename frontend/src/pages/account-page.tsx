import { useState } from "react"
import { Link, Navigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"

export function AccountPage() {
  const { user, loading, logout, exportData, removeAccount } = useAuth()
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [pending, setPending] = useState(false)

  if (!loading && !user) return <Navigate to="/connexion" replace />

  async function onExport() {
    setError(null)
    setMessage(null)
    setPending(true)
    try {
      await exportData()
      setMessage("Export téléchargé.")
    } catch (err) {
      setError(err instanceof Error ? err.message : "Export impossible.")
    } finally {
      setPending(false)
    }
  }

  async function onDelete() {
    setError(null)
    setPending(true)
    try {
      await removeAccount()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Suppression impossible.")
      setPending(false)
    }
  }

  return (
    <section className="mx-auto w-full max-w-lg space-y-6">
      <div className="space-y-2">
        <h1 className="font-heading text-3xl font-semibold tracking-tight">Mon compte</h1>
        <p className="text-muted-foreground">
          Données conservées tant que le compte existe. Vous pouvez exporter puis supprimer
          définitivement.
        </p>
      </div>

      {user && (
        <div className="space-y-1 text-sm">
          <p>
            <span className="text-muted-foreground">E-mail : </span>
            {user.email}
          </p>
          {user.display_name && (
            <p>
              <span className="text-muted-foreground">Nom : </span>
              {user.display_name}
            </p>
          )}
        </div>
      )}

      <div className="flex flex-wrap gap-3">
        <Button type="button" variant="outline" onClick={() => void onExport()} disabled={pending}>
          Exporter mes données
        </Button>
        <Button type="button" variant="ghost" onClick={() => void logout()} disabled={pending}>
          Se déconnecter
        </Button>
      </div>

      <Separator />

      <div className="space-y-3">
        <h2 className="text-lg font-medium text-destructive">Zone sensible</h2>
        <p className="text-sm text-muted-foreground">
          La suppression efface le compte et toutes les données associées. Exportez d&apos;abord si
          besoin.
        </p>
        {!confirmOpen ? (
          <Button type="button" variant="destructive" onClick={() => setConfirmOpen(true)}>
            Supprimer mon compte…
          </Button>
        ) : (
          <div className="space-y-3 rounded-lg border border-destructive/40 p-4">
            <p className="text-sm">
              Confirmez la suppression définitive. Cette action est irréversible.
            </p>
            <div className="flex flex-wrap gap-2">
              <Button type="button" variant="outline" onClick={() => void onExport()} disabled={pending}>
                Exporter avant
              </Button>
              <Button type="button" variant="destructive" onClick={() => void onDelete()} disabled={pending}>
                Confirmer la suppression
              </Button>
              <Button type="button" variant="ghost" onClick={() => setConfirmOpen(false)} disabled={pending}>
                Annuler
              </Button>
            </div>
          </div>
        )}
      </div>

      {message && <p className="text-sm text-muted-foreground">{message}</p>}
      {error && <p className="text-sm text-destructive">{error}</p>}

      <p className="text-sm">
        <Link to="/" className="underline underline-offset-4">
          Retour à l&apos;accueil
        </Link>
      </p>
    </section>
  )
}
