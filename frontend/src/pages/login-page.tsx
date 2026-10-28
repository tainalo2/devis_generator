import { useState, type FormEvent } from "react"
import { Link, Navigate, useNavigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { AuthCard, FormError, PasswordInput, authLink } from "@/components/auth-ui"
import { Seo } from "@/components/seo"
import { Button } from "@/components/ui/button"
import { Dialog } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { requestPasswordReset } from "@/lib/api"

function ResetDialog({ open, onOpenChange, initialEmail }: { open: boolean; onOpenChange: (o: boolean) => void; initialEmail: string }) {
  const [email, setEmail] = useState(initialEmail)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      setMessage(await requestPasswordReset(email))
    } catch (err) {
      setError(err instanceof Error ? err.message : "Demande impossible.")
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange} title="Mot de passe oublié" description="Indiquez l’e-mail de votre compte.">
      {message ? (
        <p className="text-sm">{message}</p>
      ) : (
        <form onSubmit={(e) => void submit(e)} className="flex flex-col gap-3.5">
          <Label>
            <span>E-mail</span>
            <Input type="email" required autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          </Label>
          {error && <FormError>{error}</FormError>}
          <Button type="submit" size="lg" className="self-end">
            Envoyer le lien
          </Button>
        </form>
      )}
    </Dialog>
  )
}

export function LoginPage() {
  const { user, loading, login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const [reset, setReset] = useState(false)

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
    <section className="flex flex-col items-center justify-center gap-4 px-4 pt-8 pb-10 md:min-h-[calc(100svh-4rem-280px)] md:py-16">
      <Seo title="Connexion" path="/connexion" />
      <div className="w-full max-w-105">
        <AuthCard title="Connexion" description="Retrouvez vos brouillons, vos clients et vos documents.">
          {error && <FormError>{error}</FormError>}
          <form onSubmit={(e) => void onSubmit(e)} className="flex flex-col gap-3.5">
            <Label>
              <span>E-mail</span>
              <Input type="email" required autoComplete="email" placeholder="nom@exemple.fr" value={email} onChange={(e) => setEmail(e.target.value)} />
            </Label>
            <div className="flex flex-col gap-1.5">
              <div className="flex items-baseline justify-between text-sm font-medium">
                <label htmlFor="mot-de-passe">Mot de passe</label>
                <button type="button" className={`${authLink} text-[13px]`} onClick={() => setReset(true)}>
                  Mot de passe oublié ?
                </button>
              </div>
              <PasswordInput id="mot-de-passe" required autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} />
            </div>
            <Button type="submit" className="h-10 w-full text-sm" disabled={pending}>
              Se connecter
            </Button>
          </form>
          <p className="text-center text-sm text-muted-foreground">
            Pas encore de compte ?{" "}
            <Link to="/inscription" className={authLink}>
              Créer un compte
            </Link>
          </p>
        </AuthCard>
      </div>
      <div className="flex w-full max-w-105 flex-wrap items-center justify-between gap-2 rounded-xl bg-muted/70 px-4 py-3 text-sm">
        <span className="text-muted-foreground">Un devis ponctuel ?</span>
        <Link to="/documents/nouveau" className={authLink}>
          Créer un devis sans compte
        </Link>
      </div>
      <ResetDialog key={String(reset)} open={reset} onOpenChange={setReset} initialEmail={email} />
    </section>
  )
}
