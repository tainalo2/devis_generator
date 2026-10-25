export type HealthzResponse = {
  status: string
  database: string
  version: string
}

export async function fetchHealthz(): Promise<HealthzResponse> {
  const res = await fetch("/api/v1/healthz")
  if (!res.ok) {
    throw new Error(`healthz HTTP ${res.status}`)
  }
  return (await res.json()) as HealthzResponse
}
