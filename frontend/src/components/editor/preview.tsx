import { Expand, Maximize, Minus, PanelLeftClose, PanelLeftOpen, Plus } from "lucide-react"
import { useEffect, useLayoutEffect, useRef, useState } from "react"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { formatDate } from "@/lib/documents"
import { formatEUR, type Client, type DocLine, type IssuerProfile, type TotalsPreview } from "@/lib/types"
import { cn } from "@/lib/utils"

import { addressLines, lineHT, type EditorKind } from "./model"

export type PreviewData = {
  kind: EditorKind
  number: string | null
  title: string
  issue_date: string
  due_date: string
  vat_regime: string
  payment_terms: string
  notes: string
  lines: DocLine[]
  issuer: IssuerProfile
  client: Client | null
  totals: TotalsPreview | null
  /** Facture d'acompte : devis d'origine et reste à facturer. */
  deposit?: { quoteNumber: string | null; quoteDate: string | null; quoteHT: number; percent: number; remaining: number }
  /** Avoir : facture corrigée. */
  relatedNumber?: string | null
}

const PAGE_W = 794
const PAGE_H = 1123

const docTitles: Record<EditorKind, string> = {
  quote: "DEVIS",
  invoice: "FACTURE",
  deposit_invoice: "FACTURE D’ACOMPTE",
  credit_note: "AVOIR",
}

const qty = (milli: number) => (milli / 1000).toLocaleString("fr-FR", { maximumFractionDigits: 3 })

/** Feuille A4 rendue en HTML : couleurs fixes, comme le papier, quel que soit le thème. */
function Sheet({ d }: { d: PreviewData }) {
  const franchise = d.vat_regime === "franchise_293b"
  const isQuote = d.kind === "quote"
  const t = d.totals
  const issuerName = d.issuer.legal_name || "[Votre raison sociale]"
  const iban = d.issuer.iban?.trim()

  const meta = [
    `Date : ${formatDate(d.issue_date)}`,
    d.kind !== "credit_note" && d.due_date ? `${isQuote ? "Valable jusqu’au" : "Échéance"} : ${formatDate(d.due_date)}` : "",
    d.deposit?.quoteNumber ? `Devis ${d.deposit.quoteNumber}` : "",
    d.relatedNumber ? `Sur la facture ${d.relatedNumber}` : "",
  ].filter(Boolean)

  const mentions = [
    franchise ? "TVA non applicable, art. 293 B du CGI." : "",
    d.deposit ? `Reste à facturer après cet acompte : ${formatEUR(d.deposit.remaining)}.` : "",
    d.payment_terms
      ? `${d.payment_terms}${!isQuote && d.due_date ? `, au plus tard le ${formatDate(d.due_date)}` : ""}.${!isQuote && iban ? ` IBAN : ${iban}` : ""}`
      : "",
    !isQuote
      ? "En cas de retard : pénalités à trois fois le taux d’intérêt légal et indemnité forfaitaire de 40 € pour frais de recouvrement. Pas d’escompte pour paiement anticipé."
      : "",
  ].filter(Boolean)

  const finalLabel = { quote: "Total TTC", invoice: "Net à payer", deposit_invoice: "Acompte à payer", credit_note: "Montant de l’avoir" }[d.kind]

  return (
    <article
      aria-label="Aperçu du document"
      className="flex flex-col gap-7 rounded-[2px] bg-white px-15 py-14 text-[#1b3a4b] tabular-nums shadow-[0_1px_3px_rgb(15_40_60/0.12),0_12px_32px_-12px_rgb(15_40_60/0.25)]"
      style={{ width: PAGE_W, minHeight: PAGE_H }}
    >
      <header className="flex justify-between gap-8">
        <div className="flex flex-col gap-0.5 text-xs leading-[18px]">
          <span className="text-[15px] leading-[22px] font-semibold">{issuerName}</span>
          {addressLines(d.issuer.address).map((l) => (
            <span key={l} className="text-[#5b6b78]">
              {l}
            </span>
          ))}
          {d.issuer.siret && <span className="text-[#5b6b78]">SIRET {d.issuer.siret}</span>}
          {!franchise && d.issuer.vat_number && <span className="text-[#5b6b78]">TVA {d.issuer.vat_number}</span>}
        </div>
        <div className="flex flex-col items-end gap-0.5 text-right text-xs leading-[18px]">
          <span className="text-[26px] leading-8 font-bold tracking-[0.08em] whitespace-nowrap">{docTitles[d.kind]}</span>
          <span className="text-[#5b6b78]">{d.number ? `N° ${d.number}` : "Numéro attribué à l’émission"}</span>
          {meta.map((m) => (
            <span key={m}>{m}</span>
          ))}
        </div>
      </header>
      <div className="h-[3px] w-14 bg-[#2f9e94]" />
      <div className="flex items-start justify-between gap-8">
        <div className="flex flex-col gap-1 text-xs">
          <span className="text-[10px] tracking-[0.08em] text-[#5b6b78] uppercase">Objet</span>
          <span className="text-sm font-semibold">{d.title || "—"}</span>
        </div>
        <div className="flex min-w-60 flex-col gap-0.5 rounded border border-[#d9e1e7] px-4 py-3 text-xs leading-[18px]">
          <span className="text-[10px] tracking-[0.08em] text-[#5b6b78] uppercase">{isQuote ? "Adressé à" : "Facturé à"}</span>
          <span className="font-semibold">{d.client?.name || "[Nom du client]"}</span>
          {addressLines(d.client?.address).map((l) => (
            <span key={l} className="text-[#5b6b78]">
              {l}
            </span>
          ))}
          {d.client?.siren && <span className="text-[#5b6b78]">SIREN {d.client.siren}</span>}
        </div>
      </div>
      <table className="w-full border-collapse text-xs leading-[18px]">
        <thead>
          <tr className="bg-[#eef3f5] text-[11px] tracking-[0.04em] text-[#5b6b78] uppercase">
            <th className="border-b border-[#d9e1e7] px-2.5 py-2 text-left font-semibold">Désignation</th>
            <th className="border-b border-[#d9e1e7] px-2.5 py-2 text-right font-semibold">Qté</th>
            <th className="border-b border-[#d9e1e7] px-2.5 py-2 text-right font-semibold">PU HT</th>
            {!franchise && <th className="border-b border-[#d9e1e7] px-2.5 py-2 text-right font-semibold">TVA</th>}
            <th className="border-b border-[#d9e1e7] px-2.5 py-2 text-right font-semibold">Total HT</th>
          </tr>
        </thead>
        <tbody>
          {d.lines.map((l, i) => (
            <tr key={i}>
              <td className="border-b border-[#e3e9ed] p-2.5 align-top">{l.description || "—"}</td>
              <td className="border-b border-[#e3e9ed] p-2.5 text-right align-top">{qty(l.quantity_milli)}</td>
              <td className="border-b border-[#e3e9ed] p-2.5 text-right align-top whitespace-nowrap">{formatEUR(l.unit_price_cents)}</td>
              {!franchise && (
                <td className="border-b border-[#e3e9ed] p-2.5 text-right align-top">{(l.tax_rate_bps / 100).toLocaleString("fr-FR")} %</td>
              )}
              <td className="border-b border-[#e3e9ed] p-2.5 text-right align-top whitespace-nowrap">{formatEUR(lineHT(l))}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="flex justify-end">
        <div className="flex w-70 flex-col gap-1.5 text-xs leading-[18px]">
          {d.deposit && (
            <>
              <Row label="Total du devis HT" value={formatEUR(d.deposit.quoteHT)} />
              <Row label={`Acompte (${d.deposit.percent} %)`} value={formatEUR(t?.subtotal_cents ?? 0)} />
            </>
          )}
          {!d.deposit && <Row label="Total HT" value={formatEUR(t?.subtotal_cents ?? 0)} />}
          {franchise ? (
            <Row label="TVA" value="—" />
          ) : (
            (t?.buckets ?? []).map((b) => (
              <Row key={b.tax_rate_bps} label={`TVA ${(b.tax_rate_bps / 100).toLocaleString("fr-FR")} %`} value={formatEUR(b.tax_cents)} />
            ))
          )}
          <div className="flex justify-between gap-4 border-t-[1.5px] border-[#1b3a4b] pt-2 text-sm font-bold">
            <span>{finalLabel}</span>
            <span>{formatEUR(t?.total_cents ?? 0)}</span>
          </div>
        </div>
      </div>
      {d.notes && <p className="text-xs leading-[18px] whitespace-pre-line">{d.notes}</p>}
      <div className="flex flex-col gap-1 text-[11px] leading-4">
        {mentions.map((m) => (
          <span key={m}>{m}</span>
        ))}
      </div>
      <div className="flex-1" />
      {isQuote && (
        <div className="flex justify-end">
          <div className="h-24 w-65 rounded border border-dashed border-[#c3d0db] px-3 py-2.5 text-[11px] leading-4 text-[#5b6b78]">
            Bon pour accord : date et signature
          </div>
        </div>
      )}
      <footer className="border-t border-[#e3e9ed] pt-3 text-center text-[10px] leading-[14px] text-[#5b6b78]">
        {issuerName}
        {d.issuer.siret ? ` · SIRET ${d.issuer.siret}` : ""}
      </footer>
    </article>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-4">
      <span className="text-[#5b6b78]">{label}</span>
      <span>{value}</span>
    </div>
  )
}

const ZOOMS = [0.5, 0.67, 0.75, 0.9, 1, 1.1, 1.25, 1.5]

export function DocumentPreview({
  data,
  formOpen,
  onToggleForm,
  className,
  compact,
  panelName = "le formulaire",
}: {
  data: PreviewData
  formOpen?: boolean
  onToggleForm?: () => void
  className?: string
  compact?: boolean
  /** Panneau masqué par le bouton de bascule. */
  panelName?: string
}) {
  const toggleLabel = `${formOpen ? "Masquer" : "Afficher"} ${panelName}`
  const rootRef = useRef<HTMLDivElement>(null)
  const scrollRef = useRef<HTMLDivElement>(null)
  const [zoom, setZoom] = useState(1)
  const [auto, setAuto] = useState(true)

  // Ajusté à la largeur tant que l'utilisateur n'a pas choisi de zoom.
  useLayoutEffect(() => {
    const el = scrollRef.current
    if (!el || !auto) return
    const fit = () => {
      const pad = compact ? 24 : 64
      setZoom(Math.min(1, Math.max(0.3, (el.clientWidth - pad) / PAGE_W)))
    }
    fit()
    const ro = new ResizeObserver(fit)
    ro.observe(el)
    return () => ro.disconnect()
  }, [auto, compact])

  const [fullscreen, setFullscreen] = useState(false)
  useEffect(() => {
    const on = () => setFullscreen(document.fullscreenElement === rootRef.current)
    document.addEventListener("fullscreenchange", on)
    return () => document.removeEventListener("fullscreenchange", on)
  }, [])

  const step = (dir: 1 | -1) => {
    setAuto(false)
    const next = dir > 0 ? ZOOMS.find((z) => z > zoom + 0.001) : [...ZOOMS].reverse().find((z) => z < zoom - 0.001)
    setZoom(next ?? zoom)
  }

  const fitPage = () => {
    const el = scrollRef.current
    if (!el) return
    setAuto(false)
    setZoom(Math.max(0.3, Math.min((el.clientHeight - 96) / PAGE_H, (el.clientWidth - 64) / PAGE_W)))
  }

  return (
    <div ref={rootRef} className={cn("relative h-full min-h-0 overflow-hidden bg-[color-mix(in_oklab,var(--muted)_90%,var(--foreground)_4%)]", className)}>
      <div ref={scrollRef} className={cn("absolute inset-0 flex justify-center overflow-auto", compact ? "px-3 pt-3 pb-20" : "px-8 pt-8 pb-24")}>
        <div className="shrink-0 self-start" style={{ zoom }}>
          <Sheet d={data} />
        </div>
      </div>
      <div className="absolute top-4 right-5 rounded-full max-lg:hidden border border-border bg-popover/90 px-2.5 py-1 text-xs text-muted-foreground">
        Aperçu · le PDF fait foi
      </div>
      <div
        role="toolbar"
        aria-label="Affichage de l’aperçu"
        className="absolute bottom-5 left-1/2 flex -translate-x-1/2 items-center gap-0.5 rounded-full border border-border bg-popover p-1 shadow-[0_10px_15px_-3px_rgb(15_40_60/0.12),0_4px_6px_-4px_rgb(15_40_60/0.12)]"
      >
        {onToggleForm && (
          <>
            <Button
              variant="ghost"
              size="icon"
              className="rounded-full"
              aria-label={toggleLabel}
              title={toggleLabel}
              onClick={onToggleForm}
            >
              {formOpen ? <PanelLeftClose /> : <PanelLeftOpen />}
            </Button>
            <Separator orientation="vertical" className="mx-1 h-5 self-center" />
          </>
        )}
        <Button variant="ghost" size="icon" className="rounded-full" aria-label="Zoom arrière" disabled={zoom <= ZOOMS[0]} onClick={() => step(-1)}>
          <Minus />
        </Button>
        <span className="min-w-12 text-center text-[13px] font-medium whitespace-nowrap tabular-nums" aria-live="polite">
          {Math.round(zoom * 100)} %
        </span>
        <Button variant="ghost" size="icon" className="rounded-full" aria-label="Zoom avant" disabled={zoom >= ZOOMS.at(-1)!} onClick={() => step(1)}>
          <Plus />
        </Button>
        <Separator orientation="vertical" className="mx-1 h-5 self-center" />
        {!compact && (
          <Button variant="ghost" size="icon" className="rounded-full" aria-label="Afficher la page entière" title="Afficher la page entière" onClick={fitPage}>
            <Expand />
          </Button>
        )}
        <Button
          variant="ghost"
          size="icon"
          className="rounded-full"
          aria-label={fullscreen ? "Quitter le plein écran" : "Plein écran"}
          title={fullscreen ? "Quitter le plein écran" : "Plein écran"}
          onClick={() => {
            if (document.fullscreenElement) void document.exitFullscreen()
            else void rootRef.current?.requestFullscreen()
          }}
        >
          <Maximize />
        </Button>
      </div>
    </div>
  )
}
