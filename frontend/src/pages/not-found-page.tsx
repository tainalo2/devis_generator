import { Link } from "react-router-dom"

import { Seo } from "@/components/seo"
import { Button } from "@/components/ui/button"

export function NotFoundPage() {
  return (
    <section className="mx-auto flex min-h-[60vh] w-full max-w-xl flex-col justify-center gap-4 px-4 py-16">
      <Seo title="Page introuvable" path="/404" noIndex description="La page demandée n’existe pas." />
      <p className="text-sm font-medium text-brand">404</p>
      <h1 className="font-heading text-3xl font-semibold tracking-tight">Page introuvable</h1>
      <p className="text-muted-foreground">
        Cette adresse ne correspond à aucun contenu de Devis Generator : VIBED.
      </p>
      <div className="flex flex-wrap gap-2">
        <Button render={<Link to="/" />}>Retour à l’accueil</Button>
        <Button variant="outline" render={<Link to="/documents/nouveau" />}>
          Créer un devis
        </Button>
      </div>
    </section>
  )
}
