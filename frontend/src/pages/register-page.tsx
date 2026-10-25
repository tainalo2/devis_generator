import { useState, type FormEvent } from "react"
import { Link, Navigate, useNavigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { Button } from "@/components/ui/button"

export function RegisterPage() {
  const { user, loading, register } = useAuth()
  const navigate = useNavigate()
  const [displayName, setDisplayName] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  if (!loading && user) return <Navigate to="/" replace />

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setPending(true)
    try {
      await register(email, password, displayName)
      navigate("/", { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : "Inscription impossible.")
    } finally {
      setPending(false)
    }
  }

  return (
    <section className="mx-auto w-full max-w-md space-y-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-semibold tracking-tight">Inscription</h1>
        <p className="text-muted-foreground">
          Créez un compte pour sauvegarder vos clients et documents. Le mode invité reste
          disponible.
        </p>
      </div>

      <form onSubmit={onSubmit} className="space-y-4">
        <label className="block space-y-1.5 text-sm">
          <span className="font-medium">Nom affiché (optionnel)</span>
          <input
            type="text"
            autoComplete="name"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
          />
        </label>
        <label className="block space-y-1.5 text-sm">
          <span className="font-medium">E-mail</span>
          <input
            type="email"
            autoComplete="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
          />
        </label>
        <label className="block space-y-1.5 text-sm">
          <span className="font-medium">Mot de passe (8 caractères min.)</span>
          <input
            type="password"
            autoComplete="new-password"
            required
            minLength={8}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
          />
        </label>
        {error && <p className="text-sm text-destructive">{error}</p>}
        <Button type="submit" disabled={pending} className="w-full">
          {pending ? "Création…" : "Créer mon compte"}
        </Button>
      </form>

      <p className="text-sm text-muted-foreground">
        Déjà inscrit ?{" "}
        <Link to="/connexion" className="underline underline-offset-4">
          Se connecter
        </Link>
      </p>
    </section>
  )
}
