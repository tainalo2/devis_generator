import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"

export function NewDocumentPage() {
  return (
    <section className="space-y-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-semibold tracking-tight">Nouveau document</h1>
        <p className="max-w-2xl text-muted-foreground">
          Squelette : devis, facture, facture d&apos;acompte ou avoir. TVA (franchise 293 B ou taux
          par ligne), numérotation et PDF serveur arriveront ensuite.
        </p>
      </div>

      <Separator />

      <div className="grid gap-4 md:grid-cols-2">
        <label className="space-y-1.5 text-sm">
          <span className="font-medium">Type</span>
          <select
            disabled
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm opacity-70"
            defaultValue="quote"
          >
            <option value="quote">Devis</option>
            <option value="invoice">Facture</option>
            <option value="deposit_invoice">Facture d&apos;acompte</option>
            <option value="credit_note">Avoir</option>
          </select>
        </label>

        <label className="space-y-1.5 text-sm">
          <span className="font-medium">Client</span>
          <input
            disabled
            placeholder="À venir"
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm opacity-70"
          />
        </label>
      </div>

      <Button type="button" disabled>
        Enregistrer (bientôt)
      </Button>
    </section>
  )
}
