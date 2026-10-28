import { useEffect, useState, type FormEvent, type ReactNode } from "react"
import { Navigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { CompanyAutocomplete } from "@/components/company-autocomplete"
import { emptyIssuer } from "@/components/editor/model"
import { PageHeader } from "@/components/page"
import { AddressFields } from "@/components/party-fields"
import { applyCompanyHit } from "@/lib/party"
import { Seo } from "@/components/seo"
import { useToast } from "@/components/toast-provider"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select } from "@/components/ui/select"
import { Spinner } from "@/components/ui/spinner"
import { VATValidator } from "@/components/vat-validator"
import { getIssuer, listDocuments, saveIssuer } from "@/lib/api"
import { useMediaQuery } from "@/lib/hooks"
import type { Document, IssuerProfile } from "@/lib/types"
import { cn } from "@/lib/utils"

const sections = [
  { id: "identite", label: "Identité légale", tone: "bg-zone-parties border-zone-parties-border", dot: "bg-zone-parties-icon" },
  { id: "coordonnees", label: "Coordonnées", tone: "bg-zone-infos border-zone-infos-border", dot: "bg-zone-infos-icon" },
  { id: "banque", label: "Banque", tone: "bg-zone-terms border-zone-terms-border", dot: "bg-zone-terms-icon" },
  { id: "defauts", label: "Valeurs par défaut", tone: "bg-zone-lines border-zone-lines-border", dot: "bg-zone-lines-icon" },
] as const

const PAYMENT_TERMS = ["Paiement à réception", "Paiement à 30 jours", "Paiement à 45 jours fin de mois", "Paiement à 60 jours"]

const numbering = [
  { kind: "quote", label: "Prochain devis", prefix: "D" },
  { kind: "invoice", label: "Prochaine facture", prefix: "F" },
  { kind: "deposit_invoice", label: "Prochain acompte", prefix: "FA" },
  { kind: "credit_note", label: "Prochain avoir", prefix: "A" },
]

/** Prochain numéro par type pour l'année en cours (même format que billing.FormatNumber). */
function nextNumber(docs: Document[], kind: string, prefix: string) {
  const year = new Date().getFullYear()
  const re = new RegExp(`^${prefix}-${year}-(\\d+)$`)
  const max = docs.filter((d) => d.kind === kind).reduce((m, d) => Math.max(m, Number(re.exec(d.number ?? "")?.[1] ?? 0)), 0)
  return `${prefix}-${year}-${String(max + 1).padStart(4, "0")}`
}

function Section({ id, title, tone, children }: { id: string; title: string; tone: string; children: ReactNode }) {
  return (
    <section id={id} aria-labelledby={`t-${id}`} className={cn("flex scroll-mt-24 flex-col gap-3.5 rounded-xl border p-5", tone)}>
      <h2 id={`t-${id}`} className="font-heading text-lg leading-[26px] font-semibold tracking-tight">
        {title}
      </h2>
      {children}
    </section>
  )
}

const g2 = "grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))]"

export function IssuerPage() {
  const { user, loading } = useAuth()
  const toast = useToast()
  const desktop = useMediaQuery("(min-width: 768px)")
  const [form, setForm] = useState<IssuerProfile | null>(null)
  const [docs, setDocs] = useState<Document[]>([])
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const [current, setCurrent] = useState<string>(sections[0].id)

  useEffect(() => {
    if (!user) return
    void getIssuer()
      .then((p) => setForm({ ...emptyIssuer(), ...p }))
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erreur"))
    void listDocuments().then(setDocs).catch(() => undefined)
  }, [user])

  // Section visible → entrée active du sommaire.
  useEffect(() => {
    if (!form) return
    const io = new IntersectionObserver(
      (entries) => {
        const top = entries.filter((e) => e.isIntersecting).sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)[0]
        if (top) setCurrent(top.target.id)
      },
      { rootMargin: "-80px 0px -60% 0px" },
    )
    sections.forEach((s) => {
      const el = document.getElementById(s.id)
      if (el) io.observe(el)
    })
    return () => io.disconnect()
  }, [form])

  if (!loading && !user) return <Navigate to="/connexion" replace />
  if (!form) return error ? <p className="text-sm text-destructive">{error}</p> : <Spinner label="Chargement de la fiche Entreprise" />

  const set = (p: Partial<IssuerProfile>) => setForm((f) => (f ? { ...f, ...p } : f))

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!form) return
    setPending(true)
    try {
      setForm({ ...emptyIssuer(), ...(await saveIssuer(form)) })
      toast.push("Fiche Entreprise enregistrée.", "success")
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Enregistrement impossible.", "error")
    } finally {
      setPending(false)
    }
  }

  const terms = PAYMENT_TERMS.includes(form.payment_terms) || !form.payment_terms ? PAYMENT_TERMS : [form.payment_terms, ...PAYMENT_TERMS]

  return (
    <form onSubmit={(e) => void onSubmit(e)} className="flex flex-col gap-6">
      <Seo title="Entreprise" path="/entreprise" noIndex />
      <PageHeader
        title="Entreprise"
        description={<span className="block max-w-xl">Ces informations figurent sur tous vos devis et factures. Les documents déjà émis ne changent pas.</span>}
        actions={
          desktop && (
            <Button type="submit" disabled={pending}>
              Enregistrer les modifications
            </Button>
          )
        }
      />

      <div className="flex flex-col items-start gap-4 md:flex-row md:gap-8">
        <nav
          aria-label="Sections de la page"
          className="sticky top-15 z-[2] -mx-4 flex w-[calc(100%+2rem)] shrink-0 gap-1 overflow-x-auto bg-background px-4 py-2 md:top-24 md:mx-0 md:w-55 md:flex-col md:p-0"
        >
          {sections.map((s) => (
            <a
              key={s.id}
              href={`#${s.id}`}
              aria-current={current === s.id ? "true" : undefined}
              onClick={() => setCurrent(s.id)}
              className={cn(
                "flex items-center gap-2 rounded-md px-2.5 py-2 text-sm font-medium whitespace-nowrap",
                current === s.id ? "bg-secondary text-foreground" : "text-muted-foreground hover:bg-muted hover:text-foreground",
              )}
            >
              <i className={cn("size-2 shrink-0 rounded-full", s.dot)} aria-hidden />
              {s.label}
            </a>
          ))}
        </nav>

        <div className="flex w-full max-w-190 min-w-0 flex-1 flex-col gap-4">
          <Section id="identite" title="Identité légale" tone={sections[0].tone}>
            <CompanyAutocomplete label="Retrouver votre entreprise" onPick={(hit) => setForm((f) => (f ? applyCompanyHit(f, hit, "legal_name") : f))} />
            <Label>
              <span>Raison sociale ou nom de l’entrepreneur</span>
              <Input required value={form.legal_name} onChange={(e) => set({ legal_name: e.target.value })} />
            </Label>
            <Label>
              <span>
                Nom commercial <span className="font-normal text-muted-foreground">(facultatif)</span>
              </span>
              <Input value={form.trade_name} onChange={(e) => set({ trade_name: e.target.value })} />
            </Label>
            <div className={g2}>
              <Label>
                <span>SIRET</span>
                <Input inputMode="numeric" placeholder="14 chiffres" value={form.siret} onChange={(e) => set({ siret: e.target.value })} />
              </Label>
              <Label>
                <span>Code APE</span>
                <Input placeholder="0000A" value={form.ape_code} onChange={(e) => set({ ape_code: e.target.value })} />
              </Label>
            </div>
            <div className={g2}>
              <Label>
                <span>SIREN</span>
                <Input inputMode="numeric" placeholder="9 chiffres" value={form.siren} onChange={(e) => set({ siren: e.target.value })} />
              </Label>
              <Label>
                <span>RCS ou RM</span>
                <Input placeholder="Dispensé d’immatriculation" value={form.rcs} onChange={(e) => set({ rcs: e.target.value })} />
              </Label>
            </div>
            <VATValidator
              value={form.vat_number}
              onChange={(vat_number) => set({ vat_number })}
              placeholder={form.default_vat_regime === "franchise_293b" ? "Non applicable en franchise" : "FR 00 000000000"}
            />
          </Section>

          <Section id="coordonnees" title="Coordonnées" tone={sections[1].tone}>
            <AddressFields value={form.address} onChange={(address) => set({ address })} />
            <div className={g2}>
              <Label>
                <span>E-mail</span>
                <Input type="email" placeholder="nom@exemple.fr" value={form.email} onChange={(e) => set({ email: e.target.value })} />
              </Label>
              <Label>
                <span>Téléphone</span>
                <Input type="tel" placeholder="06 00 00 00 00" value={form.phone} onChange={(e) => set({ phone: e.target.value })} />
              </Label>
            </div>
          </Section>

          <Section id="banque" title="Banque" tone={sections[2].tone}>
            <Label>
              <span>IBAN</span>
              <Input placeholder="FR76 0000 0000 0000 0000 0000 000" value={form.iban ?? ""} onChange={(e) => set({ iban: e.target.value })} />
            </Label>
            <p className="text-xs text-muted-foreground">L’IBAN est chiffré sur nos serveurs et figure sur les factures, jamais sur les devis.</p>
          </Section>

          <Section id="defauts" title="Valeurs par défaut des documents" tone={sections[3].tone}>
            <div className={g2}>
              <Label>
                <span>Régime de TVA</span>
                <Select value={form.default_vat_regime} onChange={(e) => set({ default_vat_regime: e.target.value })}>
                  <option value="franchise_293b">Franchise en base (art. 293 B)</option>
                  <option value="standard">TVA par ligne</option>
                </Select>
              </Label>
              <Label>
                <span>Conditions de paiement des factures</span>
                <Select value={form.payment_terms || PAYMENT_TERMS[1]} onChange={(e) => set({ payment_terms: e.target.value })}>
                  {terms.map((t) => (
                    <option key={t} value={t}>
                      {t}
                    </option>
                  ))}
                </Select>
              </Label>
            </div>
            <div className="flex flex-col gap-2">
              <span className="text-sm font-medium">Numérotation</span>
              <div className="flex flex-wrap gap-2">
                {numbering.map((n) => (
                  <span key={n.kind} className="inline-flex flex-col gap-0.5 rounded-lg border border-border bg-card px-3 py-2">
                    <span className="text-xs text-muted-foreground">{n.label}</span>
                    <span className="text-sm font-semibold tabular-nums">{nextNumber(docs, n.kind, n.prefix)}</span>
                  </span>
                ))}
              </div>
              <p className="text-xs text-muted-foreground">
                Numérotation continue par type et par année. Elle ne peut pas être réinitialisée ni modifiée à la main.
              </p>
            </div>
          </Section>
        </div>
      </div>

      {!desktop && (
        <div className="sticky bottom-0 z-[5] -mx-4 -mb-8 flex h-19 items-center border-t border-border bg-card px-4 shadow-[0_-8px_16px_-12px_rgb(15_40_60/0.25)]">
          <Button type="submit" className="h-11 flex-1 text-sm" disabled={pending}>
            Enregistrer les modifications
          </Button>
        </div>
      )}
    </form>
  )
}
