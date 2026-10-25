import { useEffect, useState } from "react"
import { Link } from "react-router-dom"

import { Seo } from "@/components/seo"
import { fetchSiteConfig, placeholder, type SiteConfig } from "@/lib/site-config"

export function PrivacyPage() {
  const [cfg, setCfg] = useState<SiteConfig | null>(null)
  useEffect(() => {
    void fetchSiteConfig().then(setCfg)
  }, [])
  const email = placeholder(cfg?.legal_publisher_email ?? "")

  return (
    <article className="mx-auto w-full max-w-3xl space-y-8 px-4 py-12 sm:px-6">
      <Seo
        title="Politique de confidentialité"
        path="/confidentialite"
        description="Politique de confidentialité de Devis Generator : VIBED — données, conservation, RGPD."
      />
      <header className="space-y-2">
        <h1 className="font-heading text-3xl font-semibold tracking-tight">Politique de confidentialité</h1>
        <p className="text-sm text-amber-800 dark:text-amber-200">
          Document à faire valider juridiquement. Rédigé d’après le fonctionnement réel de
          l’application VIBED.
        </p>
      </header>

      <section className="space-y-2 text-sm leading-relaxed">
        <h2 className="font-heading text-xl font-semibold">Responsable du traitement</h2>
        <p>
          {placeholder(cfg?.legal_publisher_name ?? "")} — contact : {email}. Adresse :{" "}
          {placeholder(cfg?.legal_publisher_address ?? "")}.
        </p>
      </section>

      <section className="space-y-2 text-sm leading-relaxed">
        <h2 className="font-heading text-xl font-semibold">Données collectées</h2>
        <ul className="list-disc space-y-1 pl-5">
          <li>
            <strong>Compte (optionnel) :</strong> adresse e-mail, mot de passe (stocké sous forme de
            empreinte argon2id), nom d’affichage éventuel, sessions et jeton CSRF.
          </li>
          <li>
            <strong>Profil émetteur / clients / documents :</strong> identité professionnelle
            (SIREN/SIRET, TVA, coordonnées), lignes de facturation, totaux, snapshots à l’émission.
          </li>
          <li>
            <strong>Données sensibles chiffrées :</strong> IBAN et adresses (AES-256-GCM côté serveur).
          </li>
          <li>
            <strong>Mode invité :</strong> le PDF est généré à la volée ; aucune conservation serveur
            des documents invités.
          </li>
          <li>
            <strong>Sources légales / VIES / recherche d’entreprise :</strong> requêtes techniques
            (requête, horodatage) pour cache et anti-abus ; pas de finalité publicitaire.
          </li>
        </ul>
      </section>

      <section className="space-y-2 text-sm leading-relaxed">
        <h2 className="font-heading text-xl font-semibold">Finalités</h2>
        <p>
          Fournir le service de génération de devis/factures, sécuriser les comptes, respecter les
          obligations d’information (mentions, sources légales), et permettre l’exercice des droits
          RGPD (export / suppression).
        </p>
      </section>

      <section className="space-y-2 text-sm leading-relaxed">
        <h2 className="font-heading text-xl font-semibold">Conservation</h2>
        <p>
          Les données de compte et métier sont conservées <strong>tant que le compte existe</strong>.
          La suppression du compte entraîne l’effacement complet des données associées (cascade
          applicative). Les journaux techniques éventuels de l’hébergeur suivent sa propre politique
          ({placeholder(cfg?.legal_host_name ?? "", "hébergeur à compléter")}).
        </p>
      </section>

      <section className="space-y-2 text-sm leading-relaxed">
        <h2 className="font-heading text-xl font-semibold">Cookies</h2>
        <p>
          Cookies strictement nécessaires uniquement : session authentifiée et jeton CSRF. Pas de
          cookies publicitaires ou de mesure d’audience tierce dans l’application VIBED.
        </p>
      </section>

      <section className="space-y-2 text-sm leading-relaxed">
        <h2 className="font-heading text-xl font-semibold">Vos droits</h2>
        <p>
          Accès, rectification, export et effacement via la page{" "}
          <Link className="underline underline-offset-4" to="/compte">
            Mon compte
          </Link>{" "}
          (export JSON, suppression définitive). Vous pouvez aussi contacter {email}. Vous disposez
          d’un droit de réclamation auprès de la CNIL.
        </p>
      </section>

      <section className="space-y-2 text-sm leading-relaxed">
        <h2 className="font-heading text-xl font-semibold">Transferts / sous-traitants</h2>
        <p>
          L’hébergement est assuré par {placeholder(cfg?.legal_host_name ?? "")}. Des appels sortants
          vers des sources publiques (DINUM, Commission européenne, BCE) peuvent être effectués pour
          la recherche d’entreprise, VIES, TEDB et taux BCE — voir{" "}
          <Link className="underline underline-offset-4" to="/sources-legales">
            Sources légales
          </Link>
          .
        </p>
      </section>
    </article>
  )
}
