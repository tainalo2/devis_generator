import { Link } from "react-router-dom"

import { Seo } from "@/components/seo"
import { Button } from "@/components/ui/button"

export function NotFoundPage() {
  return (
    <section className="relative flex min-h-[60vh] flex-col items-center justify-center gap-4 overflow-hidden px-4 py-12 text-center md:py-20">
      <Seo title="Page introuvable" path="/404" noIndex description="La page demandée n’existe pas." />
      <div
        aria-hidden
        className="animate-soft-pulse pointer-events-none absolute top-[40%] left-1/2 -mt-40 -ml-40 size-80 rounded-full bg-brand/18 blur-3xl"
      />
      <p aria-hidden className="relative font-heading text-[96px] leading-none font-semibold tracking-[-0.04em] text-foreground/85 md:text-[140px]">
        404
      </p>
      <h1 className="relative font-heading text-2xl leading-[30px] font-semibold tracking-tight">Page introuvable</h1>
      <p className="relative max-w-md text-[15px] leading-[22px] text-muted-foreground">
        Cette page n’existe pas ou a été déplacée. Vos documents, eux, sont toujours là.
      </p>
      <div className="relative mt-2 flex w-full flex-col gap-2.5 sm:w-auto sm:flex-row">
        <Button className="h-11 px-4.5 text-sm" render={<Link to="/" />}>
          Retour à l’accueil
        </Button>
        <Button variant="outline" className="h-11 px-4.5 text-sm" render={<Link to="/documents/nouveau" />}>
          Créer un devis
        </Button>
      </div>
    </section>
  )
}
