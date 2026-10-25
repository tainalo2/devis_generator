import { useEffect, useState } from "react"
import { Link } from "react-router-dom"

import { Seo } from "@/components/seo"
import { Button } from "@/components/ui/button"
import { fetchSiteConfig } from "@/lib/site-config"

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
          className="animate-soft-pulse pointer-events-none absolute -right-24 top-10 -z-10 h-72 w-72 rounded-full bg-brand/20 blur-3xl"
        />
        <div className="mx-auto grid min-h-[min(88vh,52rem)] w-full max-w-6xl items-center gap-10 px-4 py-16 sm:px-6 lg:grid-cols-[1.1fr_0.9fr] lg:px-8 lg:py-20">
          <div className="animate-fade-up space-y-6">
            <p className="font-heading text-3xl font-semibold tracking-tight text-foreground sm:text-4xl lg:text-5xl">
              Devis Generator : VIBED
            </p>
            <h1 className="max-w-xl text-balance text-xl font-medium text-foreground/90 sm:text-2xl">
              Des devis et factures clairs, conformes, sans friction.
            </h1>
            <p className="max-w-lg text-muted-foreground">
              Pour micro-entrepreneurs, indépendants et petites sociétés. Créez un document en mode
              invité, ou ouvrez un compte pour les conserver.
            </p>
            <div className="flex flex-wrap gap-3">
              <Button size="lg" render={<Link to="/documents/nouveau" />}>
                Créer un devis sans compte
              </Button>
              <Button size="lg" variant="outline" render={<Link to="/inscription" />}>
                Créer un compte
              </Button>
            </div>
          </div>
          <div className="animate-fade-in relative hidden min-h-72 lg:block" aria-hidden>
            <div className="absolute inset-6 rounded-2xl border border-border/70 bg-card/70 shadow-[0_30px_80px_-40px_rgba(15,40,60,0.45)] backdrop-blur">
              <div className="space-y-3 p-6">
                <div className="h-3 w-28 rounded bg-brand/40" />
                <div className="h-2 w-48 rounded bg-muted-foreground/20" />
                <div className="mt-8 space-y-2">
                  <div className="h-2 w-full rounded bg-muted-foreground/15" />
                  <div className="h-2 w-5/6 rounded bg-muted-foreground/15" />
                  <div className="h-2 w-4/6 rounded bg-muted-foreground/15" />
                </div>
                <div className="mt-10 flex justify-end">
                  <div className="h-8 w-28 rounded-md bg-primary/90" />
                </div>
              </div>
            </div>
            <div className="absolute -bottom-2 -left-2 right-16 top-16 -z-10 rounded-2xl border border-border/40 bg-brand/10" />
          </div>
        </div>
      </section>

      <section className="border-t border-border/70 bg-card/30">
        <div className="mx-auto grid w-full max-w-6xl gap-10 px-4 py-16 sm:px-6 lg:grid-cols-3 lg:px-8">
          {[
            {
              title: "Invité ou compte",
              text: "Générez un PDF immédiatement. Avec un compte, brouillons, clients et historique restent disponibles jusqu’à suppression.",
            },
            {
              title: "TVA maîtrisée",
              text: "Franchise art. 293 B ou taux par ligne (0 / 2,1 / 5,5 / 10 / 20 %). Mentions figées à l’émission.",
            },
            {
              title: "Sources officielles",
              text: "Recherche d’entreprises, VIES, TEDB et BCE derrière le cache légal, avec repli si une source est indisponible.",
            },
          ].map((item, i) => (
            <div key={item.title} className="animate-fade-up space-y-2" style={{ animationDelay: `${0.08 * (i + 1)}s` }}>
              <h2 className="font-heading text-xl font-semibold tracking-tight">{item.title}</h2>
              <p className="text-sm text-muted-foreground">{item.text}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mx-auto w-full max-w-6xl px-4 py-16 sm:px-6 lg:px-8">
        <div className="max-w-2xl space-y-3">
          <h2 className="font-heading text-2xl font-semibold tracking-tight">Conformité utile, pas du marketing</h2>
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
      </section>

      <section className="border-y border-border/70 bg-muted/40">
        <div className="mx-auto flex w-full max-w-6xl flex-col gap-4 px-4 py-12 sm:flex-row sm:items-center sm:justify-between sm:px-6 lg:px-8">
          <div className="max-w-xl space-y-1">
            <h2 className="font-heading text-xl font-semibold tracking-tight">Soutenir VIBED</h2>
            <p className="text-sm text-muted-foreground">
              L’outil est gratuit. Les dons restent optionnels ; aucun paiement n’est intégré dans
              l’application.
            </p>
          </div>
          {donate ? (
            <Button render={<a href={donate} target="_blank" rel="noopener noreferrer" />}>
              Faire un don
            </Button>
          ) : (
            <p className="text-sm text-muted-foreground">
              URL de don à configurer (<code className="font-mono text-xs">DONATE_URL</code>).
            </p>
          )}
        </div>
      </section>

      <section className="mx-auto w-full max-w-6xl px-4 py-16 sm:px-6 lg:px-8">
        <h2 className="font-heading text-2xl font-semibold tracking-tight">Questions fréquentes</h2>
        <dl className="mt-6 space-y-5">
          {[
            {
              q: "Dois-je créer un compte ?",
              a: "Non. Le mode invité permet de générer un PDF sans conservation serveur. Un compte optionnel sauvegarde clients et documents.",
            },
            {
              q: "Que devient mes données ?",
              a: "Elles restent tant que le compte existe. Export JSON et suppression complète sont disponibles dans Mon compte.",
            },
            {
              q: "Factur-X est-il inclus ?",
              a: "Pas en V1 (PDF simple). Le calendrier d’obligation pour les micro-entrepreneurs est rappelé dans les sources légales (à partir du 1er sept. 2027).",
            },
          ].map((item) => (
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
