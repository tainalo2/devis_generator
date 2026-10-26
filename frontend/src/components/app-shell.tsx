import { Outlet, useLocation } from "react-router-dom"

import { SiteFooter } from "@/components/site-footer"
import { SiteHeader } from "@/components/site-header"
import { SkipLink } from "@/components/public-layout"
import { cn } from "@/lib/utils"

/** L'éditeur et le détail d'un document occupent toute la largeur, sans pied de page. */
function isWorkspace(pathname: string) {
  return /^\/documents\/[^/]+$/.test(pathname)
}

export function AppShell() {
  const { pathname } = useLocation()
  const wide = isWorkspace(pathname)

  return (
    <div className="flex min-h-svh flex-col">
      <SkipLink />
      <SiteHeader />
      <main
        id="contenu"
        className={cn(
          "flex-1",
          wide ? "flex min-h-0 flex-col" : "mx-auto w-full max-w-5xl px-4 pt-5 pb-8 md:px-6 md:pt-8 md:pb-12",
        )}
      >
        <Outlet />
      </main>
      {!wide && <SiteFooter />}
    </div>
  )
}
