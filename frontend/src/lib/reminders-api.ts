import { apiFetch } from "@/lib/api"

async function readError(res: Response, fallback: string): Promise<string> {
  const data = (await res.json().catch(() => ({}))) as { error?: string }
  return data.error ?? fallback
}

export type DeliveryMode = "gmail" | "microsoft" | "smtp"
export type OAuthProvider = "gmail" | "microsoft"
export type TLSMode = "starttls" | "tls"

export type SMTPConfig = {
  host: string
  port: number
  username: string
  from_address: string
  tls_mode: TLSMode
  password_configured: boolean
}

export type SMTPInput = Omit<SMTPConfig, "password_configured"> & { password: string }

export type ReminderTemplate = {
  id: string
  name: string
  subject: string
  html: string
  created_at: string
  updated_at: string
}

export type TemplateInput = Pick<ReminderTemplate, "name" | "subject" | "html">

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

/** Mode d'envoi choisi ; null tant qu'il n'est pas configuré (le serveur répond 404). */
export async function getDeliveryMode(): Promise<DeliveryMode | null> {
  const res = await apiFetch("/api/v1/email/delivery-mode")
  if (res.status === 404) return null
  if (!res.ok) throw new Error(await readError(res, "Impossible de charger le mode d’envoi."))
  return ((await res.json()) as { mode: DeliveryMode }).mode
}

export async function setDeliveryMode(mode: DeliveryMode): Promise<DeliveryMode> {
  const res = await apiFetch("/api/v1/email/delivery-mode", { method: "PUT", body: JSON.stringify({ mode }) })
  if (!res.ok) throw new Error(await readError(res, "Impossible de configurer le mode d’envoi."))
  return ((await res.json()) as { mode: DeliveryMode }).mode
}

/** Configuration SMTP sans mot de passe ; null si absente. */
export async function getSMTPConfig(): Promise<SMTPConfig | null> {
  const res = await apiFetch("/api/v1/email/smtp")
  if (res.status === 404) return null
  if (!res.ok) throw new Error(await readError(res, "Impossible de charger la configuration SMTP."))
  return ((await res.json()) as { config: SMTPConfig }).config
}

export async function saveSMTPConfig(input: SMTPInput): Promise<SMTPConfig> {
  const res = await apiFetch("/api/v1/email/smtp", { method: "POST", body: JSON.stringify(input) })
  if (!res.ok) throw new Error(await readError(res, "Impossible d’enregistrer la configuration SMTP."))
  return ((await res.json()) as { config: SMTPConfig }).config
}

export async function deleteSMTPConfig(): Promise<void> {
  const res = await apiFetch("/api/v1/email/smtp", { method: "DELETE" })
  if (!res.ok) throw new Error(await readError(res, "Impossible de supprimer la configuration SMTP."))
}

export type OAuthStatus = "connected" | "disconnected" | "unavailable"

/** État de la connexion Gmail/Microsoft ; « unavailable » si le serveur n'a pas de client OAuth (503). */
export async function getOAuthStatus(provider: OAuthProvider): Promise<OAuthStatus> {
  const res = await apiFetch(`/api/v1/email/${provider}/status`)
  if (res.status === 503) return "unavailable"
  if (!res.ok) throw new Error(await readError(res, "Impossible de vérifier la connexion."))
  return ((await res.json()) as { connected: boolean }).connected ? "connected" : "disconnected"
}

export async function startOAuth(provider: OAuthProvider): Promise<string> {
  const res = await apiFetch(`/api/v1/email/${provider}/authorize`)
  if (!res.ok) throw new Error(await readError(res, "Impossible de démarrer la connexion."))
  const url = ((await res.json()) as { authorization_url: string }).authorization_url
  // On ne redirige que vers du https : jamais un schéma javascript: ou data: venu d'une réponse.
  if (!/^https:\/\//i.test(url)) throw new Error("Adresse d’autorisation invalide.")
  return url
}

export async function disconnectOAuth(provider: OAuthProvider): Promise<void> {
  const res = await apiFetch(`/api/v1/email/${provider}`, { method: "DELETE" })
  if (!res.ok) throw new Error(await readError(res, "Impossible de déconnecter le compte."))
}

export async function listReminderTemplates(): Promise<ReminderTemplate[]> {
  const res = await apiFetch("/api/v1/reminder-templates")
  if (!res.ok) throw new Error(await readError(res, "Impossible de charger les modèles de relance."))
  return ((await res.json()) as { templates: ReminderTemplate[] | null }).templates ?? []
}

export async function createReminderTemplate(input: TemplateInput): Promise<ReminderTemplate> {
  const res = await apiFetch("/api/v1/reminder-templates", { method: "POST", body: JSON.stringify(input) })
  if (!res.ok) throw new Error(await readError(res, "Impossible de créer le modèle."))
  return (await res.json()) as ReminderTemplate
}

export async function deleteReminderTemplate(id: string): Promise<void> {
  const res = await apiFetch(`/api/v1/reminder-templates/${encodeURIComponent(id)}`, { method: "DELETE" })
  if (!res.ok) throw new Error(await readError(res, "Impossible de supprimer le modèle."))
}

/** Envoie une relance manuelle pour une facture ; la clé d'idempotence évite un double envoi sur nouvel essai. */
export async function sendReminder(invoiceId: string, templateId: string, idempotencyKey: string): Promise<Reminder> {
  const res = await apiFetch(`/api/v1/documents/${encodeURIComponent(invoiceId)}/reminders`, {
    method: "POST",
    headers: { "Idempotency-Key": idempotencyKey },
    body: JSON.stringify({ invoice_id: invoiceId, template_id: templateId }),
  })
  if (!res.ok) throw new Error(await readError(res, "Impossible d’envoyer la relance."))
  return ((await res.json()) as { reminder: Reminder }).reminder
}

/** Redirection vers le fournisseur OAuth (isolée pour pouvoir être remplacée dans les tests). */
export function redirectTo(url: string) {
  window.location.assign(url)
}
