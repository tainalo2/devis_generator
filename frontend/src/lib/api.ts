import type { Client, Document, IssuerProfile, TotalsPreview, DocLine } from "@/lib/types"

export type HealthzResponse = {
  status: string
  database?: string
  version: string
}

export type User = {
  id: string
  email: string
  display_name: string
  created_at: string
}

function csrfToken(): string | null {
  const match = document.cookie.match(/(?:^|;\s*)vibed_csrf=([^;]+)/)
  return match ? decodeURIComponent(match[1]) : null
}

export async function apiFetch(input: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers)
  if (!headers.has("Content-Type") && init.body) {
    headers.set("Content-Type", "application/json")
  }
  const method = (init.method ?? "GET").toUpperCase()
  if (method !== "GET" && method !== "HEAD") {
    const csrf = csrfToken()
    if (csrf) headers.set("X-CSRF-Token", csrf)
  }
  return fetch(input, { ...init, headers, credentials: "include" })
}

async function readError(res: Response, fallback: string): Promise<string> {
  const data = (await res.json().catch(() => ({}))) as { error?: string }
  return data.error ?? fallback
}

export async function fetchHealthz(): Promise<HealthzResponse> {
  const res = await fetch("/api/v1/healthz")
  if (!res.ok) throw new Error(`healthz HTTP ${res.status}`)
  return (await res.json()) as HealthzResponse
}

export async function fetchMe(): Promise<User | null> {
  const res = await apiFetch("/api/v1/auth/me")
  if (res.status === 401) return null
  if (!res.ok) throw new Error(`me HTTP ${res.status}`)
  return ((await res.json()) as { user: User }).user
}

export async function login(email: string, password: string): Promise<User> {
  const res = await apiFetch("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  })
  const data = (await res.json()) as { user?: User; error?: string }
  if (!res.ok) throw new Error(data.error ?? "Connexion impossible.")
  return data.user!
}

export async function register(email: string, password: string, displayName: string): Promise<User> {
  const res = await apiFetch("/api/v1/auth/register", {
    method: "POST",
    body: JSON.stringify({ email, password, display_name: displayName }),
  })
  const data = (await res.json()) as { user?: User; error?: string }
  if (!res.ok) throw new Error(data.error ?? "Inscription impossible.")
  return data.user!
}

export async function logout(): Promise<void> {
  await apiFetch("/api/v1/auth/logout", { method: "POST", body: "{}" })
}

export async function exportAccount(): Promise<Blob> {
  const res = await apiFetch("/api/v1/auth/export")
  if (!res.ok) throw new Error(await readError(res, "Export impossible."))
  return res.blob()
}

export async function deleteAccount(): Promise<void> {
  const res = await apiFetch("/api/v1/auth/me", {
    method: "DELETE",
    body: JSON.stringify({ confirm: "SUPPRIMER" }),
  })
  if (!res.ok) throw new Error(await readError(res, "Suppression impossible."))
}

export async function listClients(): Promise<Client[]> {
  const res = await apiFetch("/api/v1/clients")
  if (!res.ok) throw new Error(await readError(res, "Chargement clients impossible."))
  return ((await res.json()) as { clients: Client[] }).clients
}

export async function saveClient(input: Partial<Client> & { name: string }, id?: string): Promise<Client> {
  const res = await apiFetch(id ? `/api/v1/clients/${id}` : "/api/v1/clients", {
    method: id ? "PUT" : "POST",
    body: JSON.stringify(input),
  })
  if (!res.ok) throw new Error(await readError(res, "Enregistrement client impossible."))
  return ((await res.json()) as { client: Client }).client
}

export async function deleteClient(id: string): Promise<void> {
  const res = await apiFetch(`/api/v1/clients/${id}`, { method: "DELETE" })
  if (!res.ok) throw new Error(await readError(res, "Suppression client impossible."))
}

export async function listDocuments(): Promise<Document[]> {
  const res = await apiFetch("/api/v1/documents")
  if (!res.ok) throw new Error(await readError(res, "Chargement documents impossible."))
  return ((await res.json()) as { documents: Document[] }).documents
}

export async function getDocument(id: string): Promise<Document> {
  const res = await apiFetch(`/api/v1/documents/${id}`)
  if (!res.ok) throw new Error(await readError(res, "Document introuvable."))
  return ((await res.json()) as { document: Document }).document
}

export async function saveDocument(body: unknown, id?: string): Promise<Document> {
  const res = await apiFetch(id ? `/api/v1/documents/${id}` : "/api/v1/documents", {
    method: id ? "PUT" : "POST",
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(await readError(res, "Enregistrement document impossible."))
  return ((await res.json()) as { document: Document }).document
}

export async function issueDocument(id: string): Promise<Document> {
  const res = await apiFetch(`/api/v1/documents/${id}/issue`, { method: "POST", body: "{}" })
  if (!res.ok) throw new Error(await readError(res, "Émission impossible."))
  return ((await res.json()) as { document: Document }).document
}

export async function downloadDocumentPDF(id: string): Promise<Blob> {
  const res = await apiFetch(`/api/v1/documents/${id}/pdf`)
  if (!res.ok) throw new Error(await readError(res, "PDF impossible."))
  return res.blob()
}

export async function previewTotals(vatRegime: string, lines: DocLine[]): Promise<TotalsPreview> {
  const res = await apiFetch("/api/v1/billing/preview", {
    method: "POST",
    body: JSON.stringify({ vat_regime: vatRegime, lines }),
  })
  if (!res.ok) throw new Error(await readError(res, "Calcul impossible."))
  const data = (await res.json()) as {
    subtotal_cents: number
    tax_cents: number
    total_cents: number
    buckets: { tax_rate_bps: number; ht_cents: number; tax_cents: number }[]
  }
  return {
    subtotal_cents: data.subtotal_cents,
    tax_cents: data.tax_cents,
    total_cents: data.total_cents,
    buckets: data.buckets,
  }
}

export async function guestPDF(payload: unknown): Promise<Blob> {
  const res = await apiFetch("/api/v1/guest/pdf", {
    method: "POST",
    body: JSON.stringify(payload),
  })
  if (!res.ok) throw new Error(await readError(res, "PDF invité impossible."))
  return res.blob()
}

export async function getIssuer(): Promise<IssuerProfile> {
  const res = await apiFetch("/api/v1/issuer")
  if (!res.ok) throw new Error(await readError(res, "Profil émetteur impossible."))
  return ((await res.json()) as { issuer: IssuerProfile }).issuer
}

export async function saveIssuer(input: IssuerProfile): Promise<IssuerProfile> {
  const res = await apiFetch("/api/v1/issuer", { method: "PUT", body: JSON.stringify(input) })
  if (!res.ok) throw new Error(await readError(res, "Enregistrement émetteur impossible."))
  return ((await res.json()) as { issuer: IssuerProfile }).issuer
}

export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

export type CompanyHit = {
  siren: string
  siret: string
  name: string
  trade_name?: string
  address?: string
  postal_code?: string
  city?: string
  naf?: string
  vat_number?: string
}

export type VATValidation = {
  country_code: string
  vat_number: string
  valid: boolean
  name?: string
  address?: string
  checked_at: string
  source: string
  cached: boolean
}

export type LegalStatusEntry = {
  key: string
  value: unknown
  source: string
  fetched_at: string
  status: string
  fresh: boolean
  age_seconds: number
}

export type LegalStatusResponse = {
  variables: LegalStatusEntry[]
  sources: { id: string; label: string; auth: string }[]
  excluded: string[]
}

export async function searchCompanies(q: string, limit = 8): Promise<CompanyHit[]> {
  const res = await apiFetch(
    `/api/v1/legal/companies/search?q=${encodeURIComponent(q)}&limit=${limit}`,
  )
  if (!res.ok) throw new Error(await readError(res, "Recherche d'entreprise impossible."))
  return ((await res.json()) as { results: CompanyHit[] }).results ?? []
}

export async function validateVAT(vatNumber: string): Promise<VATValidation> {
  const res = await apiFetch("/api/v1/legal/vat/validate", {
    method: "POST",
    body: JSON.stringify({ vat_number: vatNumber }),
  })
  if (!res.ok) throw new Error(await readError(res, "Validation TVA impossible."))
  return (await res.json()) as VATValidation
}

export async function fetchLegalStatus(): Promise<LegalStatusResponse> {
  const res = await apiFetch("/api/v1/legal/status")
  if (!res.ok) throw new Error(await readError(res, "Statut légal indisponible."))
  return (await res.json()) as LegalStatusResponse
}
