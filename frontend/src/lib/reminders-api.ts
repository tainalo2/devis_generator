import { apiFetch } from "@/lib/api"

async function readError(res: Response, fallback: string): Promise<string> {
  const data = (await res.json().catch(() => ({}))) as { error?: string }
  return data.error ?? fallback
}

export type DeliveryMode = "gmail" | "microsoft" | "smtp"

export type ReminderTemplate = {
  id: string
  name: string
  subject: string
  html: string
  created_at: string
  updated_at: string
}

export type Reminder = {
  id: string
  invoice_id: string
  template_id: string
  type: string
  recipient: string
  status: "queued" | "sending" | "sent" | "failed" | "bounced" | "skipped"
  error?: string
  created_at: string
  sent_at?: string
  attempt_count: number
}

// Get the current delivery mode
export async function getReminderDeliveryMode(): Promise<DeliveryMode> {
  const res = await apiFetch("/api/v1/email/delivery-mode")
  if (res.status === 404) return "" as DeliveryMode // Not configured
  if (!res.ok) throw new Error(await readError(res, "Impossible de charger le mode d'envoi."))
  const data = (await res.json()) as { mode: DeliveryMode }
  return data.mode
}

// Set the delivery mode
export async function setReminderDeliveryMode(mode: DeliveryMode): Promise<DeliveryMode> {
  const res = await apiFetch("/api/v1/email/delivery-mode", {
    method: "PUT",
    body: JSON.stringify({ mode }),
  })
  if (!res.ok) throw new Error(await readError(res, "Impossible de configurer le mode d'envoi."))
  const data = (await res.json()) as { mode: DeliveryMode }
  return data.mode
}

// List reminder templates
export async function listReminderTemplates(): Promise<ReminderTemplate[]> {
  const res = await apiFetch("/api/v1/reminder-templates")
  if (!res.ok) throw new Error(await readError(res, "Impossible de charger les modèles de relance."))
  const data = (await res.json()) as { templates: ReminderTemplate[] }
  return data.templates
}

// Create a new reminder template
export async function createReminderTemplate(template: Omit<ReminderTemplate, "id" | "created_at" | "updated_at">): Promise<ReminderTemplate> {
  const res = await apiFetch("/api/v1/reminder-templates", {
    method: "POST",
    body: JSON.stringify(template),
  })
  if (!res.ok) throw new Error(await readError(res, "Impossible de créer le modèle de relance."))
  return (await res.json()) as ReminderTemplate
}

// Update a reminder template
export async function updateReminderTemplate(id: string, template: Omit<ReminderTemplate, "id" | "created_at" | "updated_at">): Promise<ReminderTemplate> {
  const res = await apiFetch(`/api/v1/reminder-templates/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: JSON.stringify(template),
  })
  if (!res.ok) throw new Error(await readError(res, "Impossible de mettre à jour le modèle de relance."))
  return (await res.json()) as ReminderTemplate
}

// Delete a reminder template
export async function deleteReminderTemplate(id: string): Promise<void> {
  const res = await apiFetch(`/api/v1/reminder-templates/${encodeURIComponent(id)}`, {
    method: "DELETE",
  })
  if (!res.ok) throw new Error(await readError(res, "Impossible de supprimer le modèle de relance."))
}
