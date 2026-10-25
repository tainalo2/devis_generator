import { Link, NavLink, Outlet } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { SiteFooter } from "@/components/site-footer"
import { ThemeToggle } from "@/components/theme-toggle"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

const linkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "rounded-md px-2.5 py-1.5 text-sm transition-colors",
    isActive ? "text-foreground" : "text-muted-foreground hover:text-foreground",
  )

export function PublicLayout() {
  const { user, loading } = useAuth()

  return (
    <div className="flex min-h-svh flex-col">
      <a
        href="#contenu"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-md focus:bg-card focus:px-3 focus:py-2"
      >
        Aller au contenu
      </a>
      <header className="border-b border-border/80 bg-background/80 backdrop-blur">
        <div className="mx-auto flex w-full max-w-6xl flex-wrap items-center justify-between gap-3 px-4 py-3 sm:px-6 lg:px-8">
          <div className="flex min-w-0 flex-wrap items-center gap-4">
            <Link to="/" className="font-heading text-base font-semibold tracking-tight sm:text-lg">
              Devis Generator : VIBED
            </Link>
            <nav className="hidden items-center gap-1 sm:flex" aria-label="Navigation publique">
              <NavLink to="/documents/nouveau" className={linkClass}>
                Créer un devis
              </NavLink>
              <NavLink to="/sources-legales" className={linkClass}>
                Sources
              </NavLink>
              {user && (
                <NavLink to="/tableau-de-bord" className={linkClass}>
                  Tableau de bord
                </NavLink>
              )}
            </nav>
          </div>
          <div className="flex items-center gap-2">
            {!loading && user ? (
              <Button variant="outline" size="sm" render={<Link to="/compte" />}>
                Mon compte
              </Button>
            ) : !loading ? (
              <>
                <Button variant="ghost" size="sm" render={<Link to="/connexion" />}>
                  Connexion
                </Button>
                <Button size="sm" render={<Link to="/inscription" />}>
                  Inscription
                </Button>
              </>
            ) : null}
            <ThemeToggle />
          </div>
        </div>
      </header>
      <main id="contenu" className="flex-1">
        <Outlet />
      </main>
      <SiteFooter />
    </div>
  )
}
