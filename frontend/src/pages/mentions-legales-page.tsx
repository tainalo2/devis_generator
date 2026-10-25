import { useEffect, useState } from "react"

import { Seo } from "@/components/seo"
import { fetchSiteConfig, placeholder, type SiteConfig } from "@/lib/site-config"

export function MentionsLegalesPage() {
  const [cfg, setCfg] = useState<SiteConfig | null>(null)
  useEffect(() => {
    void fetchSiteConfig().then(setCfg)
  }, [])

  const c = cfg ?? {
    donate_url: "",
    legal_publisher_name: "",
    legal_publisher_address: "",
    legal_publisher_email: "",
    legal_publisher_siret: "",
    legal_host_name: "",
    legal_host_address: "",
    legal_host_url: "",
  }

  return (
    <article className="mx-auto w-full max-w-3xl space-y-8 px-4 py-12 sm:px-6">
      <Seo
        title="Mentions légales"
        path="/mentions-legales"
        description="Mentions légales de Devis Generator : VIBED — éditeur, hébergeur, licence."
      />
      <header className="space-y-2">
        <h1 className="font-heading text-3xl font-semibold tracking-tight">Mentions légales</h1>
        <p className="text-sm text-amber-800 dark:text-amber-200">
          Texte à faire valider juridiquement. Les champs éditeur / hébergeur sont configurables et
          restent marqués « à compléter » tant que les variables d’environnement ne sont pas renseignées.
        </p>
      </header>

      <section className="space-y-2 text-sm leading-relaxed">
        <h2 className="font-heading text-xl font-semibold">Éditeur</h2>
        <p>
          <strong>Nom / raison sociale :</strong> {placeholder(c.legal_publisher_name)}
          <br />
          <strong>Adresse :</strong> {placeholder(c.legal_publisher_address)}
          <br />
          <strong>E-mail de contact :</strong> {placeholder(c.legal_publisher_email)}
          <br />
          <strong>SIRET :</strong> {placeholder(c.legal_publisher_siret)}
        </p>
        <p className="text-muted-foreground">
          Variables : <code className="font-mono text-xs">LEGAL_PUBLISHER_NAME</code>,{" "}
          <code className="font-mono text-xs">LEGAL_PUBLISHER_ADDRESS</code>,{" "}
          <code className="font-mono text-xs">LEGAL_PUBLISHER_EMAIL</code>,{" "}
          <code className="font-mono text-xs">LEGAL_PUBLISHER_SIRET</code>.
        </p>
      </section>

      <section className="space-y-2 text-sm leading-relaxed">
        <h2 className="font-heading text-xl font-semibold">Hébergement</h2>
        <p>
          <strong>Hébergeur :</strong> {placeholder(c.legal_host_name)}
          <br />
          <strong>Adresse :</strong> {placeholder(c.legal_host_address)}
          <br />
          <strong>Site :</strong> {placeholder(c.legal_host_url)}
        </p>
        <p className="text-muted-foreground">
          Variables : <code className="font-mono text-xs">LEGAL_HOST_NAME</code>,{" "}
          <code className="font-mono text-xs">LEGAL_HOST_ADDRESS</code>,{" "}
          <code className="font-mono text-xs">LEGAL_HOST_URL</code>.
        </p>
      </section>

      <section className="space-y-2 text-sm leading-relaxed">
        <h2 className="font-heading text-xl font-semibold">Objet du service</h2>
        <p>
          Devis Generator : VIBED est une application web permettant de créer des devis, factures,
          factures d’acompte et avoirs, en mode invité (PDF sans conservation serveur) ou avec un
          compte optionnel. Le service est proposé gratuitement ; un lien de don optionnel peut être
          affiché ({c.donate_url ? "configuré" : "non configuré"}).
        </p>
      </section>

      <section className="space-y-2 text-sm leading-relaxed">
        <h2 className="font-heading text-xl font-semibold">Licence</h2>
        <p>
          Le code de la stack VIBED (backend, frontend, documentation associée) est publié sous
          licence Apache-2.0. Le site statique historique présent à la racine du dépôt demeure sous
          GPL-3.0.
        </p>
      </section>

      <section className="space-y-2 text-sm leading-relaxed">
        <h2 className="font-heading text-xl font-semibold">Responsabilité</h2>
        <p>
          Les documents générés et les mentions légales proposés sont des aides à la rédaction. Ils
          ne constituent pas un conseil juridique ou fiscal. L’utilisateur reste responsable de la
          conformité de ses documents et de la vérification des informations (SIRET, TVA, mentions).
        </p>
      </section>
    </article>
  )
}
