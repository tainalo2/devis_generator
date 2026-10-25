import { useState, type FormEvent } from "react"
import { Link, Navigate, useNavigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { Button } from "@/components/ui/button"

export function LoginPage() {
  const { user, loading, login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  if (!loading && user) return <Navigate to="/tableau-de-bord" replace />

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setPending(true)
    try {
      await login(email, password)
      navigate("/tableau-de-bord", { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : "Connexion impossible.")
    } finally {
      setPending(false)
    }
  }

  return (
    <section className="mx-auto w-full max-w-md space-y-6 px-4 py-12">
      <div className="space-y-2">
        <h1 className="font-heading text-3xl font-semibold tracking-tight">Connexion</h1>
        <p className="text-muted-foreground">
          Compte optionnel — vous pouvez continuer en invité sans vous connecter.
        </p>
      </div>

      <form onSubmit={onSubmit} className="space-y-4">
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
          <span className="font-medium">Mot de passe</span>
          <input
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
          />
        </label>
        {error && <p className="text-sm text-destructive">{error}</p>}
        <Button type="submit" disabled={pending} className="w-full">
          {pending ? "Connexion…" : "Se connecter"}
        </Button>
      </form>

      <p className="text-sm text-muted-foreground">
        Pas encore de compte ?{" "}
        <Link to="/inscription" className="underline underline-offset-4">
          Créer un compte
        </Link>
      </p>
      <p className="text-sm text-muted-foreground">
        <Link to="/" className="underline underline-offset-4">
          Continuer sans compte
        </Link>
      </p>
    </section>
  )
}
