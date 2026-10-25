import { Menu, X } from "lucide-react"
import { useState } from "react"
import { Link, NavLink, Outlet } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { SiteFooter } from "@/components/site-footer"
import { ThemeToggle } from "@/components/theme-toggle"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

const appLinks = [
  { to: "/tableau-de-bord", label: "Tableau de bord", auth: true },
  { to: "/documents", label: "Documents", auth: true },
  { to: "/documents/nouveau", label: "Nouveau", auth: false },
  { to: "/clients", label: "Clients", auth: true },
  { to: "/entreprise", label: "Entreprise", auth: true },
  { to: "/sources-legales", label: "Sources légales", auth: false },
]

const navClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "block rounded-md px-3 py-2 text-sm font-medium transition-colors",
    isActive
      ? "bg-sidebar-accent text-sidebar-accent-foreground"
      : "text-sidebar-foreground/80 hover:bg-sidebar-accent/70 hover:text-sidebar-foreground",
  )

export function AppShell() {
  const { user, loading } = useAuth()
  const [open, setOpen] = useState(false)
  const links = appLinks.filter((l) => !l.auth || !!user)

  return (
    <div className="flex min-h-svh flex-col">
      <a
        href="#contenu"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-md focus:bg-card focus:px-3 focus:py-2"
      >
        Aller au contenu
      </a>
      <div className="flex flex-1">
        <aside
          className={cn(
            "fixed inset-y-0 left-0 z-40 flex w-64 flex-col border-r border-sidebar-border bg-sidebar transition-transform md:static md:translate-x-0",
            open ? "translate-x-0" : "-translate-x-full",
          )}
          aria-label="Navigation principale"
        >
          <div className="flex items-center justify-between gap-2 border-b border-sidebar-border px-4 py-4">
            <Link to="/" className="font-heading text-base font-semibold tracking-tight" onClick={() => setOpen(false)}>
              Devis Generator : VIBED
            </Link>
            <Button
              variant="ghost"
              size="icon-sm"
              className="md:hidden"
              aria-label="Fermer le menu"
              onClick={() => setOpen(false)}
            >
              <X />
            </Button>
          </div>
          <nav className="flex flex-1 flex-col gap-1 p-3">
            {links.map((l) => (
              <NavLink key={l.to} to={l.to} className={navClass} onClick={() => setOpen(false)}>
                {l.label}
              </NavLink>
            ))}
          </nav>
          <div className="border-t border-sidebar-border p-3">
            {!loading && user ? (
              <Button variant="outline" size="sm" className="w-full" render={<Link to="/compte" onClick={() => setOpen(false)} />}>
                Mon compte
              </Button>
            ) : !loading ? (
              <div className="flex flex-col gap-2">
                <Button size="sm" className="w-full" render={<Link to="/connexion" onClick={() => setOpen(false)} />}>
                  Connexion
                </Button>
                <Button variant="ghost" size="sm" className="w-full" render={<Link to="/inscription" onClick={() => setOpen(false)} />}>
                  Inscription
                </Button>
              </div>
            ) : null}
          </div>
        </aside>

        {open && (
          <button
            type="button"
            className="fixed inset-0 z-30 bg-foreground/30 md:hidden"
            aria-label="Fermer le menu"
            onClick={() => setOpen(false)}
          />
        )}

        <div className="flex min-w-0 flex-1 flex-col">
          <header className="sticky top-0 z-20 flex items-center justify-between gap-3 border-b border-border bg-background/85 px-4 py-3 backdrop-blur md:px-6">
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="icon-sm"
                className="md:hidden"
                aria-label="Ouvrir le menu"
                aria-expanded={open}
                onClick={() => setOpen(true)}
              >
                <Menu />
              </Button>
              <p className="text-sm text-muted-foreground md:hidden">Menu</p>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="ghost" size="sm" render={<Link to="/" />}>
                Accueil
              </Button>
              <ThemeToggle />
            </div>
          </header>
          <main id="contenu" className="mx-auto w-full max-w-5xl flex-1 px-4 py-8 md:px-6">
            <Outlet />
          </main>
          <SiteFooter />
        </div>
      </div>
    </div>
  )
}
