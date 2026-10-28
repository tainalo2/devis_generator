import { useEffect, useState } from "react"
import { Link } from "react-router-dom"

import { LogoMark } from "@/components/logo"
import { fetchSiteConfig, type SiteConfig } from "@/lib/site-config"

export function SiteFooter() {
  const [cfg, setCfg] = useState<SiteConfig | null>(null)

  useEffect(() => {
    void fetchSiteConfig().then(setCfg)
  }, [])

  const donate = cfg?.donate_url?.trim()

  return (
    <footer className="border-t border-border bg-card/40">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 px-4 py-10 sm:px-6 lg:px-8">
        <div className="flex flex-col justify-between gap-6 md:flex-row md:items-start">
          <div className="max-w-sm space-y-2">
            <div className="flex items-center gap-2.5">
              <LogoMark />
              <p className="font-heading text-lg font-semibold tracking-tight">Devis Generator : VIBED</p>
            </div>
            <p className="text-sm text-muted-foreground">
              Outil gratuit pour micro-entrepreneurs, indépendants et petites sociétés. Financé par
              des dons, sans prestataire de paiement intégré.
            </p>
            {donate ? (
              <p className="text-sm">
                <a
                  href={donate}
                  className="text-brand underline-offset-4 hover:underline"
                  rel="noopener noreferrer"
                  target="_blank"
                >
                  Soutenir le projet
                </a>
              </p>
            ) : (
              <p className="text-sm text-muted-foreground">
                Lien de don : configurable via <code className="font-mono text-xs">DONATE_URL</code>.
              </p>
            )}
          </div>
          <nav aria-label="Pied de page" className="grid grid-cols-2 gap-x-10 gap-y-2 text-sm">
            <Link className="text-muted-foreground hover:text-foreground" to="/documents/nouveau">
              Créer un devis
            </Link>
            <Link className="text-muted-foreground hover:text-foreground" to="/sources-legales">
              Sources légales
            </Link>
            <Link className="text-muted-foreground hover:text-foreground" to="/mentions-legales">
              Mentions légales
            </Link>
            <Link className="text-muted-foreground hover:text-foreground" to="/confidentialite">
              Confidentialité
            </Link>
            <Link className="text-muted-foreground hover:text-foreground" to="/connexion">
              Connexion
            </Link>
            <a
              className="text-muted-foreground hover:text-foreground"
              href="https://www.apache.org/licenses/LICENSE-2.0"
              rel="noopener noreferrer"
              target="_blank"
            >
              Licence Apache-2.0
            </a>
          </nav>
        </div>
        <p className="text-xs text-muted-foreground">
          Code VIBED sous Apache-2.0. Le site statique historique à la racine du dépôt reste sous
          GPL-3.0.
        </p>
      </div>
    </footer>
  )
}
