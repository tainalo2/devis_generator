import { Mail } from "lucide-react"
import { useEffect, useState, type FormEvent } from "react"
import { Link } from "react-router-dom"

import { Button } from "@/components/ui/button"
import { Dialog } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { useToast } from "@/components/use-toast"
import { getDeliveryMode, listReminderTemplates, sendReminder, type ReminderTemplate } from "@/lib/reminders-api"
import { textLinkClass } from "@/lib/styles"

type Props = {
  invoiceId: string
  invoiceName: string
  onClose: () => void
}

/** Relance manuelle d'une facture : choix du modèle, envoi depuis le compte configuré.
 * À monter à l'ouverture seulement : l'état (dont la clé d'idempotence) repart de zéro à chaque ouverture. */
export function SendReminderDialog({ invoiceId, invoiceName, onClose }: Props) {
  const open = true
  const onOpenChange = (next: boolean) => {
    if (!next) onClose()
  }
  const toast = useToast()
  const [templates, setTemplates] = useState<ReminderTemplate[] | null>(null)
  const [hasMode, setHasMode] = useState<boolean | null>(null)
  const [templateId, setTemplateId] = useState("")
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  // Une clé par ouverture : un nouvel essai après échec réseau ne peut pas envoyer deux fois.
  const [idempotencyKey] = useState(() => crypto.randomUUID())

  useEffect(() => {
    let active = true
    void Promise.all([listReminderTemplates(), getDeliveryMode()])
      .then(([list, mode]) => {
        if (!active) return
        setTemplates(list)
        setTemplateId(list[0]?.id ?? "")
        setHasMode(mode !== null)
      })
      .catch((err: unknown) => active && setError(err instanceof Error ? err.message : "Impossible de charger la configuration."))
    return () => {
      active = false
    }
  }, [])

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!templateId || pending) return
    setPending(true)
    setError(null)
    try {
      const reminder = await sendReminder(invoiceId, templateId, idempotencyKey)
      toast.push(reminder.status === "sent" ? `Relance envoyée à ${reminder.recipient}.` : "Relance enregistrée.", "success")
      onOpenChange(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Impossible d’envoyer la relance.")
    } finally {
      setPending(false)
    }
  }

  const ready = templates !== null && hasMode !== null
  const blocked = ready && (templates.length === 0 || !hasMode)
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Envoyer une relance"
      icon={<Mail className="size-5" aria-hidden />}
      description={`Relance de ${invoiceName} par e-mail, depuis votre compte d’envoi.`}
      footer={
        <>
          <Button variant="ghost" type="button" onClick={() => onOpenChange(false)}>
            Annuler
          </Button>
          <Button type="submit" form="send-reminder-form" disabled={!ready || blocked || !templateId || pending}>
            {pending ? "Envoi…" : "Envoyer la relance"}
          </Button>
        </>
      }
    >
      <form id="send-reminder-form" onSubmit={(e) => void submit(e)} className="flex flex-col gap-3">
        {!ready && !error && <p className="text-sm text-muted-foreground">Chargement…</p>}
        {ready && !hasMode && (
          <p className="text-sm">
            Aucun mode d’envoi configuré. <Link className={textLinkClass} to="/compte#s-relances">Configurer l’envoi des relances</Link>
          </p>
        )}
        {ready && templates.length === 0 && (
          <p className="text-sm">
            Aucun modèle de relance. <Link className={textLinkClass} to="/compte#s-relances">Créer un modèle</Link>
          </p>
        )}
        {ready && templates.length > 0 && (
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="reminder-template">Modèle</Label>
            <select
              id="reminder-template"
              value={templateId}
              onChange={(e) => setTemplateId(e.target.value)}
              className="h-9 rounded-md border border-input bg-background px-3 text-sm focus-visible:border-ring"
            >
              {templates.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
          </div>
        )}
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
      </form>
    </Dialog>
  )
}
