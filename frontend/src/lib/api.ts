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
  return fetch(input, {
    ...init,
    headers,
    credentials: "include",
  })
}

export async function fetchHealthz(): Promise<HealthzResponse> {
  const res = await fetch("/api/v1/healthz")
  if (!res.ok) {
    throw new Error(`healthz HTTP ${res.status}`)
  }
  return (await res.json()) as HealthzResponse
}

export async function fetchMe(): Promise<User | null> {
  const res = await apiFetch("/api/v1/auth/me")
  if (res.status === 401) return null
  if (!res.ok) throw new Error(`me HTTP ${res.status}`)
  const data = (await res.json()) as { user: User }
  return data.user
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

export async function register(
  email: string,
  password: string,
  displayName: string,
): Promise<User> {
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
  if (!res.ok) {
    const data = (await res.json().catch(() => ({}))) as { error?: string }
    throw new Error(data.error ?? "Export impossible.")
  }
  return res.blob()
}

export async function deleteAccount(): Promise<void> {
  const res = await apiFetch("/api/v1/auth/me", {
    method: "DELETE",
    body: JSON.stringify({ confirm: "SUPPRIMER" }),
  })
  if (!res.ok) {
    const data = (await res.json().catch(() => ({}))) as { error?: string }
    throw new Error(data.error ?? "Suppression impossible.")
  }
}

export async function requestPasswordReset(email: string): Promise<string> {
  const res = await apiFetch("/api/v1/auth/password-reset/request", {
    method: "POST",
    body: JSON.stringify({ email }),
  })
  const data = (await res.json()) as { message?: string; error?: string }
  if (!res.ok) throw new Error(data.error ?? "Demande impossible.")
  return data.message ?? "Demande enregistrée."
}
