import { Link, NavLink, Outlet } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { ThemeToggle } from "@/components/theme-toggle"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "rounded-md px-3 py-2 text-sm font-medium transition-colors",
    isActive
      ? "bg-accent text-accent-foreground"
      : "text-muted-foreground hover:bg-accent/60 hover:text-foreground",
  )

export function AppLayout() {
  const { user, loading } = useAuth()

  return (
    <div className="flex min-h-svh flex-col bg-background">
      <header className="border-b border-border">
        <div className="mx-auto flex w-full max-w-5xl flex-wrap items-center justify-between gap-3 px-4 py-3">
          <div className="flex min-w-0 flex-wrap items-center gap-2 sm:gap-4">
            <NavLink
              to="/"
              className="text-sm font-semibold tracking-tight text-foreground sm:text-base"
            >
              Devis Generator : VIBED
            </NavLink>
            <nav className="flex flex-wrap items-center gap-1">
              <NavLink to="/documents/nouveau" className={navLinkClass}>
                Nouveau
              </NavLink>
              {user && (
                <>
                  <NavLink to="/documents" className={navLinkClass}>
                    Documents
                  </NavLink>
                  <NavLink to="/clients" className={navLinkClass}>
                    Clients
                  </NavLink>
                  <NavLink to="/entreprise" className={navLinkClass}>
                    Entreprise
                  </NavLink>
                </>
              )}
              <NavLink to="/sources-legales" className={navLinkClass}>
                Sources légales
              </NavLink>
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
      <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-8">
        <Outlet />
      </main>
    </div>
  )
}
