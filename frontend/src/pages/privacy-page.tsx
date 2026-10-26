import { useEffect, useState } from "react"
import { Link } from "react-router-dom"

import { LegalLayout } from "@/components/legal-layout"
import { Seo } from "@/components/seo"
import { fetchSiteConfig, placeholder, type SiteConfig } from "@/lib/site-config"

const keyPoints = [
  { title: "Sans compte, rien n’est gardé", text: "Le mode invité génère le PDF sans rien enregistrer." },
  { title: "Données sensibles chiffrées", text: "IBAN et adresses sont chiffrés sur nos serveurs." },
  { title: "Vos données vous suivent", text: "Export JSON et suppression complète en un clic." },
]

export function PrivacyPage() {
  const [cfg, setCfg] = useState<SiteConfig | null>(null)
  useEffect(() => {
    void fetchSiteConfig().then(setCfg)
  }, [])
  const email = placeholder(cfg?.legal_publisher_email ?? "")

  return (
    <>
      <Seo
        title="Politique de confidentialité"
        path="/confidentialite"
        description="Politique de confidentialité de Devis Generator : VIBED : données, conservation, RGPD."
      />
      <LegalLayout
        title="Politique de confidentialité"
        subtitle="Document à faire valider juridiquement. Rédigé d’après le fonctionnement réel de l’application VIBED."
        keyPoints={keyPoints}
        sections={[
          {
            id: "responsable",
            label: "Responsable du traitement",
            content: (
              <p>
                {placeholder(cfg?.legal_publisher_name ?? "")}, contact : {email}. Adresse : {placeholder(cfg?.legal_publisher_address ?? "")}.
              </p>
            ),
          },
          {
            id: "donnees",
            label: "Données collectées",
            content: (
              <ul>
                <li>
                  <strong>Compte (facultatif) :</strong> adresse e-mail, mot de passe (stocké sous forme d’empreinte argon2id), nom
                  d’affichage éventuel, sessions et jeton CSRF.
                </li>
                <li>
                  <strong>Émetteur, clients et documents :</strong> identité professionnelle (SIREN, SIRET, TVA, coordonnées), lignes de
                  facturation, totaux, instantanés figés à l’émission.
                </li>
                <li>
                  <strong>Sources légales, VIES et recherche d’entreprise :</strong> requêtes techniques (requête, horodatage) pour le cache
                  et l’anti-abus ; aucune finalité publicitaire.
                </li>
              </ul>
            ),
          },
          {
            id: "chiffrement",
            label: "Données chiffrées",
            title: "Données sensibles chiffrées",
            content: <p>Les IBAN et les adresses sont chiffrés côté serveur (AES-256-GCM) avant d’être enregistrés.</p>,
          },
          {
            id: "invite",
            label: "Mode invité",
            content: <p>Sans compte, le PDF est généré à la volée : aucun document invité n’est conservé sur nos serveurs.</p>,
          },
          {
            id: "finalites",
            label: "Finalités",
            content: (
              <p>
                Fournir le service de génération de devis et de factures, sécuriser les comptes, respecter les obligations d’information
                (mentions, sources légales) et permettre l’exercice des droits RGPD (export, suppression).
              </p>
            ),
          },
          {
            id: "conservation",
            label: "Durée de conservation",
            content: (
              <p>
                Les données de compte et métier sont conservées <strong>tant que le compte existe</strong>. La suppression du compte efface
                toutes les données associées. Vous devez de votre côté conserver vos factures émises pendant dix ans : exportez-les avant de
                supprimer votre compte. Les journaux techniques de l’hébergeur suivent sa propre politique (
                {placeholder(cfg?.legal_host_name ?? "", "hébergeur à compléter")}).
              </p>
            ),
          },
          {
            id: "cookies",
            label: "Cookies",
            content: (
              <p>
                Cookies strictement nécessaires uniquement : session authentifiée et jeton CSRF. Aucun cookie publicitaire ni de mesure
                d’audience tierce.
              </p>
            ),
          },
          {
            id: "droits",
            label: "Vos droits",
            content: (
              <p>
                Accès, rectification, export et effacement depuis <Link to="/compte">Mon compte</Link> (export JSON, suppression définitive).
                Pour toute autre demande : {email}. Vous pouvez aussi saisir la CNIL.
              </p>
            ),
          },
          {
            id: "tiers",
            label: "Services interrogés",
            title: "Hébergement et services interrogés",
            content: (
              <p>
                L’hébergement est assuré par {placeholder(cfg?.legal_host_name ?? "")}. Pour vérifier une entreprise, un n° de TVA ou un taux,
                l’outil interroge des sources publiques (DINUM, Commission européenne, BCE). Le détail est sur la page{" "}
                <Link to="/sources-legales">Sources légales</Link>.
              </p>
            ),
          },
        ]}
      />
    </>
  )
}
