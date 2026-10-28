import { Check, ChevronDown } from "lucide-react"
import { useState, type FormEvent } from "react"
import { Link, Navigate, useNavigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { AuthCard, FormError, PasswordInput, authLink } from "@/components/auth-ui"
import { Seo } from "@/components/seo"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { cn } from "@/lib/utils"

const benefits = [
  { title: "Vos brouillons conservés", text: "Reprenez un devis plus tard, sur n’importe quel appareil." },
  { title: "Vos clients enregistrés", text: "Coordonnées et n° de TVA vérifiés une seule fois." },
  { title: "Une numérotation tenue pour vous", text: "Continue et sans trou, comme l’exige la loi." },
]

/** Règle appliquée par le serveur (minPasswordLen). */
const MIN_PASSWORD = 8

function Benefits({ compact }: { compact?: boolean }) {
  return (
    <>
      {benefits.map((b) => (
        <div key={b.title} className="flex items-start gap-3">
          <i aria-hidden className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-brand/15 text-brand">
            <Check className="size-4" />
          </i>
          <span className="flex flex-col gap-0.5">
            <span className={cn("font-semibold", compact ? "text-sm" : "text-[15px] leading-[22px]")}>{b.title}</span>
            <span className={cn("text-muted-foreground", compact ? "text-[13px] leading-[18px]" : "text-sm")}>{b.text}</span>
          </span>
        </div>
      ))}
    </>
  )
}

export function RegisterPage() {
  const { user, loading, register } = useAuth()
  const navigate = useNavigate()
  const [displayName, setDisplayName] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [consent, setConsent] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  if (!loading && user) return <Navigate to="/tableau-de-bord" replace />

  const longEnough = password.length >= MIN_PASSWORD

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setPending(true)
    try {
      await register(email, password, displayName)
      navigate("/tableau-de-bord", { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : "Inscription impossible.")
    } finally {
      setPending(false)
    }
  }

  return (
    <section className="flex flex-col items-center justify-center gap-6 px-4 pt-6 pb-10 md:min-h-[calc(100svh-4rem-280px)] md:flex-row md:gap-16 md:py-16">
      <Seo title="Créer un compte" path="/inscription" />
      <aside className="hidden w-85 flex-col gap-5 md:flex">
        <h2 className="font-heading text-[22px] leading-7 font-semibold tracking-tight">Pourquoi un compte ?</h2>
        <Benefits />
        <p className="text-[13px] leading-[18px] text-muted-foreground">
          Gratuit, sans moyen de paiement. Vos données s’exportent et se suppriment à tout moment.
        </p>
      </aside>
      <div className="flex w-full max-w-110 flex-col gap-3">
        <details className="group rounded-xl border border-[color-mix(in_oklab,var(--brand)_30%,var(--border))] bg-[color-mix(in_oklab,var(--brand)_10%,var(--card))] px-4 py-3 md:hidden">
          <summary className="flex cursor-pointer list-none items-center justify-between text-sm font-semibold">
            Pourquoi un compte ?
            <ChevronDown className="size-4 transition-transform group-open:rotate-180" aria-hidden />
          </summary>
          <div className="flex flex-col gap-3 pt-3">
            <Benefits compact />
          </div>
        </details>
        <AuthCard title="Créer un compte" description="Deux minutes, puis votre premier devis.">
          {error && <FormError>{error}</FormError>}
          <form onSubmit={(e) => void onSubmit(e)} className="flex flex-col gap-3.5">
            <Label>
              <span>
                Nom <span className="font-normal text-muted-foreground">(facultatif)</span>
              </span>
              <Input autoComplete="name" placeholder="Prénom Nom" value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
            </Label>
            <Label>
              <span>E-mail</span>
              <Input type="email" required autoComplete="email" placeholder="nom@exemple.fr" value={email} onChange={(e) => setEmail(e.target.value)} />
            </Label>
            <Label>
              <span>Mot de passe</span>
              <PasswordInput required minLength={MIN_PASSWORD} maxLength={128} autoComplete="new-password" aria-describedby="regles-mdp" value={password} onChange={(e) => setPassword(e.target.value)} />
            </Label>
            <ul id="regles-mdp" aria-label="Règles du mot de passe" className="-mt-1.5 flex flex-col gap-1">
              <li className={cn("flex items-center gap-2 text-[13px] leading-[18px]", longEnough ? "text-foreground" : "text-muted-foreground")}>
                <span
                  aria-hidden
                  className={cn(
                    "inline-flex size-3.5 items-center justify-center rounded-full border-[1.5px] text-brand-foreground",
                    longEnough ? "border-brand bg-brand" : "border-border",
                  )}
                >
                  {longEnough && <Check className="size-2.5" strokeWidth={3.5} />}
                </span>
                {MIN_PASSWORD} caractères au moins
                <span className="sr-only">{longEnough ? " : respecté" : " : non respecté"}</span>
              </li>
              <li className="pl-5.5 text-[13px] leading-[18px] text-muted-foreground">Une phrase de passe longue reste le plus sûr.</li>
            </ul>
            <label className="flex cursor-pointer items-start gap-2 text-[13px] leading-[18px]">
              <input type="checkbox" required className="mt-px size-4 accent-primary" checked={consent} onChange={(e) => setConsent(e.target.checked)} />
              <span>
                J’accepte la{" "}
                <Link to="/confidentialite" className={authLink}>
                  politique de confidentialité
                </Link>
                .
              </span>
            </label>
            <Button type="submit" className="h-10 w-full text-sm" disabled={pending || !consent}>
              Créer mon compte
            </Button>
          </form>
          <p className="text-center text-sm text-muted-foreground">
            Déjà un compte ?{" "}
            <Link to="/connexion" className={authLink}>
              Se connecter
            </Link>
          </p>
        </AuthCard>
      </div>
    </section>
  )
}
