export type Address = {
  line1: string
  line2: string
  postal_code: string
  city: string
  country: string
}

export type Client = {
  id: string
  name: string
  email: string
  address: Address
  siren: string
  siret: string
  vat_number: string
  iban?: string
  notes: string
  created_at?: string
}

export type ClientSnapshot = Pick<Client, "name" | "email" | "address" | "siren" | "siret" | "vat_number">

export type DocLine = {
  id?: string
  position: number
  description: string
  quantity_milli: number
  unit_price_cents: number
  discount_bps: number
  tax_rate_bps: number
  tax_cents?: number
  total_cents?: number
}

export type Document = {
  id: string
  client_id: string | null
  kind: string
  status: string
  number: string | null
  title: string
  issue_date: string | null
  due_date: string | null
  currency: string
  vat_regime: string
  notes: string
  payment_terms: string
  related_document_id: string | null
  subtotal_cents: number
  tax_cents: number
  total_cents: number
  issued_at?: string | null
  sent_at?: string | null
  accepted_at?: string | null
  paid_at?: string | null
  created_at?: string
  updated_at?: string
  legal_mentions?: string
  /** Émetteur figé à l'émission. */
  issuer_snapshot?: IssuerProfile | null
  /** Client figé à l'émission (fait foi même si la fiche change ou disparaît). */
  client_snapshot?: ClientSnapshot | null
  /** Nom du client figé, fourni par la liste des documents. */
  client_name?: string
  lines: DocLine[]
}

export type IssuerProfile = {
  legal_name: string
  trade_name: string
  email: string
  phone: string
  siret: string
  siren: string
  vat_number: string
  ape_code: string
  rcs: string
  address: Address
  iban?: string
  default_vat_regime: string
  payment_terms: string
}

export type TotalsPreview = {
  subtotal_cents: number
  tax_cents: number
  total_cents: number
  buckets: { tax_rate_bps: number; ht_cents: number; tax_cents: number }[]
}

export const VAT_RATES = [
  { bps: 0, label: "0 %" },
  { bps: 210, label: "2,1 %" },
  { bps: 550, label: "5,5 %" },
  { bps: 1000, label: "10 %" },
  { bps: 2000, label: "20 %" },
] as const

export const DOC_KINDS = [
  { value: "quote", label: "Devis" },
  { value: "invoice", label: "Facture" },
  { value: "deposit_invoice", label: "Facture d’acompte" },
  { value: "credit_note", label: "Avoir" },
] as const

export function emptyAddress(): Address {
  return { line1: "", line2: "", postal_code: "", city: "", country: "FR" }
}

export function emptyLine(): DocLine {
  return {
    position: 0,
    description: "",
    quantity_milli: 1000,
    unit_price_cents: 0,
    discount_bps: 0,
    tax_rate_bps: 0,
  }
}

const eur = new Intl.NumberFormat("fr-FR", { style: "currency", currency: "EUR" })

/** Montant en euros, séparateur de milliers et vrai signe moins (−). */
export function formatEUR(cents: number): string {
  const s = eur.format(Math.abs(cents) / 100)
  return cents < 0 ? `\u2212${s}` : s
}
