import { Check, Heart, MessageCircle, RefreshCw, CreditCard, BarChart3, Lock } from "lucide-react"
import { useEffect, useState } from "react"
import { Link } from "react-router-dom"

import { Seo } from "@/components/seo"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { DemoChat } from "@/components/demo-chat"
import { fetchSiteConfig } from "@/lib/site-config"

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

/**
 * Hero section: emphasize legal compliance as the key differentiator.
 * Above the fold with conformity message and CTA to demo.
 */
function HeroSection() {
  return (
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
            <Badge variant="success">Conformité légale</Badge>
            <Badge variant="muted">Factur-X • réforme 2026</Badge>
            <Badge variant="muted">Français</Badge>
          </div>
          <h1 className="font-heading text-[clamp(36px,4.4vw,56px)] leading-[1.05] font-semibold tracking-tight">
            Factures électroniques conformes : l'obligation qui arrive
          </h1>
          <p className="max-w-xl font-sans text-xl font-medium tracking-[-0.01em] text-balance text-foreground/90 sm:text-2xl">
            À partir de 2026-2027, la facture électronique devient obligatoire en France. Devis Generator vous aide à respecter cette obligation sans changement d'outil.
          </p>
          <p className="max-w-lg text-muted-foreground">
            Créez des devis et factures électroniques au format Factur-X. Numérotation continue, traçabilité, chiffrement : tous les critères légaux sont inclus. Gratuit en self-hosted, cloud optionnel.
          </p>
          <div className="flex flex-col gap-3 pt-1 sm:flex-row sm:flex-wrap">
            <Button size="lg" className="h-11 sm:h-9" render={<Link to="#demo" />}>
              <MessageCircle className="size-4 mr-2" />
              Voir la démo interactive
            </Button>
            <Button size="lg" variant="outline" className="h-11 sm:h-9" render={<Link to="/documents/nouveau" />}>
              Créer un devis gratuitement
            </Button>
          </div>
        </div>
        <HeroCard />
      </div>
    </section>
  )
}

/**
 * Demo interactive section: positioned high on the page.
 */
function DemoSection() {
  return (
    <section id="demo" className="border-t border-border/70 bg-card/30">
      <div className={`${container} py-12 md:py-16`}>
        <div className="mb-8 flex flex-col gap-2">
          <h2 className="font-heading text-[28px] leading-9 font-semibold tracking-tight">
            Démo interactive
          </h2>
          <p className="max-w-2xl text-muted-foreground">
            Posez vos questions sur la conformité, la création de devis, ou l'intégration de la facturation électronique. L'assistant vous guide en temps réel.
          </p>
        </div>
        <div className="mx-auto max-w-2xl">
          <DemoChat />
        </div>
      </div>
    </section>
  )
}

/** Key features in priority order. */
const features = [
  {
    icon: Lock,
    title: "Conformité Factur-X",
    text: "Factures électroniques au format structuré Factur-X, avec numérotation continue et immuabilité. Prêt pour l'obligation légale 2026-2027 et la transmission via une PDP.",
  },
  {
    icon: RefreshCw,
    title: "Relances automatiques",
    text: "Envoyez des rappels de paiement automatiques via email. Configuration flexible, historique des envois, anti-spam intégré.",
  },
  {
    icon: CreditCard,
    title: "Paiement en ligne",
    text: "3 modes de paiement : virement bancaire, carte bancaire via Stripe, portefeuille numérique. Pas de frais cachés.",
  },
  {
    icon: BarChart3,
    title: "Analytics et rapports",
    text: "Visualisez vos revenus, analysez vos tendances. Données exportables pour vos outils comptables.",
  },
]

/**
 * Features section highlighting the key capabilities.
 */
function FeaturesSection() {
  return (
    <section aria-label="Fonctionnalités clés" className="border-t border-border/70">
      <div className={`${container} grid gap-8 py-12 md:grid-cols-2 md:gap-10 md:py-16`}>
        {features.map((item, i) => (
          <div
            key={item.title}
            className="animate-fade-up flex flex-col gap-3"
            style={{ animationDelay: `${0.08 * (i + 1)}s` }}
          >
            <div className="flex size-10 items-center justify-center rounded-lg bg-brand/15 text-brand">
              <item.icon className="size-5" aria-hidden />
            </div>
            <h3 className="mt-1 font-heading text-xl font-semibold tracking-tight">{item.title}</h3>
            <p className="text-sm text-muted-foreground">{item.text}</p>
          </div>
        ))}
      </div>
    </section>
  )
}

/** Pricing tiers. */
const pricingTiers = [
  {
    name: "Gratuit (Self-hosted)",
    price: "0€",
    period: "à vie",
    description: "Installez Devis Generator sur votre propre serveur",
    features: [
      "Format de facturation électronique Factur-X inclus",
      "Numérotation continue et immuabilité",
      "IBAN et adresses chiffrés",
      "Pas d'artificiel sur les limites légales",
      "Support communautaire",
      "Limité : volume de documents/mois ou support basique",
    ],
    cta: "Installer localement",
    ctaLink: "/docs/installation",
    highlight: false,
  },
  {
    name: "Cloud Essentials",
    price: "À partir de 29€",
    period: "/mois",
    description: "Hébergement géré avec les fonctionnalités professionnelles",
    features: [
      "Format de facturation électronique Factur-X inclus",
      "Relances automatiques incluses",
      "Paiement en ligne activé (3 modes)",
      "Hébergement sécurisé et maintenance",
      "Support email",
      "Accès API pour intégrations",
    ],
    cta: "Commencer l'essai gratuit",
    ctaLink: "/inscription",
    highlight: true,
  },
  {
    name: "Cloud Plus",
    price: "À partir de 79€",
    period: "/mois",
    description: "Pour les équipes avec besoins avancés",
    features: [
      "Tout de Cloud Essentials",
      "Onboarding et migration assistés",
      "SLA support (réponse 24h)",
      "Multi-comptes et gestion d'équipe",
      "Rapports et analytics avancés",
      "Webhook et automations custom",
    ],
    cta: "Contacter commercial",
    ctaLink: "/contact",
    highlight: false,
  },
]

/**
 * Pricing section with 3 tiers.
 */
function PricingSection() {
  return (
    <section className="border-t border-border/70 bg-card/30">
      <div className={`${container} py-12 md:py-16`}>
        <div className="mb-12 flex flex-col gap-2 text-center">
          <h2 className="font-heading text-[28px] leading-9 font-semibold tracking-tight">
            Tarification simple et transparente
          </h2>
          <p className="mx-auto max-w-2xl text-muted-foreground">
            Aucun prix caché. Conformité légale incluse dans tous les plans.
          </p>
        </div>
        <div className="grid gap-6 md:grid-cols-3">
          {pricingTiers.map((tier) => (
            <div
              key={tier.name}
              className={`flex flex-col gap-6 rounded-xl border p-6 ${
                tier.highlight
                  ? "border-brand/50 bg-brand/5 ring-1 ring-brand/20"
                  : "border-border bg-card/70"
              }`}
            >
              <div>
                <h3 className="font-heading text-xl font-semibold">{tier.name}</h3>
                <p className="mt-1 text-sm text-muted-foreground">{tier.description}</p>
              </div>
              <div>
                <p className="font-heading text-3xl font-semibold">
                  {tier.price}
                  <span className="text-sm text-muted-foreground">{tier.period}</span>
                </p>
              </div>
              <ul className="flex flex-col gap-2.5">
                {tier.features.map((feature) => (
                  <li key={feature} className="flex items-start gap-2 text-sm">
                    <Check className="mt-0.5 size-4 shrink-0 text-brand" aria-hidden />
                    <span>{feature}</span>
                  </li>
                ))}
              </ul>
              <Button
                className="mt-auto w-full"
                variant={tier.highlight ? "default" : "outline"}
                render={<Link to={tier.ctaLink} />}
              >
                {tier.cta}
              </Button>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

/**
 * Self-hosting / open-source section: trust + sovereignty argument.
 */
function SelfHostingSection() {
  return (
    <section className="border-t border-border/70">
      <div className={`${container} grid items-center gap-12 py-12 md:py-16 lg:grid-cols-[1fr_1.4fr] lg:gap-16`}>
        <div className="flex max-w-2xl flex-col gap-4">
          <h2 className="font-heading text-[28px] leading-9 font-semibold tracking-tight">
            Souveraineté numérique et transparence
          </h2>
          <p className="text-muted-foreground">
            Devis Generator est conçu pour la confiance. Le code source est accessible, la documentation détaille toutes les intégrations légales, et vous gardez le contrôle complet de vos données.
          </p>
          <ul className="flex flex-col gap-3">
            <li className="flex items-center gap-3 text-sm">
              <Check className="size-4 shrink-0 text-brand" aria-hidden />
              <span>Installez sur votre propre serveur (open-source)</span>
            </li>
            <li className="flex items-center gap-3 text-sm">
              <Check className="size-4 shrink-0 text-brand" aria-hidden />
              <span>Aucune dépendance externe pour la conformité légale</span>
            </li>
            <li className="flex items-center gap-3 text-sm">
              <Check className="size-4 shrink-0 text-brand" aria-hidden />
              <span>Données chiffrées, audit complet disponible</span>
            </li>
            <li className="flex items-center gap-3 text-sm">
              <Check className="size-4 shrink-0 text-brand" aria-hidden />
              <span>Conforme RGPD : export et suppression totale</span>
            </li>
          </ul>
          <Button variant="outline" render={<a href="https://github.com/tainalo2/devis_generator" target="_blank" rel="noopener noreferrer" />}>
            Voir le code sur GitHub
          </Button>
        </div>
        <div className="rounded-xl border border-border bg-card/70 p-8">
          <div className="space-y-4">
            <div className="h-3 w-32 rounded bg-brand/40" />
            <div className="h-2 w-48 rounded bg-muted-foreground/20" />
            <div className="h-2 w-40 rounded bg-muted-foreground/15" />
            <Separator className="my-4" />
            <div className="space-y-2">
              {[1, 2, 3].map((i) => (
                <div key={i} className="flex gap-2">
                  <div className="h-2 w-8 rounded bg-brand/30" />
                  <div className="h-2 w-24 rounded bg-muted-foreground/15" />
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

/** FAQ focused on legal compliance concerns. */
const faqItems = [
  {
    q: "Quelle est la différence entre Factur-X et une PDP ?",
    a: "Factur-X est un format de facture électronique. Une PDP (Plateforme de Dématérialisation Partenaire) est un intermédiaire qui transmet les factures et les données réglementaires. Devis Generator génère nativement le format Factur-X, destiné à être transmis par l'intermédiaire d'une PDP dans le cadre de la réforme 2026-2027.",
  },
  {
    q: "Qui est concerné par cette obligation ?",
    a: "Tous les micro-entrepreneurs, indépendants, TPE et entreprises qui émettent des factures en France. Les seules exceptions sont les très petites activités. Si vous avez un SIRET, vous êtes concerné.",
  },
  {
    q: "Dois-je migrer mes anciens documents ?",
    a: "Non. Seuls les nouveaux documents concernés doivent suivre les règles de facturation électronique, par exemple au format Factur-X et avec une transmission via une PDP. Vos archives restent telles qu'elles sont. Devis Generator les conserve, et vous pouvez les exporter.",
  },
  {
    q: "Le self-hosted ou cloud : lequel choisir ?",
    a: "Self-hosted : contrôle total, gratuit, idéal pour les petites équipes. Cloud : hébergement géré, relances automatiques, paiement intégré. Choisissez selon vos besoins de maintenance.",
  },
  {
    q: "Mes données sont-elles sécurisées ?",
    a: "Oui. IBAN et adresses sont chiffrés AES-256. Les serveurs cloud sont en France. Vous pouvez exporter ou supprimer vos données à tout moment (RGPD complet).",
  },
  {
    q: "Puis-je tester avant de m'engager ?",
    a: "Bien sûr. Aucune carte bancaire requise. Créez un devis en mode invité (gratuit, sans compte), ou inscrivez-vous pour 30 jours d'essai cloud gratuit.",
  },
]

/**
 * FAQ section: legal compliance-focused questions.
 */
function FaqSection() {
  return (
    <section className={`${container} py-12 md:py-16`}>
      <div className="mb-8 flex flex-col gap-2">
        <h2 className="font-heading text-[28px] leading-9 font-semibold tracking-tight">
          Questions sur la conformité légale
        </h2>
        <p className="text-muted-foreground">
          L'obligation de facture électronique suscite beaucoup de questions. Voici les réponses aux plus fréquentes.
        </p>
      </div>
      <dl className="grid gap-6 md:grid-cols-2">
        {faqItems.map((item) => (
          <div key={item.q} className="flex flex-col gap-2 rounded-lg border border-border bg-card/30 p-4">
            <dt className="font-medium text-sm">{item.q}</dt>
            <dd className="text-sm text-muted-foreground">{item.a}</dd>
          </div>
        ))}
      </dl>
    </section>
  )
}

/**
 * Support section: optional donation / advisory.
 */
function SupportSection() {
  const [donate, setDonate] = useState("")

  useEffect(() => {
    void fetchSiteConfig().then((c) => setDonate(c.donate_url.trim()))
  }, [])

  return (
    <section className="border-y border-border/70 bg-muted/40">
      <div className={`${container} flex flex-col gap-4 py-10 sm:flex-row sm:items-center sm:justify-between sm:gap-8 md:py-12`}>
        <div className="flex max-w-xl flex-col gap-1">
          <h2 className="font-heading text-xl font-semibold tracking-tight">Soutenir Devis Generator</h2>
          <p className="text-sm text-muted-foreground">
            L'outil open-source est gratuit. Les dons optionnels aident au développement et à la maintenance.
          </p>
        </div>
        {donate ? (
          <Button className="h-11 sm:h-8" render={<a href={donate} target="_blank" rel="noopener noreferrer" />}>
            <Heart className="size-4 mr-2" />
            Faire un don
          </Button>
        ) : (
          <p className="text-sm text-muted-foreground">
            URL de don à configurer (<code className="font-mono text-xs">DONATE_URL</code>).
          </p>
        )}
      </div>
    </section>
  )
}

export function LandingPage() {
  return (
    <>
      <Seo path="/" />
      <HeroSection />
      <DemoSection />
      <FeaturesSection />
      <PricingSection />
      <SelfHostingSection />
      <FaqSection />
      <SupportSection />
    </>
  )
}
