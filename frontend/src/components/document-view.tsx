import { Menu as MenuPrimitive } from "@base-ui/react/menu"
import { Check, ChevronLeft, Download, Ellipsis, Lock, Mail } from "lucide-react"
import { useEffect, useMemo, useState, type ReactNode } from "react"
import { Link, useNavigate } from "react-router-dom"

import { DocumentPreview, type PreviewData } from "@/components/editor/preview"
import { docHT, emptyIssuer, type EditorKind } from "@/components/editor/model"
import { CardSection, CardTitle } from "@/components/page"
import { useToast } from "@/components/toast-provider"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  convertQuoteToInvoice,
  createCreditNote,
  downloadBlob,
  downloadDocumentPDF,
  getClient,
  listDocuments,
  saveDocument,
  updateDocumentStatus,
  type TrackedStatus,
} from "@/lib/api"
import { documentName, documentStatus, formatDate, formatDateTime, isInvoiceKind, kindLabel } from "@/lib/documents"
import { useMediaQuery } from "@/lib/hooks"
import { menuItemClass, menuPopupClass, textLinkClass } from "@/lib/styles"
import { formatEUR, type Client, type Document, type TotalsPreview } from "@/lib/types"
import { cn } from "@/lib/utils"

const fem = (kind: string) => kind !== "quote" && kind !== "credit_note"

type Step = { label: string; at: string | null | undefined; pending: string }

function Steps({ steps }: { steps: Step[] }) {
  return (
    <ol className="flex flex-col">
      {steps.map((s, i) => {
        const done = !!s.at
        const nextDone = !!steps[i + 1]?.at
        return (
          <li key={s.label} className="flex gap-2.5">
            <span aria-hidden className="flex flex-col items-center">
              <span
                className={cn(
                  "flex size-[18px] items-center justify-center rounded-full border-[1.5px] text-brand-foreground",
                  done ? "border-brand bg-brand" : "border-border",
                )}
              >
                {done && <Check className="size-2.5" strokeWidth={3.5} />}
              </span>
              {i < steps.length - 1 && <span className={cn("min-h-3.5 w-[1.5px] flex-1", done && nextDone ? "bg-brand" : "bg-border")} />}
            </span>
            <span className="flex flex-col pb-3">
              <span className={cn("text-sm font-medium", !done && "text-muted-foreground")}>
                {s.label}
                <span className="sr-only">{done ? " : fait" : " : en attente"}</span>
              </span>
              <span className="text-xs text-muted-foreground tabular-nums">{done ? formatDateTime(s.at) : s.pending}</span>
            </span>
          </li>
        )
      })}
    </ol>
  )
}

function totalsOf(doc: Document): TotalsPreview {
  const buckets = new Map<number, { ht_cents: number; tax_cents: number }>()
  for (const l of doc.lines ?? []) {
    const b = buckets.get(l.tax_rate_bps) ?? { ht_cents: 0, tax_cents: 0 }
    b.ht_cents += l.total_cents ?? 0
    b.tax_cents += l.tax_cents ?? 0
    buckets.set(l.tax_rate_bps, b)
  }
  return {
    subtotal_cents: doc.subtotal_cents,
    tax_cents: doc.tax_cents,
    total_cents: doc.total_cents,
    buckets: [...buckets.entries()].filter(([r]) => r > 0).map(([tax_rate_bps, b]) => ({ tax_rate_bps, ...b })),
  }
}

/** Document émis : immuable, seul le suivi (envoi, acceptation, paiement) évolue. */
export function DocumentView({ doc, onChange }: { doc: Document; onChange: (d: Document) => void }) {
  const navigate = useNavigate()
  const toast = useToast()
  const desktop = useMediaQuery("(min-width: 1024px)")
  const [client, setClient] = useState<Client | null>(null)
  const [docs, setDocs] = useState<Document[]>([])
  const [sideOpen, setSideOpen] = useState(true)
  const [pending, setPending] = useState(false)

  useEffect(() => {
    if (doc.client_id) void getClient(doc.client_id).then(setClient).catch(() => setClient(null))
    void listDocuments().then(setDocs).catch(() => undefined)
  }, [doc.client_id])

  const origin = doc.related_document_id ? docs.find((d) => d.id === doc.related_document_id) : undefined
  const children = useMemo(() => docs.filter((d) => d.related_document_id === doc.id), [docs, doc.id])
  const st = documentStatus(doc)
  const isInvoice = isInvoiceKind(doc.kind)
  const isQuote = doc.kind === "quote"
  const name = documentName(doc)
  const e = fem(doc.kind) ? "e" : ""

  async function run<T>(fn: () => Promise<T>, done: (r: T) => void) {
    setPending(true)
    try {
      done(await fn())
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Action impossible.", "error")
    } finally {
      setPending(false)
    }
  }

  const setStatus = (s: TrackedStatus, msg: string) =>
    run(
      () => updateDocumentStatus(doc.id, s),
      (d) => {
        onChange({ ...d, lines: d.lines ?? doc.lines })
        toast.push(msg, "success")
      },
    )

  const openDraft = (d: Document, msg: string) => {
    toast.push(msg, "success")
    navigate(`/documents/${d.id}`)
  }

  const downloadPDF = () =>
    run(
      () => downloadDocumentPDF(doc.id),
      (b) => downloadBlob(b, `${doc.number ?? doc.id}.pdf`),
    )

  const mailto = (() => {
    const subject = `${name}${doc.title ? ` · ${doc.title}` : ""}`
    const body = `Bonjour,\n\nVeuillez trouver ci-joint ${isQuote ? "le devis" : doc.kind === "credit_note" ? "l’avoir" : "la facture"} ${doc.number ?? ""} d’un montant de ${formatEUR(doc.total_cents)}.\n\nCordialement,`
    return `mailto:${client?.email ?? ""}?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(body)}`
  })()

  const duplicate = () =>
    run(
      () =>
        saveDocument({
          kind: doc.kind === "credit_note" ? "invoice" : doc.kind,
          title: doc.title,
          vat_regime: doc.vat_regime,
          client_id: doc.client_id,
          issue_date: null,
          due_date: null,
          notes: doc.notes,
          payment_terms: doc.payment_terms,
          currency: doc.currency,
          lines: (doc.lines ?? []).map(({ id: _id, tax_cents: _t, total_cents: _tt, ...l }) => l),
        }),
      (d) => openDraft(d, "Copie créée en brouillon."),
    )

  // Action principale selon le type et le statut.
  let primary: ReactNode = null
  if (isInvoice && doc.status !== "paid") {
    primary = (
      <Button disabled={pending} onClick={() => void setStatus("paid", "Facture marquée comme payée.")}>
        <Check />
        Marquer comme payée
      </Button>
    )
  } else if (isQuote && (doc.status === "issued" || doc.status === "sent")) {
    primary = (
      <Button disabled={pending} onClick={() => void setStatus("accepted", "Devis marqué comme accepté.")}>
        <Check />
        Marquer comme accepté
      </Button>
    )
  } else if (isQuote && doc.status === "accepted") {
    primary = (
      <Button disabled={pending} onClick={() => void run(() => convertQuoteToInvoice(doc.id), (d) => openDraft(d, "Facture créée en brouillon depuis le devis."))}>
        Créer la facture
      </Button>
    )
  }

  const menuItems: { label: string; onClick: () => void }[] = [
    ...(doc.status === "issued" ? [{ label: `Marquer comme envoyé${e}`, onClick: () => void setStatus("sent", `Document marqué comme envoyé.`) }] : []),
    ...(isQuote && (doc.status === "issued" || doc.status === "sent")
      ? [{ label: "Marquer comme refusé", onClick: () => void setStatus("rejected", "Devis marqué comme refusé.") }]
      : []),
    ...(isQuote && doc.status !== "rejected"
      ? [{ label: "Créer une facture d’acompte", onClick: () => navigate(`/documents/nouveau?type=acompte&devis=${doc.id}`) }]
      : []),
    ...(isInvoice
      ? [{ label: "Créer un avoir", onClick: () => void run(() => createCreditNote(doc.id), (d) => openDraft(d, "Avoir créé en brouillon.")) }]
      : []),
    { label: "Dupliquer", onClick: () => void duplicate() },
  ]

  const actionsMenu = (
    <MenuPrimitive.Root>
      <MenuPrimitive.Trigger render={<Button variant="ghost" size="icon" className="max-lg:size-11" aria-label="Autres actions" />}>
        <Ellipsis />
      </MenuPrimitive.Trigger>
      <MenuPrimitive.Portal>
        <MenuPrimitive.Positioner side="bottom" align="end" sideOffset={4} className="z-50">
          <MenuPrimitive.Popup className={cn(menuPopupClass, "w-60")}>
            {!desktop && (
              <>
                <MenuPrimitive.Item className={menuItemClass} onClick={() => void downloadPDF()}>
                  <Download />
                  Télécharger le PDF
                </MenuPrimitive.Item>
                <MenuPrimitive.LinkItem className={menuItemClass} render={<a href={mailto} />}>
                  <Mail />
                  Envoyer par e-mail
                </MenuPrimitive.LinkItem>
                <div role="separator" className="my-1 h-px bg-border" />
              </>
            )}
            {menuItems.map((m) => (
              <MenuPrimitive.Item key={m.label} className={menuItemClass} onClick={m.onClick}>
                {m.label}
              </MenuPrimitive.Item>
            ))}
          </MenuPrimitive.Popup>
        </MenuPrimitive.Positioner>
      </MenuPrimitive.Portal>
    </MenuPrimitive.Root>
  )

  /* ---------- Panneau latéral ---------- */

  const amountLabel = isInvoice ? "Reste à payer" : isQuote ? "Total du devis" : "Montant de l’avoir"
  const amount = isInvoice && doc.status === "paid" ? 0 : doc.total_cents

  const trackSteps: Step[] = isInvoice
    ? [
        { label: "Émise", at: doc.issued_at, pending: "" },
        { label: "Envoyée au client", at: doc.sent_at, pending: "Non indiqué" },
        { label: "Payée", at: doc.paid_at, pending: "En attente" },
      ]
    : isQuote
      ? [
          { label: "Émis", at: doc.issued_at, pending: "" },
          { label: "Envoyé au client", at: doc.sent_at, pending: "Non indiqué" },
          { label: doc.status === "rejected" ? "Refusé" : "Accepté", at: doc.accepted_at, pending: "En attente de réponse" },
        ]
      : [
          { label: "Émis", at: doc.issued_at, pending: "" },
          { label: "Envoyé au client", at: doc.sent_at, pending: "Non indiqué" },
        ]

  const history = [
    { label: "Créé" + e, at: doc.created_at },
    origin ? { label: `Créé${e} depuis ${documentName(origin).toLowerCase()}`, at: doc.created_at } : null,
    { label: `Émis${e}, numéro ${doc.number} attribué`, at: doc.issued_at },
    { label: `Marqué${e} comme envoyé${e}`, at: doc.sent_at },
    { label: doc.status === "rejected" ? "Refusé par le client" : "Accepté par le client", at: doc.accepted_at },
    { label: "Paiement reçu", at: doc.paid_at },
  ]
    .filter((h): h is { label: string; at: string } => !!h?.at)
    .sort((a, b) => b.at.localeCompare(a.at))

  const linked = [...(origin ? [origin] : []), ...children]

  const side = (
    <aside
      aria-label="Informations du document"
      className={cn("flex flex-col gap-3", desktop ? "w-100 shrink-0 overflow-y-auto border-r border-border p-5" : "p-4")}
    >
      <CardSection aria-labelledby="suivi-titre" className="flex flex-col gap-3.5 p-4">
        <div className="flex items-baseline justify-between gap-2">
          <CardTitle id="suivi-titre">{isInvoice ? "Paiement" : "Suivi"}</CardTitle>
          {doc.due_date && doc.kind !== "credit_note" && (
            <span className="text-xs text-muted-foreground">
              {isQuote ? "Valable jusqu’au" : "Échéance le"} {formatDate(doc.due_date)}
            </span>
          )}
        </div>
        <div className="flex flex-col gap-0.5">
          <span className="text-xs text-muted-foreground">{amountLabel}</span>
          <span className="font-heading text-[28px] leading-[34px] font-semibold tracking-tight tabular-nums">{formatEUR(amount)}</span>
        </div>
        <Steps steps={trackSteps} />
      </CardSection>

      <CardSection aria-labelledby="client-titre" className="flex flex-col gap-2 p-4">
        <div className="flex items-baseline justify-between gap-2">
          <CardTitle id="client-titre">Client</CardTitle>
          {client && (
            <Link to={`/clients/${client.id}`} className={textLinkClass}>
              Voir la fiche
            </Link>
          )}
        </div>
        {client ? (
          <>
            <span className="text-sm font-semibold">{client.name}</span>
            <span className="text-[13px] text-muted-foreground">
              {[client.address?.line1, [client.address?.postal_code, client.address?.city].filter(Boolean).join(" "), client.siren && `SIREN ${client.siren}`]
                .filter(Boolean)
                .join(" · ")}
            </span>
            {client.email && <span className="text-[13px] text-muted-foreground">{client.email}</span>}
          </>
        ) : (
          <span className="text-sm text-muted-foreground">Aucun client rattaché.</span>
        )}
      </CardSection>

      <CardSection aria-labelledby="lies-titre" className="flex flex-col gap-2.5 p-4">
        <CardTitle id="lies-titre">Documents liés</CardTitle>
        {linked.length === 0 && <p className="text-sm text-muted-foreground">Aucun document lié.</p>}
        {linked.map((d) => {
          const s = documentStatus(d)
          return (
            <Link key={d.id} to={`/documents/${d.id}`} className="flex items-center justify-between gap-3 rounded-lg bg-muted/70 px-3 py-2.5">
              <span className="flex flex-col gap-0.5">
                <span className="text-sm font-medium">
                  {kindLabel(d.kind)} {d.number ?? "(brouillon)"}
                </span>
                <span className="text-xs text-muted-foreground tabular-nums">
                  {formatDate(d.issue_date)} · {formatEUR(d.total_cents)}
                </span>
              </span>
              <Badge variant={s.variant}>{s.label}</Badge>
            </Link>
          )
        })}
        {isInvoice && (
          <>
            <p className="text-xs text-muted-foreground">Une erreur sur cette facture ? Elle se corrige par un avoir.</p>
            <Button
              variant="outline"
              className="self-start"
              disabled={pending}
              onClick={() => void run(() => createCreditNote(doc.id), (d) => openDraft(d, "Avoir créé en brouillon."))}
            >
              Créer un avoir
            </Button>
          </>
        )}
      </CardSection>

      <CardSection aria-labelledby="histo-titre" className="flex flex-col gap-2.5 p-4">
        <CardTitle id="histo-titre">Historique</CardTitle>
        {history.map((h) => (
          <div key={h.label} className="flex justify-between gap-3 text-[13px] leading-[18px]">
            <span>{h.label}</span>
            <span className="whitespace-nowrap text-muted-foreground tabular-nums">
              {new Date(h.at).toLocaleDateString("fr-FR", { day: "2-digit", month: "2-digit" })} ·{" "}
              {new Date(h.at).toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" })}
            </span>
          </div>
        ))}
      </CardSection>
    </aside>
  )

  /* ---------- Aperçu ---------- */

  const preview: PreviewData = {
    kind: doc.kind as EditorKind,
    number: doc.number,
    title: doc.title,
    issue_date: doc.issue_date ?? "",
    due_date: doc.due_date ?? "",
    vat_regime: doc.vat_regime,
    payment_terms: doc.payment_terms,
    notes: doc.notes,
    lines: doc.lines ?? [],
    issuer: { ...emptyIssuer(), ...(doc.issuer_snapshot ?? {}) },
    client,
    totals: totalsOf(doc),
    deposit:
      doc.kind === "deposit_invoice" && origin
        ? {
            quoteNumber: origin.number,
            quoteDate: origin.issue_date,
            quoteHT: origin.subtotal_cents,
            percent: origin.subtotal_cents ? Math.round((docHT(doc.lines ?? []) / origin.subtotal_cents) * 100) : 0,
            remaining: origin.total_cents - doc.total_cents,
          }
        : undefined,
    relatedNumber: doc.kind === "credit_note" ? (origin?.number ?? null) : null,
  }

  const statusLine = (
    <>
      <Badge variant={st.variant}>{st.label}</Badge>
      <span className="flex items-center gap-1.5 text-xs whitespace-nowrap text-muted-foreground">
        <Lock className="size-3.5" aria-hidden />
        Émis{e} le {formatDate(doc.issue_date)} · non modifiable
      </span>
    </>
  )

  return (
    <div className={cn("flex flex-col", desktop && "h-[calc(100svh-4rem)] min-h-160 overflow-hidden")}>
      <header className="flex shrink-0 flex-col gap-1 border-b border-border bg-card py-2 pr-2 pl-1 lg:min-h-16 lg:justify-center lg:py-0 lg:pr-4 lg:pl-3">
        <div className="flex items-center gap-2.5">
          <Button variant="ghost" size="icon-sm" className="max-lg:size-11" aria-label="Retour aux documents" render={<Link to="/documents" />}>
            <ChevronLeft />
          </Button>
          <h1 className="min-w-0 truncate font-heading text-lg leading-7 font-semibold tracking-tight tabular-nums lg:flex-none lg:text-xl">{name}</h1>
          <div className="hidden items-center gap-2.5 lg:flex">{statusLine}</div>
          <div className="ml-auto flex shrink-0 items-center gap-2">
            {desktop && (
              <>
                <Button variant="outline" disabled={pending} onClick={() => void downloadPDF()}>
                  <Download />
                  Télécharger le PDF
                </Button>
                <Button variant="outline" render={<a href={mailto} />}>
                  <Mail />
                  Envoyer par e-mail
                </Button>
                {primary}
              </>
            )}
            {actionsMenu}
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2.5 pb-1 pl-3 lg:hidden">{statusLine}</div>
      </header>

      {desktop ? (
        <div className="flex min-h-0 flex-1">
          {sideOpen && side}
          <div className="min-h-0 min-w-0 flex-1">
            <DocumentPreview data={preview} formOpen={sideOpen} onToggleForm={() => setSideOpen((o) => !o)} panelName="les informations" />
          </div>
        </div>
      ) : (
        <>
          <div className="h-130">
            <DocumentPreview data={preview} compact />
          </div>
          {side}
          <div className="sticky bottom-0 z-[5] flex h-19 items-center justify-between gap-3 border-t border-border bg-card pr-3 pl-4 shadow-[0_-8px_16px_-12px_rgb(15_40_60/0.25)]">
            <span className="flex flex-col">
              <span className="text-xs text-muted-foreground">{amountLabel}</span>
              <span className="font-heading text-xl leading-[26px] font-semibold tabular-nums">{formatEUR(amount)}</span>
            </span>
            {primary && <div className="[&_button]:h-11 [&_button]:px-3.5 [&_button]:text-sm">{primary}</div>}
          </div>
        </>
      )}
    </div>
  )
}
