import { NavLink, Outlet } from "react-router-dom"

import { ThemeToggle } from "@/components/theme-toggle"
import { cn } from "@/lib/utils"

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "rounded-md px-3 py-2 text-sm font-medium transition-colors",
    isActive
      ? "bg-accent text-accent-foreground"
      : "text-muted-foreground hover:bg-accent/60 hover:text-foreground",
  )

export function AppLayout() {
  return (
    <div className="flex min-h-svh flex-col bg-background">
      <header className="border-b border-border">
        <div className="mx-auto flex w-full max-w-5xl items-center justify-between gap-4 px-4 py-3">
          <div className="flex items-center gap-6">
            <NavLink to="/" className="text-base font-semibold tracking-tight text-foreground">
              Devis Generator : VIBED
            </NavLink>

            <nav className="flex items-center gap-1">
              <NavLink to="/" end className={navLinkClass}>
                Accueil
              </NavLink>
              <NavLink to="/documents/nouveau" className={navLinkClass}>
                Nouveau document
              </NavLink>
            </nav>
          </div>
          <ThemeToggle />
        </div>
      </header>
      <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-8">
        <Outlet />
      </main>
    </div>
  )
}
