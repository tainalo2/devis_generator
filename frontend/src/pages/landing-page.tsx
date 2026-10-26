import { Check, CircleUserRound, Heart, Percent, Scale } from "lucide-react"
import { useEffect, useState } from "react"
import { Link } from "react-router-dom"

import { Seo } from "@/components/seo"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { fetchSiteConfig } from "@/lib/site-config"

const atouts = [
  {
    icon: CircleUserRound,
    title: "Invité ou compte",
    text: "Générez un PDF immédiatement. Avec un compte, brouillons, clients et historique restent disponibles jusqu’à suppression.",
  },
  {
    icon: Percent,
    title: "TVA maîtrisée",
    text: "Franchise art. 293 B ou taux par ligne (0 / 2,1 / 5,5 / 10 / 20 %). Mentions figées à l’émission.",
  },
  {
    icon: Scale,
    title: "Sources officielles",
    text: "Recherche d’entreprises, VIES, TEDB et BCE derrière le cache légal, avec repli si une source est indisponible.",
  },
]

const garanties = [
  "Numérotation continue des documents",
  "Documents émis immuables, corrigés par un avoir",
  "IBAN et adresses chiffrés",
  "Export et suppression RGPD",
]

const faq = [
  {
    q: "Dois-je créer un compte ?",
    a: "Non. Le mode invité permet de générer un PDF sans conservation serveur. Un compte optionnel sauvegarde clients et documents.",
  },
  {
    q: "Que deviennent mes données ?",
    a: "Elles restent tant que le compte existe. Export JSON et suppression complète sont disponibles dans Mon compte.",
  },
  {
    q: "Factur-X est-il inclus ?",
    a: "Pas en V1 (PDF simple). Le calendrier d’obligation pour les micro-entrepreneurs est rappelé dans les sources légales (à partir du 1er sept. 2027).",
  },
]

const container = "mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8"

/** Carte abstraite du héros : barres de texte arrondies et bouton primary. */
function HeroCard() {
  const lines = [
    ["58%", "w-14"],
    ["48%", "w-11"],
    ["38%", "w-13"],
  ]
  return (
    <div className="animate-fade-in relative hidden h-100 min-w-0 lg:block" aria-hidden>
      <div className="absolute top-18 right-16 bottom-4 -left-2 rounded-2xl border border-border/40 bg-brand/10" />
      <div className="absolute inset-x-6 top-6 bottom-10 flex flex-col gap-3 rounded-2xl border border-border/70 bg-card/70 p-6 shadow-[0_30px_80px_-40px_rgba(15,40,60,0.45)] backdrop-blur">
        <div className="flex items-center justify-between">
          <div className="h-3 w-28 rounded bg-brand/40" />
          <Badge variant="muted">Brouillon</Badge>
        </div>
        <div className="h-2 w-48 rounded bg-muted-foreground/20" />
        <div className="mt-6 flex flex-col gap-4">
          {lines.map(([w, right]) => (
            <div key={w} className="flex justify-between gap-6">
              <div className="h-2 rounded bg-muted-foreground/15" style={{ width: w }} />
              <div className={`h-2 rounded bg-muted-foreground/25 ${right}`} />
            </div>
          ))}
        </div>
        <Separator className="mt-3" />
        <div className="flex items-center justify-between">
          <div className="h-2.5 w-20 rounded bg-foreground/20" />
          <div className="h-3 w-24 rounded bg-brand/40" />
        </div>
        <div className="flex-1" />
        <div className="flex justify-end">
          <div className="h-8 w-28 rounded-md bg-primary/90" />
        </div>
      </div>
    </div>
  )
}

export function LandingPage() {
  const [donate, setDonate] = useState("")

  useEffect(() => {
    void fetchSiteConfig().then((c) => setDonate(c.donate_url.trim()))
  }, [])

  return (
    <>
      <Seo path="/" />
      <section className="relative overflow-hidden">
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 -z-10"
          style={{
            background:
              "linear-gradient(145deg, color-mix(in oklch, var(--brand) 18%, transparent), transparent 45%), linear-gradient(to bottom, color-mix(in oklch, var(--background) 40%, transparent), var(--background))",
          }}
        />
        <div
          aria-hidden
          className="animate-soft-pulse pointer-events-none absolute top-10 -right-24 -z-10 h-72 w-72 rounded-full bg-brand/20 blur-3xl"
        />
        <div className={`${container} grid items-center gap-12 pt-10 pb-12 lg:min-h-160 lg:grid-cols-[1.25fr_1fr] lg:py-20`}>
          <div className="animate-fade-up flex min-w-0 flex-col gap-6">
            <div className="flex flex-wrap gap-2">
              <Badge variant="success">Gratuit</Badge>
              <Badge variant="muted">Sans compte</Badge>
              <Badge variant="muted">PDF immédiat</Badge>
            </div>
            <p className="font-heading text-[clamp(36px,4.4vw,56px)] leading-[1.05] font-semibold tracking-tight">
              Devis Generator : VIBED
            </p>
            <h1 className="max-w-xl font-sans text-xl font-medium tracking-[-0.01em] text-balance text-foreground/90 sm:text-2xl">
              Des devis et factures clairs, conformes, sans friction.
            </h1>
            <p className="max-w-lg text-muted-foreground">
              Pour micro-entrepreneurs, indépendants et petites sociétés. Créez un document en mode
              invité, ou ouvrez un compte pour les conserver.
            </p>
            <div className="flex flex-col gap-3 pt-1 sm:flex-row sm:flex-wrap">
              <Button size="lg" className="h-11 sm:h-9" render={<Link to="/documents/nouveau" />}>
                Créer un devis sans compte
              </Button>
              <Button size="lg" variant="outline" className="h-11 sm:h-9" render={<Link to="/inscription" />}>
                Créer un compte
              </Button>
            </div>
          </div>
          <HeroCard />
        </div>
      </section>

      <section aria-label="Atouts" className="border-t border-border/70 bg-card/30">
        <div className={`${container} grid gap-8 py-12 md:grid-cols-3 md:gap-10 md:py-16`}>
          {atouts.map((item, i) => (
            <div
              key={item.title}
              className="animate-fade-up flex flex-col gap-3"
              style={{ animationDelay: `${0.08 * (i + 1)}s` }}
            >
              <div className="flex size-9 items-center justify-center rounded-lg bg-brand/15 text-brand">
                <item.icon className="size-4.5" aria-hidden />
              </div>
              <h2 className="mt-1 font-heading text-xl font-semibold tracking-tight">{item.title}</h2>
              <p className="text-sm text-muted-foreground">{item.text}</p>
            </div>
          ))}
        </div>
      </section>

      <section>
        <div className={`${container} grid items-center gap-7 py-12 md:py-16 lg:grid-cols-[1.4fr_1fr] lg:gap-12`}>
          <div className="flex max-w-2xl flex-col gap-3">
            <h2 className="font-heading text-[28px] leading-9 font-semibold tracking-tight">
              Conformité utile, pas du marketing
            </h2>
            <p className="text-muted-foreground">
              Numérotation continue, documents émis immuables, chiffrement des IBAN et adresses,
              export et suppression RGPD. Les pages légales détaillent ce que l’application fait vraiment.
            </p>
            <div className="flex flex-wrap gap-3 pt-2">
              <Button variant="outline" render={<Link to="/sources-legales" />}>
                Voir les sources légales
              </Button>
              <Button variant="ghost" render={<Link to="/confidentialite" />}>
                Politique de confidentialité
              </Button>
            </div>
          </div>
          <ul className="flex flex-col gap-3.5 rounded-xl border border-border bg-card/70 px-6 py-5">
            {garanties.map((g) => (
              <li key={g} className="flex items-center gap-3 text-sm">
                <Check className="size-4 shrink-0 text-brand" aria-hidden />
                {g}
              </li>
            ))}
          </ul>
        </div>
      </section>

      <section className="border-y border-border/70 bg-muted/40">
        <div className={`${container} flex flex-col gap-4 py-10 sm:flex-row sm:items-center sm:justify-between sm:gap-8 md:py-12`}>
          <div className="flex max-w-xl flex-col gap-1">
            <h2 className="font-heading text-xl font-semibold tracking-tight">Soutenir VIBED</h2>
            <p className="text-sm text-muted-foreground">
              L’outil est gratuit. Les dons restent optionnels ; aucun paiement n’est intégré dans
              l’application.
            </p>
          </div>
          {donate ? (
            <Button className="h-11 sm:h-8" render={<a href={donate} target="_blank" rel="noopener noreferrer" />}>
              <Heart />
              Faire un don
            </Button>
          ) : (
            <p className="text-sm text-muted-foreground">
              URL de don à configurer (<code className="font-mono text-xs">DONATE_URL</code>).
            </p>
          )}
        </div>
      </section>

      <section className={`${container} flex flex-col gap-6 py-12 md:py-16`}>
        <h2 className="font-heading text-[28px] leading-9 font-semibold tracking-tight">Questions fréquentes</h2>
        <dl className="flex max-w-3xl flex-col gap-5">
          {faq.map((item) => (
            <div key={item.q} className="border-b border-border pb-4">
              <dt className="font-medium">{item.q}</dt>
              <dd className="mt-1 text-sm text-muted-foreground">{item.a}</dd>
            </div>
          ))}
        </dl>
      </section>
    </>
  )
}
