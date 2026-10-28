import { Menu as MenuPrimitive } from "@base-ui/react/menu"
import { Check, ChevronLeft, ChevronRight, Download, Ellipsis, Send } from "lucide-react"
import { useCallback, useEffect, useMemo, useRef, useState, type KeyboardEvent, type PointerEvent, type ReactNode } from "react"
import { Link, useNavigate } from "react-router-dom"

import { useToast } from "@/components/toast-provider"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Dialog } from "@/components/ui/dialog"
import { Segmented } from "@/components/ui/segmented"
import {
  deleteDocument,
  downloadBlob,
  downloadDocumentPDF,
  getDocument,
  getIssuer,
  guestPDF,
  issueDocument,
  listClients,
  listDocuments,
  previewTotals,
  saveClient,
  saveDocument,
  saveIssuer,
} from "@/lib/api"
import { localStore, useMediaQuery } from "@/lib/hooks"
import { menuItemClass, menuPopupClass } from "@/lib/styles"
import { formatEUR, type Client, type Document, type IssuerProfile, type TotalsPreview } from "@/lib/types"
import { cn } from "@/lib/utils"

import { ClientDialog, IssuerDialog } from "./dialogs"
import {
  depositLines,
  depositPercentOf,
  docHT,
  draftTitles,
  editorTitles,
  emptyClient,
  emptyIssuer,
  fromDocument,
  mainTotalLabel,
  newEditorValue,
  toPayload,
  type EditorKind,
  type EditorValue,
} from "./model"
import { DocumentPreview, type PreviewData } from "./preview"
import { InfosZone, LinesZone, PartiesZone, TermsZone } from "./zones"

const GUEST_ISSUER_KEY = "vibed.invite.emetteur"
const GUEST_CLIENT_KEY = "vibed.invite.client"
const FORM_W_KEY = "vibed.editeur.largeur"
const FORM_W_DEFAULT = 600
const FORM_W_MIN = 420

type SaveState = "new" | "dirty" | "saving" | "saved" | "error"

const kindOptions = (guest: boolean): { value: EditorKind; label: string }[] => [
  { value: "quote", label: "Devis" },
  { value: "invoice", label: "Facture" },
  ...(guest ? [] : [{ value: "deposit_invoice" as const, label: "Facture d’acompte" }]),
]

function clampW(w: number) {
  const max = Math.max(FORM_W_MIN, window.innerWidth - 480)
  return Math.min(max, Math.max(FORM_W_MIN, Math.round(w)))
}

type Props = {
  /** Brouillon existant ; absent pour un nouveau document. */
  draft?: Document
  guest: boolean
  /** Type demandé à la création (?type=facture). */
  initialKind?: EditorKind
  /** Devis d'origine demandé à la création d'un acompte (?devis=…). */
  initialQuoteId?: string
  /** Client présélectionné (?client=…). */
  initialClientId?: string
  onIssued?: (doc: Document) => void
}

export function DocumentEditor({ draft, guest, initialKind, initialQuoteId, initialClientId, onIssued }: Props) {
  const navigate = useNavigate()
  const toast = useToast()
  const desktop = useMediaQuery("(min-width: 1024px)")

  const [value, setValue] = useState<EditorValue>(() => (draft ? fromDocument(draft) : { ...newEditorValue(initialKind), client_id: initialClientId ?? "" }))
  const [docId, setDocId] = useState<string | null>(draft?.id ?? null)
  const [saveState, setSaveState] = useState<SaveState>(draft ? "saved" : "new")
  const [issuer, setIssuer] = useState<IssuerProfile>(() => (guest ? { ...emptyIssuer(), ...localStore.get<IssuerProfile>(GUEST_ISSUER_KEY) } : emptyIssuer()))
  const [guestClient, setGuestClient] = useState<Client>(() => ({ ...emptyClient(), ...localStore.get<Client>(GUEST_CLIENT_KEY) }))
  const [clients, setClients] = useState<Client[]>([])
  const [docs, setDocs] = useState<Document[]>([])
  const [totals, setTotals] = useState<TotalsPreview | null>(null)
  const [dialog, setDialog] = useState<"client" | "issuer" | "emit" | null>(null)
  const [pending, setPending] = useState(false)

  /* ---------- Chargement ---------- */

  useEffect(() => {
    if (guest) return
    void listClients().then(setClients).catch(() => undefined)
    void listDocuments().then(setDocs).catch(() => undefined)
    void getIssuer()
      .then((p) => {
        setIssuer({ ...emptyIssuer(), ...p })
        if (!draft && p.default_vat_regime) setValue((v) => ({ ...v, vat_regime: p.default_vat_regime }))
        if (!draft && p.payment_terms) setValue((v) => (v.kind === "quote" ? v : { ...v, payment_terms: p.payment_terms }))
      })
      .catch(() => undefined)
  }, [guest, draft])

  const byId = useMemo(() => new Map(docs.map((d) => [d.id, d])), [docs])
  const quotes = useMemo(() => docs.filter((d) => d.kind === "quote" && d.status !== "draft" && d.status !== "rejected"), [docs])
  const related = value.related_document_id ? byId.get(value.related_document_id) : undefined
  // La liste ne contient pas les lignes : on les charge pour le devis d'origine.
  const [quoteFull, setQuoteFull] = useState<Document | undefined>(undefined)
  const percentRestored = useRef(false)
  useEffect(() => {
    if (value.kind !== "deposit_invoice" || !value.related_document_id) return
    let cancelled = false
    void getDocument(value.related_document_id).then((q) => {
      if (cancelled) return
      setQuoteFull(q)
      // Brouillon d'acompte existant : retrouver le pourcentage à partir de ses lignes.
      if (draft && draft.related_document_id === q.id && !percentRestored.current) {
        percentRestored.current = true
        setValue((v) => ({ ...v, deposit_percent: depositPercentOf(draft, q) }))
      }
    })
    return () => {
      cancelled = true
    }
  }, [value.kind, value.related_document_id, draft])

  const quote = value.kind === "deposit_invoice" && quoteFull?.id === value.related_document_id ? quoteFull : undefined

  /** Lignes réellement émises : pour un acompte, calculées depuis le devis. */
  const lines = useMemo(
    () => (value.kind === "deposit_invoice" ? (quote ? depositLines(quote, value.deposit_percent) : []) : value.lines),
    [value.kind, value.lines, value.deposit_percent, quote],
  )
  const effective = useMemo(() => ({ ...value, lines }), [value, lines])

  const client = guest ? guestClient : (clients.find((c) => c.id === value.client_id) ?? null)

  /* ---------- Totaux ---------- */

  useEffect(() => {
    let cancelled = false
    const t = window.setTimeout(() => {
      void previewTotals(value.vat_regime, lines)
        .then((res) => !cancelled && setTotals(res))
        .catch(() => !cancelled && setTotals(null))
    }, 200)
    return () => {
      cancelled = true
      window.clearTimeout(t)
    }
  }, [value.vat_regime, lines])

  /* ---------- Modifications et enregistrement ---------- */

  const patch = useCallback((p: Partial<EditorValue>) => {
    setValue((v) => {
      const next = { ...v, ...p }
      // Un acompte reprend le client et le régime de TVA du devis d'origine.
      if (p.related_document_id !== undefined && next.kind === "deposit_invoice") {
        const q = docs.find((d) => d.id === p.related_document_id)
        if (q) Object.assign(next, { client_id: q.client_id ?? "", vat_regime: q.vat_regime, title: v.title || (q.title ? `Acompte : ${q.title}` : "") })
      }
      return next
    })
    setSaveState((s) => (s === "new" ? "new" : "dirty"))
  }, [docs])

  // Acompte demandé depuis un devis : rattacher le devis dès que la liste est chargée.
  const quoteApplied = useRef(false)
  useEffect(() => {
    if (draft || !initialQuoteId || quoteApplied.current || docs.length === 0) return
    quoteApplied.current = true
    patch({ related_document_id: initialQuoteId })
  }, [draft, initialQuoteId, docs, patch])

  const save = useCallback(async (): Promise<Document | null> => {
    if (guest) return null
    if (value.kind === "deposit_invoice" && !value.related_document_id) {
      toast.push("Choisissez le devis d’origine de l’acompte.", "error")
      return null
    }
    setSaveState("saving")
    try {
      const d = await saveDocument(toPayload(effective), docId ?? undefined)
      setSaveState("saved")
      if (!docId) {
        setDocId(d.id)
        navigate(`/documents/${d.id}`, { replace: true })
      }
      return d
    } catch (err) {
      setSaveState("error")
      toast.push(err instanceof Error ? err.message : "Enregistrement impossible.", "error")
      return null
    }
  }, [guest, value.kind, value.related_document_id, effective, docId, navigate, toast])

  // Enregistrement automatique d'un brouillon existant.
  const saveRef = useRef(save)
  useEffect(() => {
    saveRef.current = save
  }, [save])
  useEffect(() => {
    if (guest || !docId || saveState !== "dirty") return
    const t = window.setTimeout(() => void saveRef.current(), 1500)
    return () => window.clearTimeout(t)
  }, [guest, docId, saveState, value])

  async function downloadPDF() {
    setPending(true)
    try {
      if (guest) {
        const blob = await guestPDF({
          ...toPayload(effective),
          number: "",
          issuer: { ...issuer, legal_name: issuer.legal_name || "Émetteur", default_vat_regime: value.vat_regime },
          client: { ...guestClient, name: guestClient.name || "Client" },
          lines: lines.map(({ id: _id, ...rest }) => rest),
        })
        downloadBlob(blob, `${value.kind === "quote" ? "devis" : "facture"}.pdf`)
      } else {
        const d = saveState === "saved" && docId ? { id: docId } : await save()
        if (!d) return
        downloadBlob(await downloadDocumentPDF(d.id), "brouillon.pdf")
      }
      toast.push("PDF téléchargé.", "success")
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "PDF impossible.", "error")
    } finally {
      setPending(false)
    }
  }

  const missing = [
    !issuer.legal_name && "la raison sociale de l’émetteur",
    !issuer.siret && "le SIRET de l’émetteur",
    !client?.name && "le client",
    lines.length === 0 && "au moins une ligne",
    lines.some((l) => !l.description.trim()) && "la désignation de chaque ligne",
  ].filter(Boolean) as string[]

  async function emit() {
    setPending(true)
    try {
      const saved = await save()
      if (!saved) return
      const issued = await issueDocument(saved.id)
      setDialog(null)
      toast.push(`${issued.kind === "quote" ? "Devis" : "Document"} ${issued.number} émis.`, "success")
      onIssued?.(issued)
      navigate(`/documents/${issued.id}`, { replace: true })
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Émission impossible.", "error")
    } finally {
      setPending(false)
    }
  }

  async function removeDraft() {
    if (!docId) return navigate("/documents")
    try {
      await deleteDocument(docId)
      toast.push("Brouillon supprimé.", "success")
      navigate("/documents", { replace: true })
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Suppression impossible.", "error")
    }
  }

  /* ---------- Parties ---------- */

  async function submitIssuer(next: IssuerProfile) {
    if (guest) {
      const { iban: _iban, ...withoutIban } = next
      localStore.set(GUEST_ISSUER_KEY, withoutIban)
      setIssuer(next)
      return
    }
    setIssuer({ ...emptyIssuer(), ...(await saveIssuer(next)) })
    toast.push("Fiche Entreprise mise à jour.", "success")
  }

  async function submitClient(next: Client) {
    if (guest) {
      localStore.set(GUEST_CLIENT_KEY, next)
      setGuestClient(next)
      return
    }
    const { id: _id, ...input } = next
    const saved = await saveClient(input)
    setClients((list) => [...list, saved].sort((a, b) => a.name.localeCompare(b.name, "fr")))
    patch({ client_id: saved.id })
    toast.push("Client ajouté.", "success")
  }

  /* ---------- Aperçu ---------- */

  const preview: PreviewData = {
    ...effective,
    number: null,
    issuer,
    client,
    totals,
    deposit:
      value.kind === "deposit_invoice" && quote
        ? {
            quoteNumber: quote.number,
            quoteDate: quote.issue_date,
            quoteHT: docHT(quote.lines ?? []),
            percent: value.deposit_percent,
            remaining: quote.total_cents - (totals?.total_cents ?? 0),
          }
        : undefined,
    relatedNumber: value.kind === "credit_note" ? (related?.number ?? null) : null,
  }

  /* ---------- Rendu ---------- */

  const title = draft ? draftTitles[value.kind] : editorTitles[value.kind]
  const emitLabel = value.kind === "quote" ? "Émettre le devis" : value.kind === "credit_note" ? "Émettre l’avoir" : "Émettre la facture"
  const statusText = guest
    ? "Mode invité : rien n’est conservé"
    : { new: "Non enregistré", dirty: "Modifications en attente", saving: "Enregistrement…", saved: "Enregistré automatiquement", error: "Échec de l’enregistrement" }[saveState]
  const canPickKind = value.kind !== "credit_note"
  const backTo = guest ? "/" : "/documents"

  const zones = {
    infos: <InfosZone value={value} patch={patch} quotes={quotes} depositAmount={docHT(lines)} />,
    parties: (
      <PartiesZone
        issuer={issuer}
        client={client}
        clients={clients}
        value={value}
        patch={patch}
        guest={guest}
        onEditIssuer={() => setDialog("issuer")}
        onEditClient={() => setDialog("client")}
      />
    ),
    lines: <LinesZone value={value} patch={patch} quote={quote} depositLines={lines} />,
    terms: <TermsZone value={value} patch={patch} lockRegime={value.kind === "deposit_invoice" && !!quote} />,
  }

  const totalsItems =
    value.kind === "deposit_invoice" && quote
      ? [
          { label: "Total du devis", value: formatEUR(quote.total_cents) },
          { label: "Reste à facturer", value: formatEUR(quote.total_cents - (totals?.total_cents ?? 0)) },
        ]
      : [
          { label: "Total HT", value: formatEUR(totals?.subtotal_cents ?? 0) },
          { label: "TVA", value: value.vat_regime === "franchise_293b" ? "Non applicable" : formatEUR(totals?.tax_cents ?? 0) },
        ]

  const dialogs = (
    <>
      <ClientDialog open={dialog === "client"} onOpenChange={(o) => setDialog(o ? "client" : null)} initial={guest ? guestClient : emptyClient()} onSubmit={submitClient} guest={guest} />
      <IssuerDialog open={dialog === "issuer"} onOpenChange={(o) => setDialog(o ? "issuer" : null)} initial={issuer} onSubmit={submitIssuer} guest={guest} />
      <Dialog
        open={dialog === "emit"}
        onOpenChange={(o) => setDialog(o ? "emit" : null)}
        icon={<Send className="size-5 text-brand" aria-hidden />}
        title={emitLabel}
        description="Un numéro définitif est attribué et le document ne pourra plus être modifié. Une erreur se corrige ensuite par un avoir."
      >
        {missing.length > 0 ? (
          <div className="rounded-lg bg-destructive/10 px-3 py-2.5 text-sm text-destructive">
            Avant d’émettre, renseignez : {missing.join(", ")}.
          </div>
        ) : (
          <p className="text-sm">
            {title.replace(" (brouillon)", "")} · {client?.name} · <strong className="tabular-nums">{formatEUR(totals?.total_cents ?? 0)}</strong>
          </p>
        )}
        <div className="sticky -bottom-5 -mx-5 -mb-5 mt-1 flex justify-end gap-2 border-t border-border bg-card px-5 py-3">
          <Button variant="ghost" size="lg" onClick={() => setDialog(null)}>
            Annuler
          </Button>
          <Button size="lg" disabled={pending || missing.length > 0} onClick={() => void emit()}>
            <Send />
            {emitLabel}
          </Button>
        </div>
      </Dialog>
    </>
  )

  if (desktop) {
    return (
      <DesktopLayout
        title={title}
        statusText={statusText}
        saved={saveState === "saved"}
        kind={value.kind}
        kindOptions={canPickKind ? kindOptions(guest) : null}
        onKind={(k) => patch({ kind: k, related_document_id: k === "deposit_invoice" ? value.related_document_id : "" })}
        backTo={backTo}
        guest={guest}
        pending={pending}
        onSave={() => void save()}
        onPDF={() => void downloadPDF()}
        onEmit={() => setDialog("emit")}
        emitLabel={emitLabel}
        zones={zones}
        totalsItems={totalsItems}
        mainLabel={mainTotalLabel[value.kind]}
        mainValue={formatEUR(totals?.total_cents ?? 0)}
        preview={preview}
        dialogs={dialogs}
      />
    )
  }

  return (
    <MobileLayout
      title={title}
      statusText={statusText}
      kind={value.kind}
      kindOptions={canPickKind ? kindOptions(guest) : null}
      onKind={(k) => patch({ kind: k })}
      backTo={backTo}
      guest={guest}
      pending={pending}
      onSave={() => void save()}
      onPDF={() => void downloadPDF()}
      onEmit={() => setDialog("emit")}
      onDelete={docId ? () => void removeDraft() : undefined}
      emitLabel={emitLabel}
      zones={zones}
      mainLabel={mainTotalLabel[value.kind]}
      mainValue={formatEUR(totals?.total_cents ?? 0)}
      preview={preview}
      dialogs={dialogs}
    />
  )
}

/* ====================== Bureau ====================== */

type LayoutProps = {
  title: string
  statusText: string
  kind: EditorKind
  kindOptions: { value: EditorKind; label: string }[] | null
  onKind: (k: EditorKind) => void
  backTo: string
  guest: boolean
  pending: boolean
  onSave: () => void
  onPDF: () => void
  onEmit: () => void
  emitLabel: string
  zones: Record<"infos" | "parties" | "lines" | "terms", ReactNode>
  mainLabel: string
  mainValue: string
  preview: PreviewData
  dialogs: ReactNode
}

function DesktopLayout(p: LayoutProps & { saved: boolean; totalsItems: { label: string; value: string }[] }) {
  const [formOpen, setFormOpen] = useState(true)
  const [formW, setFormW] = useState(() => localStore.get<number>(FORM_W_KEY) ?? FORM_W_DEFAULT)
  const [dragging, setDragging] = useState(false)

  const setW = (w: number) => {
    const c = clampW(w)
    setFormW(c)
    localStore.set(FORM_W_KEY, c)
  }

  const startDrag = (e: PointerEvent<HTMLButtonElement>) => {
    if (e.button !== 0) return
    e.preventDefault()
    const startX = e.clientX
    const startW = formW
    const move = (ev: globalThis.PointerEvent) => setW(startW + ev.clientX - startX)
    const up = () => {
      window.removeEventListener("pointermove", move)
      window.removeEventListener("pointerup", up)
      window.removeEventListener("pointercancel", up)
      setDragging(false)
    }
    window.addEventListener("pointermove", move)
    window.addEventListener("pointerup", up)
    window.addEventListener("pointercancel", up)
    setDragging(true)
  }

  const onResizeKey = (e: KeyboardEvent<HTMLButtonElement>) => {
    const step = e.shiftKey ? 64 : 16
    let w = formW
    if (e.key === "ArrowLeft") w -= step
    else if (e.key === "ArrowRight") w += step
    else if (e.key === "Home") w = FORM_W_MIN
    else if (e.key === "End") w = 99999
    else return
    e.preventDefault()
    setW(w)
  }

  return (
    <div className={cn("flex h-[calc(100svh-4rem)] min-h-160 flex-col overflow-hidden", dragging && "cursor-col-resize select-none")}>
      <div className="flex h-14 shrink-0 items-center gap-4 border-b border-border bg-card pr-4 pl-3">
        <div className="flex min-w-0 flex-1 items-center gap-2.5">
          <Button variant="ghost" size="icon-sm" aria-label="Retour aux documents" render={<Link to={p.backTo} />}>
            <ChevronLeft />
          </Button>
          <h1 className="font-heading text-lg font-semibold tracking-tight whitespace-nowrap">{p.title}</h1>
          <Badge variant="muted">Brouillon</Badge>
          <span className="hidden items-center gap-1.5 truncate text-xs whitespace-nowrap text-muted-foreground xl:flex" role="status">
            {p.saved && <Check className="size-3.5" aria-hidden />}
            {p.statusText}
          </span>
        </div>
        {p.kindOptions && <Segmented label="Type de document" value={p.kind} options={p.kindOptions} onChange={p.onKind} />}
        <div className="flex flex-1 items-center justify-end gap-2">
          {!p.guest && (
            <Button variant="ghost" disabled={p.pending} onClick={p.onSave}>
              Enregistrer
            </Button>
          )}
          <Button variant={p.guest ? "default" : "outline"} disabled={p.pending} onClick={p.onPDF}>
            <Download />
            Télécharger le PDF
          </Button>
          {p.guest ? (
            <Button variant="outline" render={<Link to="/inscription" />}>
              Créer un compte
            </Button>
          ) : (
            <Button disabled={p.pending} onClick={p.onEmit}>
              <Send />
              {p.emitLabel}
            </Button>
          )}
        </div>
      </div>

      <div className="flex min-h-0 flex-1">
        {formOpen && (
          <>
            <div className="flex min-w-0 shrink-0 flex-col bg-background" style={{ width: formW }}>
              <div className="@container flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-5 *:shrink-0">
                {p.zones.infos}
                {p.zones.parties}
                {p.zones.lines}
                {p.zones.terms}
              </div>
              <div
                role="status"
                aria-label="Totaux du document"
                className="flex h-18 shrink-0 items-center justify-between gap-4 border-t border-border bg-card px-5 shadow-[0_-8px_16px_-12px_rgb(15_40_60/0.25)]"
              >
                <div className="flex gap-6 tabular-nums">
                  {p.totalsItems.map((it) => (
                    <div key={it.label} className="flex flex-col gap-0.5">
                      <span className="text-xs whitespace-nowrap text-muted-foreground">{it.label}</span>
                      <span className="text-sm font-medium whitespace-nowrap">{it.value}</span>
                    </div>
                  ))}
                </div>
                <div className="flex flex-col items-end">
                  <span className="text-xs whitespace-nowrap text-muted-foreground">{p.mainLabel}</span>
                  <span className="font-heading text-2xl leading-[30px] font-semibold tracking-tight whitespace-nowrap tabular-nums">{p.mainValue}</span>
                </div>
              </div>
            </div>
            <button
              type="button"
              aria-label="Redimensionner le formulaire (flèches gauche et droite, double-clic pour réinitialiser)"
              title="Glisser pour redimensionner, double-cliquer pour réinitialiser"
              data-dragging={dragging}
              onPointerDown={startDrag}
              onKeyDown={onResizeKey}
              onDoubleClick={() => setW(FORM_W_DEFAULT)}
              className="group relative z-[2] -mx-1 w-[9px] shrink-0 cursor-col-resize touch-none focus-visible:ring-0 focus-visible:ring-offset-0"
            >
              <span className="absolute inset-y-0 left-1 w-px bg-border transition-all group-hover:left-[3px] group-hover:w-[3px] group-hover:bg-ring group-focus-visible:left-[3px] group-focus-visible:w-[3px] group-focus-visible:bg-ring group-data-[dragging=true]:left-[3px] group-data-[dragging=true]:w-[3px] group-data-[dragging=true]:bg-ring" />
              <span className="absolute top-1/2 left-px -mt-4 h-8 w-[7px] rounded border border-border bg-card transition-colors group-hover:border-ring group-focus-visible:border-ring group-data-[dragging=true]:border-ring" />
            </button>
          </>
        )}
        <div className="min-h-0 min-w-0 flex-1">
          <DocumentPreview data={p.preview} formOpen={formOpen} onToggleForm={() => setFormOpen((o) => !o)} />
        </div>
      </div>
      {p.dialogs}
    </div>
  )
}

/* ====================== Mobile ====================== */

const steps = [
  { id: 1, label: "Parties" },
  { id: 2, label: "Lignes" },
  { id: 3, label: "Aperçu" },
] as const

function MobileLayout(p: LayoutProps & { onDelete?: () => void }) {
  const [step, setStep] = useState(1)
  const scrollRef = useRef<HTMLDivElement>(null)
  const go = (n: number) => {
    setStep(n)
    scrollRef.current?.scrollTo({ top: 0 })
  }

  return (
    <div className="flex h-[calc(100svh-3.75rem)] flex-col overflow-hidden">
      <div className="flex h-13 shrink-0 items-center gap-1 border-b border-border bg-card px-1">
        <Button variant="ghost" size="icon" className="size-11" aria-label="Retour aux documents" render={<Link to={p.backTo} />}>
          <ChevronLeft />
        </Button>
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <h1 className="truncate font-heading text-base font-semibold tracking-tight">{p.title}</h1>
          <Badge variant="muted">Brouillon</Badge>
        </div>
        <MenuPrimitive.Root>
          <MenuPrimitive.Trigger render={<Button variant="ghost" size="icon" className="size-11" aria-label="Actions du document : type, enregistrer, émettre" />}>
            <Ellipsis />
          </MenuPrimitive.Trigger>
          <MenuPrimitive.Portal>
            <MenuPrimitive.Positioner side="bottom" align="end" sideOffset={4} className="z-50">
              <MenuPrimitive.Popup className={cn(menuPopupClass, "w-64")}>
                {p.kindOptions && (
                  <MenuPrimitive.RadioGroup value={p.kind} onValueChange={(v) => p.onKind(v as EditorKind)}>
                    <MenuPrimitive.GroupLabel className="px-2.5 py-1 text-[11px] font-semibold tracking-[0.06em] text-muted-foreground uppercase">
                      Type de document
                    </MenuPrimitive.GroupLabel>
                    {p.kindOptions.map((o) => (
                      <MenuPrimitive.RadioItem key={o.value} value={o.value} className={menuItemClass} closeOnClick>
                        <span className="flex size-4 items-center justify-center">
                          <MenuPrimitive.RadioItemIndicator>
                            <Check />
                          </MenuPrimitive.RadioItemIndicator>
                        </span>
                        {o.label}
                      </MenuPrimitive.RadioItem>
                    ))}
                  </MenuPrimitive.RadioGroup>
                )}
                <div role="separator" className="my-1 h-px bg-border" />
                {!p.guest && (
                  <MenuPrimitive.Item className={menuItemClass} onClick={p.onSave}>
                    Enregistrer
                  </MenuPrimitive.Item>
                )}
                <MenuPrimitive.Item className={menuItemClass} onClick={p.onPDF}>
                  Télécharger le PDF
                </MenuPrimitive.Item>
                {!p.guest && (
                  <MenuPrimitive.Item className={menuItemClass} onClick={p.onEmit}>
                    {p.emitLabel}
                  </MenuPrimitive.Item>
                )}
                {p.onDelete && (
                  <MenuPrimitive.Item className={cn(menuItemClass, "text-destructive")} onClick={p.onDelete}>
                    Supprimer le brouillon
                  </MenuPrimitive.Item>
                )}
              </MenuPrimitive.Popup>
            </MenuPrimitive.Positioner>
          </MenuPrimitive.Portal>
        </MenuPrimitive.Root>
      </div>

      <nav aria-label="Étapes" className="shrink-0 border-b border-border bg-background px-3 py-1.5">
        <div className="flex gap-0.5 rounded-lg bg-muted p-0.5">
          {steps.map((s) => (
            <button
              key={s.id}
              type="button"
              aria-current={step === s.id ? "step" : undefined}
              onClick={() => go(s.id)}
              className={cn(
                "flex h-9 flex-1 items-center justify-center gap-1.5 rounded-md px-2 text-[13px] font-medium whitespace-nowrap focus-visible:ring-offset-0",
                step === s.id ? "bg-card text-foreground shadow-[0_1px_2px_rgb(15_40_60/0.1)]" : "text-muted-foreground",
              )}
            >
              <span
                className={cn(
                  "inline-flex size-[18px] items-center justify-center rounded-full text-[11px]",
                  step === s.id ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground",
                )}
              >
                {s.id}
              </span>
              {s.label}
            </button>
          ))}
        </div>
      </nav>

      <div className="flex min-h-0 flex-1 flex-col">
        {step < 3 ? (
          <div ref={scrollRef} className="@container flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-3 *:shrink-0">
            {/* Étape 1 : informations et parties ; étape 2 : lignes et conditions. */}
            {step === 1 ? (
              <>
                {p.zones.infos}
                {p.zones.parties}
              </>
            ) : (
              <>
                {p.zones.lines}
                {p.zones.terms}
              </>
            )}
            {p.guest && <p className="px-1 text-xs text-muted-foreground">{p.statusText}.</p>}
          </div>
        ) : (
          <DocumentPreview data={p.preview} compact className="flex-1" />
        )}
      </div>

      <div className="flex h-19 shrink-0 items-center justify-between gap-3 border-t border-border bg-card pr-3 pl-4 shadow-[0_-8px_16px_-12px_rgb(15_40_60/0.25)]">
        <div role="status" className="flex min-w-0 flex-col">
          <span className="text-xs whitespace-nowrap text-muted-foreground">{p.mainLabel}</span>
          <span className="font-heading text-xl leading-[26px] font-semibold tracking-tight whitespace-nowrap tabular-nums">{p.mainValue}</span>
        </div>
        <div className="flex items-center gap-2">
          {step > 1 && (
            <Button variant="outline" size="icon" className="size-11" aria-label="Étape précédente" onClick={() => go(step - 1)}>
              <ChevronLeft />
            </Button>
          )}
          {step < 3 ? (
            <Button className="h-11 px-3.5 text-sm" onClick={() => go(step + 1)}>
              {step === 1 ? "Lignes" : "Aperçu"}
              <ChevronRight />
            </Button>
          ) : p.guest ? (
            <Button className="h-11 px-3.5 text-sm" disabled={p.pending} onClick={p.onPDF}>
              <Download />
              Télécharger le PDF
            </Button>
          ) : (
            <Button className="h-11 px-3.5 text-sm" disabled={p.pending} onClick={p.onEmit}>
              <Send />
              Émettre
            </Button>
          )}
        </div>
      </div>
      {p.dialogs}
    </div>
  )
}
