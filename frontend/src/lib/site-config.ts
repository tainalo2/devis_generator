import { apiFetch } from "@/lib/api"

export type SiteConfig = {
  donate_url: string
  legal_publisher_name: string
  legal_publisher_address: string
  legal_publisher_email: string
  legal_publisher_siret: string
  legal_host_name: string
  legal_host_address: string
  legal_host_url: string
}

const empty: SiteConfig = {
  donate_url: "",
  legal_publisher_name: "",
  legal_publisher_address: "",
  legal_publisher_email: "",
  legal_publisher_siret: "",
  legal_host_name: "",
  legal_host_address: "",
  legal_host_url: "",
}

let cached: SiteConfig | null = null

export async function fetchSiteConfig(): Promise<SiteConfig> {
  if (cached) return cached
  try {
    const res = await apiFetch("/api/v1/public/config")
    if (!res.ok) return empty
    cached = { ...empty, ...((await res.json()) as Partial<SiteConfig>) }
    return cached
  } catch {
    return empty
  }
}

export function placeholder(value: string, label = "à compléter"): string {
  const v = value.trim()
  return v || `« ${label} »`
}
