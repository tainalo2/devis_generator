import { useEffect, useState } from "react"
import { Navigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { PageHeader } from "@/components/page"
import { Seo } from "@/components/seo"
import { useToast } from "@/components/toast-provider"
import { Button } from "@/components/ui/button"
import { getAdminPaymentFeatures, updateAdminPaymentFeatures, type PaymentFeatures } from "@/lib/api"

const modes: { key: Extract<keyof PaymentFeatures, string>; title: string; description: string }[] = [
  { key: "rib", title: "RIB / IBAN", description: "Affiche les coordonnées bancaires pour le virement." },
  { key: "byo_provider", title: "Prestataire du propriétaire", description: "Autorise un compte Stripe, Mollie ou PayPal propre au propriétaire." },
  { key: "stripe_connect", title: "Stripe Connect plateforme", description: "Autorise les paiements traités par la plateforme via Stripe Connect." },
]

export function AdminPaymentFeaturesPage() {
  const { user, loading } = useAuth()
  const toast = useToast()
  const [features, setFeatures] = useState<PaymentFeatures | null>(null)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!user?.is_admin) return
    void getAdminPaymentFeatures()
      .then(setFeatures)
      .catch((err) => toast.push(err instanceof Error ? err.message : "Configuration indisponible.", "error"))
  }, [toast, user?.is_admin])

  if (!loading && !user?.is_admin) return <Navigate to="/" replace />

  async function toggle(key: keyof PaymentFeatures) {
    if (!features) return
    const next = { ...features, [key]: !features[key] }
    setSaving(true)
    try {
      setFeatures(await updateAdminPaymentFeatures(next))
      toast.push("Modes de paiement mis à jour.", "success")
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Mise à jour impossible.", "error")
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-208 flex-col gap-5">
      <Seo title="Administration des paiements" path="/administration/paiements" noIndex />
      <PageHeader title="Modes de paiement" description="Chaque intégration est désactivée par défaut et doit être activée explicitement." />
      <div className="flex flex-col divide-y divide-border rounded-xl border border-border bg-card">
        {modes.map((mode) => (
          <section key={mode.key} className="flex flex-wrap items-center justify-between gap-4 p-5">
            <div className="max-w-xl">
              <h2 className="font-heading text-lg font-semibold">{mode.title}</h2>
              <p className="text-sm text-muted-foreground">{mode.description}</p>
            </div>
            <Button
              variant={features?.[mode.key] ? "default" : "outline"}
              disabled={!features || saving}
              aria-pressed={features?.[mode.key] ?? false}
              onClick={() => void toggle(mode.key)}
            >
              {features?.[mode.key] ? "Activé" : "Désactivé"}
            </Button>
          </section>
        ))}
      </div>
    </div>
  )
}
