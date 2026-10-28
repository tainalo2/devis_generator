import { Building2, FileText, GripVertical, List, Plus, Scale, Trash2, User, type LucideIcon } from "lucide-react"
import { useState, type ReactNode } from "react"

import { Button } from "@/components/ui/button"
import { DecimalInput } from "@/components/ui/decimal-input"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select } from "@/components/ui/select"
import { formatDate } from "@/lib/documents"
import { VAT_RATES, emptyLine, formatEUR, type Client, type DocLine, type Document, type IssuerProfile } from "@/lib/types"
import { cn } from "@/lib/utils"

import { addressLines, docHT, lineHT, type EditorValue } from "./model"

type Tone = "infos" | "parties" | "lines" | "terms"

const toneClass: Record<Tone, { box: string; icon: string }> = {
  infos: { box: "bg-zone-infos border-zone-infos-border", icon: "bg-zone-infos-icon" },
  parties: { box: "bg-zone-parties border-zone-parties-border", icon: "bg-zone-parties-icon" },
  lines: { box: "bg-zone-lines border-zone-lines-border", icon: "bg-zone-lines-icon" },
  terms: { box: "bg-zone-terms border-zone-terms-border", icon: "bg-zone-terms-icon" },
}

export function ZoneIcon({ icon: Icon, tone, className }: { icon: LucideIcon; tone: Tone; className?: string }) {
  return (
    <span aria-hidden className={cn("inline-flex size-6 shrink-0 items-center justify-center rounded-md text-white", toneClass[tone].icon, className)}>
      <Icon className="size-3.5" />
    </span>
  )
}

function ZoneTitle({ id, icon, tone, children }: { id?: string; icon: LucideIcon; tone: Tone; children: ReactNode }) {
  return (
    <h2 id={id} className="flex items-center gap-2 font-heading text-base font-semibold tracking-tight">
      <ZoneIcon icon={icon} tone={tone} />
      {children}
    </h2>
  )
}

const caption = "text-xs font-medium text-muted-foreground"
const inlineLink =
  "rounded-sm px-1 py-1.5 text-xs font-medium text-brand underline-offset-4 hover:underline disabled:pointer-events-none disabled:opacity-50"

type Patch = (p: Partial<EditorValue>) => void

/* ---------- Informations ---------- */

export function InfosZone({
  value,
  patch,
  quotes,
  depositAmount,
}: {
  value: EditorValue
  patch: Patch
  /** Devis émis disponibles pour une facture d'acompte. */
  quotes: Document[]
  depositAmount: number
}) {
  const isDeposit = value.kind === "deposit_invoice"
  return (
    <section aria-labelledby="infos-titre" className={cn("flex flex-col gap-3.5 rounded-xl border p-4", toneClass.infos.box)}>
      <div className="flex flex-wrap items-baseline justify-between gap-3">
        <ZoneTitle id="infos-titre" icon={FileText} tone="infos">
          Informations
        </ZoneTitle>
        <span className="text-xs text-muted-foreground">Numéro attribué à l’émission</span>
      </div>

      {isDeposit && (
        <div className="grid items-end gap-3 @lg:grid-cols-[minmax(0,1fr)_auto]">
          <Label>
            <span>Devis d’origine</span>
            <Select value={value.related_document_id} onChange={(e) => patch({ related_document_id: e.target.value })}>
              <option value="">Choisir un devis émis</option>
              {quotes.map((q) => (
                <option key={q.id} value={q.id}>
                  {q.number} · {formatDate(q.issue_date)}
                  {q.title ? ` · ${q.title}` : ""}
                </option>
              ))}
            </Select>
          </Label>
          <Label>
            <span>Acompte</span>
            <span className="flex items-center gap-2">
              <DecimalInput
                aria-label="Pourcentage d’acompte"
                className="w-20"
                scale={1}
                suffix="%"
                value={value.deposit_percent}
                onValueChange={(n) => patch({ deposit_percent: Math.min(100, Math.max(1, n)) })}
              />
              <span className="text-[13px] font-normal whitespace-nowrap text-muted-foreground">soit {formatEUR(depositAmount)} HT</span>
            </span>
          </Label>
        </div>
      )}
      {isDeposit && quotes.length === 0 && (
        <p className="text-xs text-muted-foreground">Émettez d’abord un devis : l’acompte s’y rattache.</p>
      )}

      <Label>
        <span>Objet</span>
        <Input value={value.title} placeholder="Refonte du site vitrine" onChange={(e) => patch({ title: e.target.value })} />
      </Label>
      <div className="grid grid-cols-2 gap-3">
        <Label>
          <span>Date d’émission</span>
          <Input type="date" value={value.issue_date} onChange={(e) => patch({ issue_date: e.target.value })} />
        </Label>
        {value.kind !== "credit_note" && (
          <Label>
            <span>{value.kind === "quote" ? "Valable jusqu’au" : "Date d’échéance"}</span>
            <Input type="date" value={value.due_date} onChange={(e) => patch({ due_date: e.target.value })} />
          </Label>
        )}
      </div>
    </section>
  )
}

/* ---------- Émetteur et client ---------- */

function PartyText({ name, lines, empty }: { name: string; lines: string[]; empty: string }) {
  if (!name) return <span className="text-[13px] text-muted-foreground">{empty}</span>
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-sm font-semibold">{name}</span>
      {lines.map((l) => (
        <span key={l} className="text-[13px] text-muted-foreground">
          {l}
        </span>
      ))}
    </div>
  )
}

export function PartiesZone({
  issuer,
  client,
  clients,
  value,
  patch,
  guest,
  onEditIssuer,
  onEditClient,
}: {
  issuer: IssuerProfile
  client: Client | null
  clients: Client[]
  value: EditorValue
  patch: Patch
  guest: boolean
  onEditIssuer: () => void
  onEditClient: () => void
}) {
  const issuerLines = [...addressLines(issuer.address), issuer.siret ? `SIRET ${issuer.siret}` : ""].filter(Boolean)
  const clientLines = client
    ? [...addressLines(client.address), client.siren ? `SIREN ${client.siren}` : "", client.email].filter(Boolean)
    : []
  const lockedClient = value.kind === "deposit_invoice" || value.kind === "credit_note"

  return (
    <section aria-label="Émetteur et client" className={cn("grid overflow-hidden rounded-xl border @lg:grid-cols-2", toneClass.parties.box)}>
      <div className="flex flex-col gap-2.5 border-b border-zone-parties-border p-4 @lg:border-r @lg:border-b-0">
        <div className="flex items-center justify-between gap-2">
          <ZoneTitle icon={Building2} tone="parties">
            Émetteur
          </ZoneTitle>
          <button type="button" className={inlineLink} aria-haspopup="dialog" onClick={onEditIssuer}>
            {issuer.legal_name ? "Modifier" : "Renseigner"}
          </button>
        </div>
        <PartyText name={issuer.legal_name} lines={issuerLines} empty="Vos nom, adresse et SIRET apparaîtront en tête du document." />
        <span className="text-xs text-muted-foreground">{guest ? "Conservé dans ce navigateur uniquement" : "Repris de la fiche Entreprise"}</span>
      </div>
      <div className="flex flex-col gap-2.5 p-4">
        <div className="flex items-center justify-between gap-2">
          <ZoneTitle icon={User} tone="parties">
            Client
          </ZoneTitle>
          {!lockedClient && (
            <button type="button" className={inlineLink} aria-haspopup="dialog" onClick={onEditClient}>
              {guest ? (client?.name ? "Modifier" : "+ Saisir le client") : "+ Nouveau client"}
            </button>
          )}
        </div>
        {!guest && (
          <label>
            <span className="sr-only">Choisir un client</span>
            <Select value={value.client_id} disabled={lockedClient} onChange={(e) => patch({ client_id: e.target.value })}>
              <option value="">Aucun client</option>
              {clients.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </Select>
          </label>
        )}
        {guest ? (
          <PartyText name={client?.name ?? ""} lines={clientLines} empty="Aucun client saisi." />
        ) : (
          clientLines.length > 0 && (
            <div className="flex flex-col gap-0.5">
              {clientLines.map((l) => (
                <span key={l} className="text-[13px] text-muted-foreground">
                  {l}
                </span>
              ))}
            </div>
          )
        )}
        {lockedClient && <span className="text-xs text-muted-foreground">Repris du document d’origine</span>}
      </div>
    </section>
  )
}

/* ---------- Lignes ---------- */

function move<T>(list: T[], from: number, to: number) {
  const next = [...list]
  const [item] = next.splice(from, 1)
  next.splice(to, 0, item)
  return next
}

export function LinesZone({
  value,
  patch,
  quote,
  depositLines = [],
}: {
  value: EditorValue
  patch: Patch
  /** Devis d'origine (facture d'acompte). */
  quote?: Document
  /** Lignes calculées depuis le devis (facture d'acompte). */
  depositLines?: DocLine[]
}) {
  const [armed, setArmed] = useState<number | null>(null)
  const [dragFrom, setDragFrom] = useState<number | null>(null)
  const franchise = value.vat_regime === "franchise_293b"
  const lines = value.lines
  const count = `${lines.length} ligne${lines.length > 1 ? "s" : ""}`

  const updateLine = (i: number, p: Partial<DocLine>) => patch({ lines: lines.map((l, idx) => (idx === i ? { ...l, ...p } : l)) })

  if (value.kind === "deposit_invoice") {
    return (
      <section aria-labelledby="lignes-titre" className={cn("flex flex-col gap-2.5 rounded-xl border p-4", toneClass.lines.box)}>
        <div className="flex items-baseline justify-between gap-3">
          <ZoneTitle id="lignes-titre" icon={List} tone="lines">
            Lignes
          </ZoneTitle>
          <span className={caption}>
            {depositLines.length} ligne{depositLines.length > 1 ? "s" : ""}
          </span>
        </div>
        {quote ? (
          depositLines.map((l, i) => (
            <div key={i} className="flex items-center justify-between gap-3 rounded-lg border border-zone-lines-item bg-background px-3 py-3.5">
              <div className="flex min-w-0 flex-col gap-0.5">
                <span className="text-sm font-medium">{l.description}</span>
                <span className="text-xs text-muted-foreground">
                  Calculée depuis le devis d’origine ({formatEUR(docHT(quote.lines ?? []))} HT)
                </span>
              </div>
              <span className="text-sm font-semibold whitespace-nowrap tabular-nums">{formatEUR(lineHT(l))}</span>
            </div>
          ))
        ) : (
          <p className="text-sm text-muted-foreground">Choisissez le devis d’origine dans Informations.</p>
        )}
        <p className="text-xs text-muted-foreground">Pour changer le montant, modifiez le pourcentage dans Informations.</p>
      </section>
    )
  }

  return (
    <section aria-labelledby="lignes-titre" className={cn("flex flex-col gap-2.5 rounded-xl border p-4", toneClass.lines.box)}>
      <div className="flex items-baseline justify-between gap-3">
        <ZoneTitle id="lignes-titre" icon={List} tone="lines">
          Lignes
        </ZoneTitle>
        <span className={caption}>{count}</span>
      </div>

      <div aria-hidden className="hidden gap-2 px-3 @lg:flex">
        <span className="w-4 shrink-0" />
        <div className="grid flex-1 grid-cols-4 gap-2">
          <span className={caption}>Quantité</span>
          <span className={caption}>Prix unitaire HT</span>
          <span className={caption}>TVA</span>
          <span className={cn(caption, "text-right")}>Total HT</span>
        </div>
        <span className="w-7 shrink-0" />
      </div>

      {lines.map((line, i) => (
        <div
          key={line.id ?? `l-${i}`}
          draggable={armed === i}
          onDragStart={(e) => {
            setDragFrom(i)
            e.dataTransfer.effectAllowed = "move"
          }}
          onDragOver={(e) => {
            if (dragFrom === null) return
            e.preventDefault()
            if (dragFrom !== i) {
              patch({ lines: move(lines, dragFrom, i) })
              setDragFrom(i)
            }
          }}
          onDragEnd={() => {
            setDragFrom(null)
            setArmed(null)
          }}
          className={cn(
            "flex items-start gap-2 rounded-lg border border-zone-lines-item bg-background p-3",
            dragFrom === i && "opacity-60",
          )}
        >
          <button
            type="button"
            className="mt-2.5 -ml-1 flex w-5 shrink-0 cursor-grab justify-center rounded-sm text-muted-foreground active:cursor-grabbing"
            aria-label={`Déplacer la ligne ${i + 1} (flèches haut et bas)`}
            onPointerDown={() => setArmed(i)}
            onPointerUp={() => setArmed(null)}
            onKeyDown={(e) => {
              if (e.key === "ArrowUp" && i > 0) {
                e.preventDefault()
                patch({ lines: move(lines, i, i - 1) })
              } else if (e.key === "ArrowDown" && i < lines.length - 1) {
                e.preventDefault()
                patch({ lines: move(lines, i, i + 1) })
              }
            }}
          >
            <GripVertical className="size-4" />
          </button>
          <div className="flex min-w-0 flex-1 flex-col gap-2">
            <Input
              aria-label={`Désignation de la ligne ${i + 1}`}
              placeholder="Désignation"
              value={line.description}
              onChange={(e) => updateLine(i, { description: e.target.value })}
            />
            <div className="grid grid-cols-2 items-end gap-2 @lg:grid-cols-4">
              <label className="flex flex-col gap-1">
                <span className={cn(caption, "@lg:sr-only")}>Quantité</span>
                <DecimalInput
                  aria-label={undefined}
                  scale={1000}
                  value={line.quantity_milli}
                  onValueChange={(n) => updateLine(i, { quantity_milli: n })}
                />
              </label>
              <label className="flex flex-col gap-1">
                <span className={cn(caption, "@lg:sr-only")}>Prix unitaire HT</span>
                <DecimalInput
                  scale={100}
                  decimals={2}
                  suffix="€"
                  value={line.unit_price_cents}
                  onValueChange={(n) => updateLine(i, { unit_price_cents: n })}
                />
              </label>
              <label className="flex flex-col gap-1">
                <span className={cn(caption, "@lg:sr-only")}>TVA</span>
                <Select
                  value={franchise ? "" : line.tax_rate_bps}
                  disabled={franchise}
                  title={franchise ? "Franchise en base : TVA non applicable" : undefined}
                  onChange={(e) => updateLine(i, { tax_rate_bps: Number(e.target.value) })}
                >
                  {franchise && <option value="">Franchise</option>}
                  {VAT_RATES.map((r) => (
                    <option key={r.bps} value={r.bps}>
                      {r.label}
                    </option>
                  ))}
                </Select>
              </label>
              <div className="flex min-h-9 flex-col items-end justify-end gap-1">
                <span className={cn(caption, "@lg:hidden")}>Total HT</span>
                <span className="text-sm leading-9 font-semibold tabular-nums">{formatEUR(lineHT(line))}</span>
              </div>
            </div>
          </div>
          <Button
            variant="ghost"
            size="icon-sm"
            className="mt-1"
            aria-label={`Supprimer la ligne ${i + 1}`}
            disabled={lines.length === 1}
            onClick={() => patch({ lines: lines.filter((_, idx) => idx !== i) })}
          >
            <Trash2 />
          </Button>
        </div>
      ))}

      <Button
        variant="outline"
        className="self-start"
        onClick={() => patch({ lines: [...lines, { ...emptyLine(), tax_rate_bps: lines.at(-1)?.tax_rate_bps ?? 0 }] })}
      >
        <Plus />
        Ajouter une ligne
      </Button>
    </section>
  )
}

/* ---------- Conditions et notes ---------- */

export function TermsZone({ value, patch, lockRegime }: { value: EditorValue; patch: Patch; lockRegime?: boolean }) {
  const invoice = value.kind !== "quote"
  const franchise = value.vat_regime === "franchise_293b"
  const legal = invoice
    ? `Ajoutées au document : ${franchise ? "« TVA non applicable, art. 293 B du CGI », " : ""}les pénalités de retard, l’indemnité forfaitaire de 40 € pour frais de recouvrement et l’absence d’escompte pour paiement anticipé.`
    : franchise
      ? "La mention « TVA non applicable, art. 293 B du CGI » est ajoutée au document."
      : "La TVA est calculée par ligne et détaillée par taux sur le document."

  return (
    <section aria-labelledby="conditions-titre" className={cn("flex flex-col gap-3.5 rounded-xl border p-4", toneClass.terms.box)}>
      <ZoneTitle id="conditions-titre" icon={Scale} tone="terms">
        Conditions et notes
      </ZoneTitle>
      <div className="grid gap-3 @lg:grid-cols-2">
        <Label>
          <span>Régime de TVA</span>
          <Select value={value.vat_regime} disabled={lockRegime} onChange={(e) => patch({ vat_regime: e.target.value })}>
            <option value="franchise_293b">Franchise en base (art. 293 B)</option>
            <option value="standard">TVA par ligne</option>
          </Select>
        </Label>
        <Label>
          <span>Conditions de paiement</span>
          <Input value={value.payment_terms} onChange={(e) => patch({ payment_terms: e.target.value })} />
        </Label>
      </div>
      <Label>
        <span>Notes</span>
        <textarea
          className="flex min-h-19 w-full resize-y rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:border-ring"
          placeholder="Visibles sur le document, sous les lignes"
          value={value.notes}
          onChange={(e) => patch({ notes: e.target.value })}
        />
      </Label>
      <p className="text-xs text-muted-foreground">{legal}</p>
    </section>
  )
}
