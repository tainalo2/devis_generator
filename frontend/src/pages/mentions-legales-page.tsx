import { useEffect, useState } from "react"
import { Link } from "react-router-dom"

import { LegalLayout } from "@/components/legal-layout"
import { Seo } from "@/components/seo"
import { fetchSiteConfig, placeholder, type SiteConfig } from "@/lib/site-config"

export function MentionsLegalesPage() {
  const [cfg, setCfg] = useState<SiteConfig | null>(null)
  useEffect(() => {
    void fetchSiteConfig().then(setCfg)
  }, [])
  const v = (key: keyof SiteConfig) => placeholder(cfg?.[key] ?? "")

  return (
    <>
      <Seo title="Mentions légales" path="/mentions-legales" description="Mentions légales de Devis Generator : VIBED : éditeur, hébergeur, licence." />
      <LegalLayout
        title="Mentions légales"
        subtitle="Texte à faire valider juridiquement. Les champs éditeur et hébergeur restent marqués « à compléter » tant que la configuration du serveur ne les renseigne pas."
        sections={[
          {
            id: "editeur",
            label: "Éditeur du site",
            content: (
              <>
                <dl>
                  <dt>Nom ou raison sociale</dt>
                  <dd>{v("legal_publisher_name")}</dd>
                  <dt>Adresse</dt>
                  <dd>{v("legal_publisher_address")}</dd>
                  <dt>E-mail de contact</dt>
                  <dd>{v("legal_publisher_email")}</dd>
                  <dt>SIRET</dt>
                  <dd>{v("legal_publisher_siret")}</dd>
                </dl>
                <p>
                  Ces valeurs viennent de la configuration du serveur : <code>LEGAL_PUBLISHER_NAME</code>, <code>LEGAL_PUBLISHER_ADDRESS</code>,{" "}
                  <code>LEGAL_PUBLISHER_EMAIL</code>, <code>LEGAL_PUBLISHER_SIRET</code>.
                </p>
              </>
            ),
          },
          {
            id: "hebergement",
            label: "Hébergement",
            content: (
              <>
                <dl>
                  <dt>Hébergeur</dt>
                  <dd>{v("legal_host_name")}</dd>
                  <dt>Adresse</dt>
                  <dd>{v("legal_host_address")}</dd>
                  <dt>Site</dt>
                  <dd>{v("legal_host_url")}</dd>
                </dl>
                <p>
                  Variables : <code>LEGAL_HOST_NAME</code>, <code>LEGAL_HOST_ADDRESS</code>, <code>LEGAL_HOST_URL</code>.
                </p>
              </>
            ),
          },
          {
            id: "objet",
            label: "Objet du service",
            content: (
              <p>
                Devis Generator : VIBED est un outil gratuit de création de devis, factures, factures d’acompte et avoirs pour les
                micro-entrepreneurs, indépendants et petites sociétés. Il s’utilise sans compte (PDF sans conservation serveur) ; un compte
                facultatif conserve vos brouillons, clients et documents. Un lien de don optionnel peut être affiché (
                {cfg?.donate_url ? "configuré" : "non configuré"}).
              </p>
            ),
          },
          {
            id: "licences",
            label: "Propriété intellectuelle",
            title: "Propriété intellectuelle et licences",
            content: (
              <p>
                Le code de l’application VIBED (backend, frontend, documentation associée) est publié sous licence Apache-2.0. Le site statique
                historique, à la racine du dépôt, reste sous licence GPL-3.0. Les documents que vous créez vous appartiennent.
              </p>
            ),
          },
          {
            id: "responsabilite",
            label: "Responsabilité",
            content: (
              <>
                <p>
                  Les documents générés et les mentions proposées sont des aides à la rédaction. Ils ne constituent pas un conseil juridique ou
                  fiscal. Vous restez responsable de la conformité de vos documents et de la vérification des informations (SIRET, TVA,
                  mentions).
                </p>
                <p>
                  Les taux et contrôles proviennent de sources officielles détaillées sur la page <Link to="/sources-legales">Sources légales</Link>.
                </p>
              </>
            ),
          },
          {
            id: "contact",
            label: "Contact",
            content: <p>Pour toute question sur ces mentions : {v("legal_publisher_email")}.</p>,
          },
        ]}
      />
    </>
  )
}
