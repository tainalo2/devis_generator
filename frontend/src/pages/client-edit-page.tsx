import { Menu as MenuPrimitive } from "@base-ui/react/menu"
import { ChevronLeft, Ellipsis, Plus } from "lucide-react"
import { useEffect, useMemo, useState, type FormEvent, type ReactNode } from "react"
import { Link, Navigate, useNavigate, useParams } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { CompanyAutocomplete } from "@/components/company-autocomplete"
import { Avatar, Card, CardSection, CardTitle, Overline, Tabs } from "@/components/page"
import { AddressFields } from "@/components/party-fields"
import { applyCompanyHit } from "@/lib/party"
import { Seo } from "@/components/seo"
import { useToast } from "@/components/toast-provider"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Dialog } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select } from "@/components/ui/select"
import { Spinner } from "@/components/ui/spinner"
import { VATValidator } from "@/components/vat-validator"
import { deleteClient, getClient, listDocuments, saveClient } from "@/lib/api"
import { clientStats, documentStatus, formatDate, isIndividual, kindLabel } from "@/lib/documents"
import { useMediaQuery } from "@/lib/hooks"
import { menuItemClass, menuPopupClass, textLinkClass } from "@/lib/styles"
import { emptyAddress, formatEUR, type Document } from "@/lib/types"
import { cn } from "@/lib/utils"

const emptyForm = () => ({
  name: "",
  email: "",
  siren: "",
  siret: "",
  vat_number: "",
  iban: "",
  notes: "",
  address: emptyAddress(),
})

function Zone({ id, title, className, children }: { id: string; title: string; className?: string; children: ReactNode }) {
  return (
    <section aria-labelledby={id} className={cn("flex flex-col gap-3.5 rounded-xl border border-zone-parties-border bg-zone-parties p-4", className)}>
      <CardTitle id={id}>{title}</CardTitle>
      {children}
    </section>
  )
}

const DOCS_SHOWN = 5

export function ClientEditPage() {
  const { id } = useParams()
  const isNew = !id || id === "nouveau"
  const { user, loading } = useAuth()
  const navigate = useNavigate()
  const toast = useToast()
  const desktop = useMediaQuery("(min-width: 768px)")
  const [form, setForm] = useState(emptyForm)
  const [createdAt, setCreatedAt] = useState<string | null>(null)
  const [loaded, setLoaded] = useState(isNew)
  const [docs, setDocs] = useState<Document[]>([])
  const [docKind, setDocKind] = useState("")
  const [tab, setTab] = useState<"info" | "docs">("info")
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)

  useEffect(() => {
    if (isNew || !user || !id) return
    void getClient(id)
      .then((c) => {
        setForm({ ...emptyForm(), ...c, iban: c.iban ?? "", address: c.address ?? emptyAddress() })
        setCreatedAt(c.created_at ?? null)
        setLoaded(true)
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Client introuvable."))
    void listDocuments().then(setDocs).catch(() => undefined)
  }, [id, isNew, user])

  const stats = useMemo(() => clientStats(docs, id ?? ""), [docs, id])

  if (!loading && !user) return <Navigate to="/connexion" replace />

  const set = (p: Partial<typeof form>) => setForm((f) => ({ ...f, ...p }))
  const individual = isIndividual(form) && !!form.name

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setPending(true)
    setError(null)
    try {
      const saved = await saveClient(form, isNew ? undefined : id)
      toast.push(isNew ? "Client ajouté." : "Client enregistré.", "success")
      if (isNew) navigate(`/clients/${saved.id}`, { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur")
    } finally {
      setPending(false)
    }
  }

  async function remove() {
    if (!id) return
    try {
      await deleteClient(id)
      toast.push("Client supprimé.", "success")
      navigate("/clients", { replace: true })
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Suppression impossible.", "error")
    }
  }

  if (!loaded) {
    return error ? <p className="text-sm text-destructive">{error}</p> : <Spinner label="Chargement du client" />
  }

  const since = createdAt ? new Date(createdAt).toLocaleDateString("fr-FR", { month: "long", year: "numeric" }) : null
  const newDocHref = `/documents/nouveau?client=${id}`
  const clientDocs = stats.docs.filter((d) => !docKind || d.kind === docKind)

  const info = (
    <div className="flex min-w-0 flex-col gap-4">
      <Zone id="id-titre" title="Identité">
        {isNew && <CompanyAutocomplete onPick={(hit) => setForm((f) => applyCompanyHit(f, hit, "name"))} />}
        <Label>
          <span>{individual ? "Nom et prénom" : "Raison sociale"}</span>
          <Input required value={form.name} onChange={(e) => set({ name: e.target.value })} />
        </Label>
        <div className="grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(180px,1fr))]">
          <Label>
            <span>SIREN</span>
            <Input inputMode="numeric" placeholder="9 chiffres" value={form.siren} onChange={(e) => set({ siren: e.target.value })} />
          </Label>
          <Label>
            <span>SIRET du siège</span>
            <Input inputMode="numeric" placeholder="14 chiffres" value={form.siret} onChange={(e) => set({ siret: e.target.value })} />
          </Label>
        </div>
        <VATValidator value={form.vat_number} onChange={(vat_number) => set({ vat_number })} />
      </Zone>
      <Zone id="coord-titre" title="Coordonnées">
        <AddressFields value={form.address} onChange={(address) => set({ address })} />
        <div className="grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(180px,1fr))]">
          <Label>
            <span>E-mail de facturation</span>
            <Input type="email" placeholder="nom@exemple.fr" value={form.email} onChange={(e) => set({ email: e.target.value })} />
          </Label>
          <Label>
            <span>
              IBAN <span className="font-normal text-muted-foreground">(facultatif, chiffré)</span>
            </span>
            <Input value={form.iban} onChange={(e) => set({ iban: e.target.value })} />
          </Label>
        </div>
      </Zone>
      <section aria-labelledby="notes-titre" className="flex flex-col gap-2.5 rounded-xl border border-zone-lines-border bg-zone-lines p-4">
        <CardTitle id="notes-titre">Notes internes</CardTitle>
        <textarea
          aria-label="Notes internes"
          className="min-h-19 w-full resize-y rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:border-ring"
          placeholder="Visibles par vous seul, jamais sur les documents"
          value={form.notes}
          onChange={(e) => set({ notes: e.target.value })}
        />
      </section>
      {error && <p className="text-sm text-destructive">{error}</p>}
    </div>
  )

  const statCards = (
    <div className="grid grid-cols-2 gap-3">
      {[
        ["CA facturé", stats.billed],
        ["À encaisser", stats.due],
      ].map(([label, v]) => (
        <Card key={label} className="flex flex-col gap-1 p-3.5">
          <Overline>{label}</Overline>
          <span className="font-heading text-xl leading-[26px] font-semibold tabular-nums md:text-[22px] md:leading-7">{formatEUR(v as number)}</span>
        </Card>
      ))}
    </div>
  )

  const docsPanel = (
    <div className="flex min-w-0 flex-col gap-4">
      {desktop && statCards}
      <CardSection aria-labelledby="cdocs-titre" className="overflow-hidden">
        <div className="flex items-center justify-between gap-2 border-b border-border px-4 py-3.5">
          <CardTitle id="cdocs-titre">Documents</CardTitle>
          <Select aria-label="Filtrer par type" className="h-8 w-32.5" value={docKind} onChange={(e) => setDocKind(e.target.value)}>
            <option value="">Tous</option>
            <option value="quote">Devis</option>
            <option value="invoice">Factures</option>
            <option value="deposit_invoice">Acomptes</option>
            <option value="credit_note">Avoirs</option>
          </Select>
        </div>
        {clientDocs.length === 0 && <p className="px-4 py-3 text-sm text-muted-foreground">Aucun document pour ce client.</p>}
        {clientDocs.slice(0, DOCS_SHOWN).map((d) => {
          const st = documentStatus(d)
          return (
            <Link key={d.id} to={`/documents/${d.id}`} className="flex items-center justify-between gap-3 border-b border-border px-4 py-3 hover:bg-muted/60">
              <span className="flex flex-col gap-0.5">
                <span className="text-sm font-medium">
                  {kindLabel(d.kind)} {d.number ?? "(brouillon)"}
                </span>
                <span className="text-xs text-muted-foreground tabular-nums">
                  {d.status === "draft" ? `Modifié le ${formatDate(d.updated_at)}` : formatDate(d.issue_date)}
                </span>
              </span>
              <span className="flex flex-col items-end gap-1">
                <span className="text-sm font-semibold tabular-nums">{formatEUR(d.total_cents)}</span>
                <Badge variant={st.variant}>{st.label}</Badge>
              </span>
            </Link>
          )
        })}
        {clientDocs.length > DOCS_SHOWN && (
          <Link to="/documents" className={cn(textLinkClass, "block px-4 py-3 text-[13px]")}>
            Voir les {clientDocs.length} documents
          </Link>
        )}
      </CardSection>
    </div>
  )

  return (
    <form onSubmit={(e) => void onSubmit(e)} className="flex flex-col gap-5">
      <Seo title={isNew ? "Nouveau client" : form.name || "Client"} path={isNew ? "/clients/nouveau" : `/clients/${id}`} noIndex />
      <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-3">
        <div className="flex min-w-0 items-center gap-3">
          <Button variant="ghost" size="icon-sm" aria-label="Retour aux clients" render={<Link to="/clients" />}>
            <ChevronLeft />
          </Button>
          {!isNew && <Avatar name={form.name} />}
          <div className="flex min-w-0 flex-col gap-0.5">
            <h1 className="truncate font-heading text-[22px] leading-[1.2] font-semibold tracking-tight md:text-[28px]">
              {isNew ? "Nouveau client" : form.name}
            </h1>
            {!isNew && (
              <span className="flex items-center gap-2 text-[13px] text-muted-foreground">
                <Badge variant="outline">{individual ? "Particulier" : "Professionnel"}</Badge>
                {since && `Client depuis ${since}`}
              </span>
            )}
          </div>
        </div>
        {desktop && (
          <div className="flex items-center gap-2">
            {!isNew && (
              <Button variant="outline" render={<Link to={newDocHref} />}>
                <Plus />
                Nouveau document
              </Button>
            )}
            <Button type="submit" disabled={pending}>
              {isNew ? "Ajouter le client" : "Enregistrer"}
            </Button>
            {!isNew && (
              <MenuPrimitive.Root>
                <MenuPrimitive.Trigger render={<Button variant="ghost" size="icon" aria-label="Autres actions : supprimer le client" />}>
                  <Ellipsis />
                </MenuPrimitive.Trigger>
                <MenuPrimitive.Portal>
                  <MenuPrimitive.Positioner side="bottom" align="end" sideOffset={4} className="z-50">
                    <MenuPrimitive.Popup className={cn(menuPopupClass, "w-52")}>
                      <MenuPrimitive.Item className={cn(menuItemClass, "text-destructive")} onClick={() => setConfirmDelete(true)}>
                        Supprimer le client
                      </MenuPrimitive.Item>
                    </MenuPrimitive.Popup>
                  </MenuPrimitive.Positioner>
                </MenuPrimitive.Portal>
              </MenuPrimitive.Root>
            )}
          </div>
        )}
      </div>

      {!desktop && !isNew && (
        <>
          {statCards}
          <Tabs
            label="Fiche client"
            value={tab}
            onChange={setTab}
            items={[
              { value: "info", label: "Informations" },
              { value: "docs", label: `Documents (${stats.count})` },
            ]}
          />
        </>
      )}

      {isNew ? (
        <div className="max-w-2xl">{info}</div>
      ) : desktop ? (
        <div className="grid items-start gap-5 md:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
          {info}
          {docsPanel}
        </div>
      ) : tab === "info" ? (
        <>
          {info}
          <Button type="button" variant="ghost" className="self-start text-destructive" onClick={() => setConfirmDelete(true)}>
            Supprimer le client
          </Button>
        </>
      ) : (
        docsPanel
      )}

      {!desktop && (
        <div className="sticky bottom-0 z-[5] -mx-4 -mb-8 flex h-19 items-center gap-2 border-t border-border bg-card px-4 shadow-[0_-8px_16px_-12px_rgb(15_40_60/0.25)]">
          {!isNew && (
            <Button variant="outline" className="h-11 flex-1 text-sm" render={<Link to={newDocHref} />}>
              Nouveau document
            </Button>
          )}
          <Button type="submit" className="h-11 flex-1 text-sm" disabled={pending}>
            {isNew ? "Ajouter le client" : "Enregistrer"}
          </Button>
        </div>
      )}

      <Dialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title="Supprimer ce client ?"
        description="Les documents déjà émis restent intacts ; ils ne seront simplement plus rattachés à une fiche."
      >
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="lg" onClick={() => setConfirmDelete(false)}>
            Annuler
          </Button>
          <Button variant="destructive" size="lg" onClick={() => void remove()}>
            Supprimer le client
          </Button>
        </div>
      </Dialog>
    </form>
  )
}
