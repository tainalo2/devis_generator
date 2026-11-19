import { Plus, Trash2 } from "lucide-react"
import { useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Dialog } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Segmented } from "@/components/ui/segmented"
import { useToast } from "@/components/use-toast"
import {
  createReminderTemplate,
  deleteReminderTemplate,
  getReminderDeliveryMode,
  listReminderTemplates,
  setReminderDeliveryMode,
  type DeliveryMode,
  type ReminderTemplate,
} from "@/lib/reminders-api"
import { formatDate } from "@/lib/documents"
import { cn } from "@/lib/utils"

export function RemindersSection() {
  const toast = useToast()
  const [deliveryMode, setDeliveryMode] = useState<DeliveryMode | null>(null)
  const [deliveryModeLoading, setDeliveryModeLoading] = useState(true)
  const [templates, setTemplates] = useState<ReminderTemplate[]>([])
  const [templatesLoading, setTemplatesLoading] = useState(true)
  const [templateDialogOpen, setTemplateDialogOpen] = useState(false)
  const [templateName, setTemplateName] = useState("")
  const [templateSubject, setTemplateSubject] = useState("")
  const [templateHtml, setTemplateHtml] = useState("")
  const [templatePending, setTemplatePending] = useState(false)

  // Load delivery mode and templates on mount
  useEffect(() => {
    let active = true
    void (async () => {
      try {
        const mode = await getReminderDeliveryMode()
        if (active) setDeliveryMode(mode)
      } catch (err) {
        if (active) toast.push(err instanceof Error ? err.message : "Impossible de charger la configuration.", "error")
      } finally {
        if (active) setDeliveryModeLoading(false)
      }
    })()
    return () => {
      active = false
    }
  }, [toast])

  // Load templates on mount
  useEffect(() => {
    let active = true
    void (async () => {
      try {
        const list = await listReminderTemplates()
        if (active) setTemplates(list)
      } catch (err) {
        if (active) toast.push(err instanceof Error ? err.message : "Impossible de charger les modèles.", "error")
      } finally {
        if (active) setTemplatesLoading(false)
      }
    })()
    return () => {
      active = false
    }
  }, [toast])

  async function handleDeliveryModeChange(mode: DeliveryMode) {
    try {
      await setReminderDeliveryMode(mode)
      setDeliveryMode(mode)
      toast.push("Mode d'envoi configuré.", "success")
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Erreur lors de la configuration.", "error")
    }
  }

  async function handleCreateTemplate() {
    const name = templateName.trim()
    if (!name || !templateSubject.trim() || !templateHtml.trim()) {
      toast.push("Tous les champs sont obligatoires.", "error")
      return
    }
    setTemplatePending(true)
    try {
      const created = await createReminderTemplate({
        name,
        subject: templateSubject,
        html: templateHtml,
      })
      setTemplates((prev) => [created, ...prev])
      setTemplateDialogOpen(false)
      setTemplateName("")
      setTemplateSubject("")
      setTemplateHtml("")
      toast.push("Modèle de relance créé.", "success")
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Erreur lors de la création.", "error")
    } finally {
      setTemplatePending(false)
    }
  }

  async function handleDeleteTemplate(id: string) {
    setTemplatePending(true)
    try {
      await deleteReminderTemplate(id)
      setTemplates((prev) => prev.filter((t) => t.id !== id))
      toast.push("Modèle de relance supprimé.", "success")
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Erreur lors de la suppression.", "error")
    } finally {
      setTemplatePending(false)
    }
  }

  return (
    <section aria-labelledby="s-reminders" className="grid gap-x-8 gap-y-3 border-t border-border py-6 md:grid-cols-[240px_minmax(0,1fr)]">
      <div className="flex flex-col gap-1">
        <h2 id="s-reminders" className="font-heading text-lg leading-[26px] font-semibold tracking-tight">
          Relances automatiques
        </h2>
        <p className="text-[13px] leading-[18px] text-muted-foreground">
          Configurez l'envoi des rappels de paiement.
        </p>
      </div>
      <div className="flex flex-col gap-3.5 rounded-xl border border-border bg-card p-4">
        {/* Delivery Mode Selection */}
        {deliveryModeLoading ? (
          <p className="text-sm text-muted-foreground">Chargement de la configuration…</p>
        ) : (
          <div className="space-y-3">
            <div>
              <Label htmlFor="delivery-mode" className="text-sm font-medium">
                Mode d'envoi
              </Label>
              <Segmented
                label="Mode d'envoi"
                value={deliveryMode || ""}
                onChange={(value) => void handleDeliveryModeChange(value as DeliveryMode)}
                options={[
                  { value: "gmail", label: "Gmail" },
                  { value: "microsoft", label: "Outlook" },
                  { value: "smtp", label: "SMTP personnalisé" },
                ]}
              />
              <p className="text-xs text-muted-foreground mt-1">
                Choisissez comment les relances seront envoyées.
              </p>
            </div>

            {/* Templates Section */}
            <div className="border-t border-border pt-4">
              <div className="flex items-center justify-between mb-3">
                <Label htmlFor="templates" className="text-sm font-medium">
                  Modèles de relance
                </Label>
                <Button
                  size="sm"
                  onClick={() => setTemplateDialogOpen(true)}
                  disabled={templatePending}
                >
                  <Plus className="size-4" />
                  Nouveau modèle
                </Button>
              </div>

              {templatesLoading ? (
                <p className="text-sm text-muted-foreground">Chargement des modèles…</p>
              ) : templates.length === 0 ? (
                <p className="rounded-lg border border-dashed border-border p-3 text-sm text-muted-foreground">
                  Aucun modèle de relance créé.
                </p>
              ) : (
                <ul className="divide-y divide-border rounded-lg border border-border">
                  {templates.map((template) => (
                    <li key={template.id} className="flex items-center justify-between gap-3 p-3">
                      <div className="min-w-0 flex-1">
                        <div className="font-medium text-sm">{template.name}</div>
                        <div className="text-xs text-muted-foreground">{template.subject}</div>
                        <p className="text-xs text-muted-foreground">
                          Créé le {formatDate(template.created_at)}
                        </p>
                      </div>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={templatePending}
                        onClick={() => void handleDeleteTemplate(template.id)}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Template Dialog */}
      <Dialog
        open={templateDialogOpen}
        onOpenChange={(open) => {
          if (!open && templatePending) return
          setTemplateDialogOpen(open)
          if (!open) {
            setTemplateName("")
            setTemplateSubject("")
            setTemplateHtml("")
          }
        }}
        title="Créer un modèle de relance"
        description="Définissez le sujet et le contenu HTML du modèle de relance."
      >
        <div className="space-y-3">
          <Label>
            <span>Nom du modèle</span>
            <Input
              autoFocus
              value={templateName}
              maxLength={200}
              placeholder="Ex. Rappel 7 jours après"
              onChange={(e) => setTemplateName(e.target.value)}
            />
          </Label>
          <Label>
            <span>Sujet de l'e-mail</span>
            <Input
              value={templateSubject}
              maxLength={200}
              placeholder="Rappel de paiement"
              onChange={(e) => setTemplateSubject(e.target.value)}
            />
          </Label>
          <Label>
            <span>Contenu HTML</span>
            <textarea
              className={cn(
                "flex min-h-[120px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm",
                "placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              )}
              value={templateHtml}
              placeholder="Contenu HTML du modèle…"
              onChange={(e) => setTemplateHtml(e.target.value)}
            />
          </Label>
          <p className="text-xs text-muted-foreground">
            Variables disponibles: {`{invoice_number}, {client_name}, {amount}, {due_date}`}
          </p>
          <div className="flex justify-end gap-2">
            <Button
              variant="outline"
              disabled={templatePending}
              onClick={() => {
                setTemplateDialogOpen(false)
                setTemplateName("")
                setTemplateSubject("")
                setTemplateHtml("")
              }}
            >
              Annuler
            </Button>
            <Button disabled={templatePending || !templateName.trim()} onClick={() => void handleCreateTemplate()}>
              Créer le modèle
            </Button>
          </div>
        </div>
      </Dialog>
    </section>
  )
}
