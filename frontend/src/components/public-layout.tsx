import { Outlet } from "react-router-dom"

import { SiteFooter } from "@/components/site-footer"
import { SiteHeader } from "@/components/site-header"

export function SkipLink() {
  return (
    <a
      href="#contenu"
      className="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-50 focus:rounded-md focus:bg-card focus:px-3 focus:py-2"
    >
      Aller au contenu
    </a>
  )
}

export function PublicLayout() {
  return (
    <div className="flex min-h-svh flex-col">
      <SkipLink />
      <SiteHeader />
      <main id="contenu" className="flex-1">
        <Outlet />
      </main>
      <SiteFooter />
    </div>
  )
}
