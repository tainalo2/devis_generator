import type { CompanyHit } from "@/lib/api"
import type { Address } from "@/lib/types"

export const COUNTRIES = [
  { value: "FR", label: "France" },
  { value: "BE", label: "Belgique" },
  { value: "LU", label: "Luxembourg" },
  { value: "CH", label: "Suisse" },
  { value: "DE", label: "Allemagne" },
  { value: "ES", label: "Espagne" },
  { value: "IT", label: "Italie" },
]

export function applyCompanyHit<T extends { name?: string; legal_name?: string; siren: string; siret: string; vat_number: string; address: Address }>(
  f: T,
  hit: CompanyHit,
  nameKey: "name" | "legal_name",
): T {
  // Émetteur : nom commercial et code APE (NAF) viennent aussi de l'annuaire.
  const issuerExtras =
    nameKey === "legal_name"
      ? { trade_name: hit.trade_name || (f as { trade_name?: string }).trade_name, ape_code: hit.naf || (f as { ape_code?: string }).ape_code }
      : {}
  return {
    ...f,
    ...issuerExtras,
    [nameKey]: hit.name || f[nameKey],
    siren: hit.siren || f.siren,
    siret: hit.siret || f.siret,
    vat_number: hit.vat_number || f.vat_number,
    address: {
      ...f.address,
      line1: hit.address || f.address.line1,
      postal_code: hit.postal_code || f.address.postal_code,
      city: hit.city || f.address.city,
    },
  }
}
