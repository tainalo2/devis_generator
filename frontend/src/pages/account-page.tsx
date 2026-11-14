import { Copy, Download, ExternalLink, KeyRound, Plus } from "lucide-react"
import { useTheme } from "next-themes"
import { useEffect, useState, type ReactNode } from "react"
import { Link, Navigate, useNavigate } from "react-router-dom"

import { PasswordInput } from "@/components/auth-ui"
import { PageHeader } from "@/components/page"
import { Seo } from "@/components/seo"
import { Button } from "@/components/ui/button"
import { Dialog } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Segmented } from "@/components/ui/segmented"
import { useAuth } from "@/components/use-auth"
import { useToast } from "@/components/use-toast"
import { createAPIKey, listAPIKeys, revokeAPIKey, type APIKey } from "@/lib/api"
import { formatDate } from "@/lib/documents"
import { textLinkClass } from "@/lib/styles"
import { cn } from "@/lib/utils"

function Row({ id, title, description, danger, children, boxClassName }: { id: string; title: string; description: string; danger?: boolean; children: ReactNode; boxClassName?: string }) {
  return (
    <section aria-labelledby={id} className="grid gap-x-8 gap-y-3 border-t border-border py-6 md:grid-cols-[240px_minmax(0,1fr)]">
      <div className="flex flex-col gap-1">
        <h2 id={id} className={cn("font-heading text-lg leading-[26px] font-semibold tracking-tight", danger && "text-destructive")}>
          {title}
        </h2>
        <p className="text-[13px] leading-[18px] text-muted-foreground">{description}</p>
      </div>
      <div className={cn("flex flex-col gap-3.5 rounded-xl border border-border bg-card p-4", boxClassName)}>{children}</div>
    </section>
  )
}

const CONFIRM_WORD = "SUPPRIMER"
const MCP_DOCS_URL = "https://github.com/tainalo2/devis_generator/blob/main/docs/mcp.md"
const apiKeyDate = new Intl.DateTimeFormat("fr-FR", { dateStyle: "long" })

function formatAPIKeyDate(value: string) {
  return apiKeyDate.format(new Date(value))
}

export function AccountPage() {
  const { user, loading, logout, exportData, removeAccount } = useAuth()
  const { theme, setTheme } = useTheme()
  const navigate = useNavigate()
  const toast = useToast()
  const [pending, setPending] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [confirmText, setConfirmText] = useState("")
  const [password, setPassword] = useState("")
  const [apiKeys, setAPIKeys] = useState<APIKey[]>([])
  const [apiKeysLoading, setAPIKeysLoading] = useState(true)
  const [keyDialogOpen, setKeyDialogOpen] = useState(false)
  const [keyName, setKeyName] = useState("")
  const [createdKey, setCreatedKey] = useState("")
  const [keyPending, setKeyPending] = useState(false)

  useEffect(() => {
    if (!user) return
    let active = true
    void listAPIKeys()
      .then((keys) => {
        if (active) setAPIKeys(keys)
      })
      .catch((err: unknown) => {
        if (active) toast.push(err instanceof Error ? err.message : "Chargement des clés API impossible.", "error")
      })
      .finally(() => {
        if (active) setAPIKeysLoading(false)
      })
    return () => {
      active = false
    }
  }, [toast, user])

  if (!loading && !user) return <Navigate to="/connexion" replace />

  async function onExport() {
    setPending(true)
    try {
      await exportData()
      toast.push("Export téléchargé.", "success")
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Export impossible.", "error")
    } finally {
      setPending(false)
    }
  }

  async function onDelete() {
    setPending(true)
    try {
      await removeAccount(password)
      toast.push("Compte supprimé.", "success")
      navigate("/", { replace: true })
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Suppression impossible.", "error")
      setPending(false)
    }
  }

  async function onCreateAPIKey() {
    const name = keyName.trim()
    if (!name) return
    setKeyPending(true)
    try {
      const created = await createAPIKey(name)
      setAPIKeys((keys) => [created.api_key, ...keys])
      setCreatedKey(created.key)
      setKeyName("")
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Création de la clé API impossible.", "error")
    } finally {
      setKeyPending(false)
    }
  }

  async function onRevokeAPIKey(id: string) {
    setKeyPending(true)
    try {
      await revokeAPIKey(id)
      setAPIKeys((keys) => keys.map((key) => (key.id === id ? { ...key, revoked_at: new Date().toISOString() } : key)))
      toast.push("Clé API révoquée.", "success")
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Révocation de la clé API impossible.", "error")
    } finally {
      setKeyPending(false)
    }
  }

  async function copyCreatedKey() {
    try {
      await navigator.clipboard.writeText(createdKey)
      toast.push("Clé copiée.", "success")
    } catch {
      toast.push("Copie impossible. Sélectionnez la clé manuellement.", "error")
    }
  }

  function closeKeyDialog() {
    setKeyDialogOpen(false)
    setKeyName("")
    setCreatedKey("")
  }

  return (
    <div className="mx-auto flex w-full max-w-208 flex-col gap-2">
      <Seo title="Mon compte" path="/compte" noIndex />
      <PageHeader
        className="pb-4"
        title="Mon compte"
        description={user ? `Connecté avec ${user.email} · compte créé le ${formatDate(user.created_at)}` : undefined}
      />

      <Row id="s-profil" title="Profil" description="Votre nom n’apparaît pas sur les documents : c’est l’émetteur qui compte.">
        <div className="grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))]">
          <Label>
            <span>Nom</span>
            <Input readOnly value={user?.display_name || "—"} />
          </Label>
          <Label>
            <span>E-mail</span>
            <Input readOnly type="email" value={user?.email ?? ""} />
          </Label>
        </div>
        <p className="text-xs text-muted-foreground">
          Les informations des documents se modifient dans la{" "}
          <Link to="/entreprise" className={textLinkClass}>
            fiche Entreprise
          </Link>
          .
        </p>
      </Row>

      <Row id="s-secu" title="Session" description="Fermez votre session sur cet appareil.">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <span className="text-sm">Vous êtes connecté sur cet appareil.</span>
          <Button
            variant="outline"
            disabled={pending}
            onClick={() => {
              void logout().then(() => navigate("/", { replace: true }))
            }}
          >
            Se déconnecter
          </Button>
        </div>
      </Row>

      <Row id="s-pref" title="Préférences" description="L’interface reste en français." boxClassName="flex-row flex-wrap items-center justify-between">
        <span className="text-sm font-medium">Thème</span>
        <Segmented
          label="Thème"
          value={(theme as "system" | "light" | "dark") ?? "system"}
          onChange={setTheme}
          options={[
            { value: "system", label: "Système" },
            { value: "light", label: "Clair" },
            { value: "dark", label: "Sombre" },
          ]}
        />
      </Row>

      <Row id="s-api-keys" title="Clés API" description="Connectez l’API publique et le serveur MCP à votre compte.">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex max-w-xl flex-col gap-1 text-sm">
            <p>Les clés sont masquées après leur création. Révoquez immédiatement toute clé qui ne doit plus être utilisée.</p>
            <a className={cn(textLinkClass, "inline-flex w-fit items-center gap-1 text-[13px]")} href={MCP_DOCS_URL} target="_blank" rel="noreferrer">
              Installer le serveur MCP avec VIBED_API_URL et VIBED_API_KEY
              <ExternalLink className="size-3.5" />
            </a>
          </div>
          <Button onClick={() => setKeyDialogOpen(true)} disabled={keyPending}>
            <Plus />
            Créer une nouvelle clé
          </Button>
        </div>

        {apiKeysLoading ? (
          <p className="text-sm text-muted-foreground">Chargement des clés…</p>
        ) : apiKeys.length === 0 ? (
          <p className="rounded-lg border border-dashed border-border p-4 text-sm text-muted-foreground">Aucune clé API créée.</p>
        ) : (
          <ul className="divide-y divide-border rounded-lg border border-border">
            {apiKeys.map((key) => {
              const revoked = Boolean(key.revoked_at)
              return (
                <li key={key.id} className="flex flex-wrap items-center justify-between gap-3 p-3">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <strong className="text-sm">{key.name}</strong>
                      {revoked && <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">Révoquée</span>}
                    </div>
                    <code className="text-xs text-muted-foreground">{key.prefix}••••••••</code>
                    <p className="text-xs text-muted-foreground">
                      Créée le {formatAPIKeyDate(key.created_at)}
                      {key.last_used_at ? ` · Dernière utilisation le ${formatAPIKeyDate(key.last_used_at)}` : " · Jamais utilisée"}
                    </p>
                  </div>
                  {!revoked && (
                    <Button variant="outline" disabled={keyPending} onClick={() => void onRevokeAPIKey(key.id)}>
                      Révoquer
                    </Button>
                  )}
                </li>
              )
            })}
          </ul>
        )}
      </Row>

      <Row id="s-data" title="Mes données" description="Vos données vous appartiennent (RGPD).">
        <p className="text-sm">Téléchargez en un fichier JSON votre profil, la fiche Entreprise, vos clients et tous vos documents.</p>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <Link to="/confidentialite" className={cn(textLinkClass, "text-[13px]")}>
            Lire la politique de confidentialité
          </Link>
          <Button variant="outline" disabled={pending} onClick={() => void onExport()}>
            <Download />
            Exporter mes données
          </Button>
        </div>
      </Row>

      <Row
        id="s-danger"
        title="Zone sensible"
        description="Action définitive."
        danger
        boxClassName="border-[color-mix(in_oklab,var(--destructive)_35%,var(--border))] bg-[color-mix(in_oklab,var(--destructive)_5%,var(--card))]"
      >
        <p className="text-sm">
          Supprimer le compte efface vos clients, vos brouillons et vos documents. Pensez à exporter vos factures émises : vous
          devez les conserver dix ans.
        </p>
        <Button variant="destructive" className="self-end" aria-haspopup="dialog" onClick={() => setConfirmOpen(true)}>
          Supprimer mon compte
        </Button>
      </Row>

      <Dialog
        open={keyDialogOpen}
        onOpenChange={(open) => {
          if (!open && keyPending) return
          if (open) setKeyDialogOpen(true)
          else closeKeyDialog()
        }}
        title={createdKey ? "Votre nouvelle clé API" : "Créer une clé API"}
        description={
          createdKey
            ? "Copiez cette clé maintenant : elle ne sera plus jamais affichée."
            : "Donnez-lui un nom qui permet d’identifier son usage."
        }
        icon={<KeyRound />}
      >
        {createdKey ? (
          <>
            <div className="rounded-lg border border-brand/30 bg-brand/5 p-3">
              <code className="block break-all text-sm font-semibold">{createdKey}</code>
            </div>
            <p className="text-sm font-medium text-destructive">Cette clé ne sera plus jamais affichée après la fermeture de cette fenêtre.</p>
            <div className="flex flex-wrap justify-end gap-2">
              <Button variant="outline" onClick={() => void copyCreatedKey()}>
                <Copy />
                Copier
              </Button>
              <Button onClick={closeKeyDialog}>J’ai copié la clé</Button>
            </div>
          </>
        ) : (
          <>
            <Label>
              <span>Nom de la clé</span>
              <Input autoFocus value={keyName} maxLength={100} placeholder="Ex. MCP préproduction" onChange={(event) => setKeyName(event.target.value)} />
            </Label>
            <p className="text-xs text-muted-foreground">La clé autorisera la lecture et l’écriture via l’API publique.</p>
            <div className="flex justify-end">
              <Button disabled={keyPending || !keyName.trim()} onClick={() => void onCreateAPIKey()}>
                Créer la clé
              </Button>
            </div>
          </>
        )}
      </Dialog>

      <Dialog
        open={confirmOpen}
        onOpenChange={(open) => {
          setConfirmOpen(open)
          setConfirmText("")
          setPassword("")
        }}
        title="Supprimer le compte ?"
        description="Cette action est irréversible. Exportez d’abord vos données si vous devez conserver vos factures."
      >
        <Label>
          <span>
            Saisissez <strong>{CONFIRM_WORD}</strong> pour confirmer
          </span>
          <Input autoComplete="off" value={confirmText} onChange={(event) => setConfirmText(event.target.value)} />
        </Label>
        <Label>
          <span>Mot de passe</span>
          <PasswordInput autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} />
        </Label>
        <div className="flex flex-wrap justify-end gap-2">
          <Button variant="outline" size="lg" disabled={pending} onClick={() => void onExport()}>
            Exporter avant
          </Button>
          <Button variant="destructive" size="lg" disabled={pending || confirmText.trim() !== CONFIRM_WORD || !password} onClick={() => void onDelete()}>
            Supprimer définitivement
          </Button>
        </div>
      </Dialog>
    </div>
  )
}
