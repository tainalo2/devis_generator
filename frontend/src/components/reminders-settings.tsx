import { Plus, Trash2 } from "lucide-react"
import { useEffect, useState, type FormEvent } from "react"

import { PasswordInput } from "@/components/auth-ui"
import { Button } from "@/components/ui/button"
import { Dialog } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Segmented } from "@/components/ui/segmented"
import { useToast } from "@/components/use-toast"
import {
  createReminderTemplate,
  deleteReminderTemplate,
  deleteSMTPConfig,
  disconnectOAuth,
  getDeliveryMode,
  getOAuthStatus,
  getSMTPConfig,
  listReminderTemplates,
  redirectTo,
  saveSMTPConfig,
  setDeliveryMode,
  startOAuth,
  type DeliveryMode,
  type OAuthProvider,
  type OAuthStatus,
  type ReminderTemplate,
  type SMTPConfig,
  type TLSMode,
} from "@/lib/reminders-api"

const PROVIDER_LABEL: Record<OAuthProvider, string> = { gmail: "Gmail", microsoft: "Microsoft 365 / Outlook" }

function errorMessage(err: unknown, fallback: string) {
  return err instanceof Error ? err.message : fallback
}

function OAuthCard({ provider }: { provider: OAuthProvider }) {
  const toast = useToast()
  const [status, setStatus] = useState<OAuthStatus | null>(null)
  const [pending, setPending] = useState(false)

  useEffect(() => {
    let active = true
    void getOAuthStatus(provider)
      .then((s) => active && setStatus(s))
      .catch((err: unknown) => {
        if (!active) return
        setStatus("unavailable")
        toast.push(errorMessage(err, "Impossible de vérifier la connexion."), "error")
      })
    return () => {
      active = false
    }
  }, [provider, toast])

  async function connect() {
    setPending(true)
    try {
      redirectTo(await startOAuth(provider))
    } catch (err) {
      toast.push(errorMessage(err, "Impossible de démarrer la connexion."), "error")
      setPending(false)
    }
  }

  async function disconnect() {
    setPending(true)
    try {
      await disconnectOAuth(provider)
      setStatus("disconnected")
      toast.push(`${PROVIDER_LABEL[provider]} déconnecté.`, "success")
    } catch (err) {
      toast.push(errorMessage(err, "Impossible de déconnecter le compte."), "error")
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border p-3">
      <div className="flex flex-col gap-0.5">
        <strong className="text-sm">{PROVIDER_LABEL[provider]}</strong>
        <span className="text-xs text-muted-foreground">
          {status === null && "Vérification…"}
          {status === "connected" && "Compte connecté : les relances partent de votre adresse."}
          {status === "disconnected" && "Non connecté."}
          {status === "unavailable" && "Connexion indisponible sur ce serveur (OAuth non configuré)."}
        </span>
      </div>
      {status === "connected" && (
        <Button variant="outline" disabled={pending} onClick={() => void disconnect()}>
          Déconnecter
        </Button>
      )}
      {status === "disconnected" && (
        <Button disabled={pending} onClick={() => void connect()}>
          Connecter {PROVIDER_LABEL[provider]}
        </Button>
      )}
    </div>
  )
}

function SMTPForm() {
  const toast = useToast()
  const [loaded, setLoaded] = useState(false)
  const [config, setConfig] = useState<SMTPConfig | null>(null)
  const [host, setHost] = useState("")
  const [port, setPort] = useState("587")
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [from, setFrom] = useState("")
  const [tls, setTLS] = useState<TLSMode>("starttls")
  const [pending, setPending] = useState(false)

  function fill(c: SMTPConfig) {
    setConfig(c)
    setHost(c.host)
    setPort(String(c.port))
    setUsername(c.username)
    setFrom(c.from_address)
    setTLS(c.tls_mode)
  }

  useEffect(() => {
    let active = true
    void getSMTPConfig()
      .then((c) => active && c && fill(c))
      .catch((err: unknown) => active && toast.push(errorMessage(err, "Impossible de charger la configuration SMTP."), "error"))
      .finally(() => active && setLoaded(true))
    return () => {
      active = false
    }
  }, [toast])

  const portNumber = Number(port)
  const valid = host.trim() !== "" && Number.isInteger(portNumber) && portNumber > 0 && portNumber < 65536 && username.trim() !== "" && from.trim() !== "" && password !== ""

  async function save(e: FormEvent) {
    e.preventDefault()
    if (!valid || pending) return
    setPending(true)
    try {
      const saved = await saveSMTPConfig({ host: host.trim(), port: portNumber, username: username.trim(), password, from_address: from.trim(), tls_mode: tls })
      fill(saved)
      setPassword("")
      toast.push("Configuration SMTP enregistrée (connexion testée).", "success")
    } catch (err) {
      toast.push(errorMessage(err, "Impossible d’enregistrer la configuration SMTP."), "error")
    } finally {
      setPending(false)
    }
  }

  async function remove() {
    setPending(true)
    try {
      await deleteSMTPConfig()
      setConfig(null)
      setHost("")
      setUsername("")
      setFrom("")
      setPassword("")
      toast.push("Configuration SMTP supprimée.", "success")
    } catch (err) {
      toast.push(errorMessage(err, "Impossible de supprimer la configuration SMTP."), "error")
    } finally {
      setPending(false)
    }
  }

  if (!loaded) return <p className="text-sm text-muted-foreground">Chargement de la configuration SMTP…</p>
  return (
    <form onSubmit={(e) => void save(e)} className="flex flex-col gap-3 rounded-lg border border-border p-3" aria-label="Configuration SMTP">
      <div className="flex flex-col gap-0.5">
        <strong className="text-sm">SMTP manuel</strong>
        <span className="text-xs text-muted-foreground">
          {config ? "Configuré. Saisissez à nouveau le mot de passe pour modifier la configuration." : "Non configuré. La connexion est testée à l’enregistrement."}
        </span>
      </div>
      <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_110px]">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="smtp-host">Serveur</Label>
          <Input id="smtp-host" value={host} onChange={(e) => setHost(e.target.value)} placeholder="smtp.exemple.fr" autoComplete="off" />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="smtp-port">Port</Label>
          <Input id="smtp-port" inputMode="numeric" value={port} onChange={(e) => setPort(e.target.value)} />
        </div>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="smtp-user">Identifiant</Label>
          <Input id="smtp-user" value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="off" />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="smtp-password">Mot de passe</Label>
          <PasswordInput id="smtp-password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="new-password" placeholder={config?.password_configured ? "Mot de passe enregistré" : ""} />
        </div>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="smtp-from">Adresse d’expédition</Label>
          <Input id="smtp-from" type="email" value={from} onChange={(e) => setFrom(e.target.value)} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Chiffrement</Label>
          <Segmented<TLSMode>
            label="Chiffrement"
            value={tls}
            onChange={setTLS}
            stretch
            options={[
              { value: "starttls", label: "STARTTLS" },
              { value: "tls", label: "TLS direct" },
            ]}
          />
        </div>
      </div>
      <div className="flex flex-wrap justify-end gap-2">
        {config && (
          <Button type="button" variant="outline" disabled={pending} onClick={() => void remove()}>
            Supprimer
          </Button>
        )}
        <Button type="submit" disabled={!valid || pending}>
          {pending ? "Test de connexion…" : "Enregistrer et tester"}
        </Button>
      </div>
    </form>
  )
}

function Templates() {
  const toast = useToast()
  const [templates, setTemplates] = useState<ReminderTemplate[] | null>(null)
  const [open, setOpen] = useState(false)
  const [name, setName] = useState("")
  const [subject, setSubject] = useState("")
  const [html, setHTML] = useState("")
  const [pending, setPending] = useState(false)

  useEffect(() => {
    let active = true
    void listReminderTemplates()
      .then((list) => active && setTemplates(list))
      .catch((err: unknown) => {
        if (!active) return
        setTemplates([])
        toast.push(errorMessage(err, "Impossible de charger les modèles."), "error")
      })
    return () => {
      active = false
    }
  }, [toast])

  const valid = name.trim() !== "" && subject.trim() !== "" && html.trim() !== ""

  async function create(e: FormEvent) {
    e.preventDefault()
    if (!valid || pending) return
    setPending(true)
    try {
      const created = await createReminderTemplate({ name: name.trim(), subject: subject.trim(), html })
      setTemplates((prev) => [created, ...(prev ?? [])])
      setOpen(false)
      setName("")
      setSubject("")
      setHTML("")
      toast.push("Modèle de relance créé.", "success")
    } catch (err) {
      toast.push(errorMessage(err, "Impossible de créer le modèle."), "error")
    } finally {
      setPending(false)
    }
  }

  async function remove(id: string) {
    setPending(true)
    try {
      await deleteReminderTemplate(id)
      setTemplates((prev) => (prev ?? []).filter((t) => t.id !== id))
      toast.push("Modèle supprimé.", "success")
    } catch (err) {
      toast.push(errorMessage(err, "Impossible de supprimer le modèle."), "error")
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <strong className="text-sm">Modèles de relance</strong>
        <Button variant="outline" onClick={() => setOpen(true)}>
          <Plus />
          Nouveau modèle
        </Button>
      </div>
      {templates === null ? (
        <p className="text-sm text-muted-foreground">Chargement des modèles…</p>
      ) : templates.length === 0 ? (
        <p className="rounded-lg border border-dashed border-border p-4 text-sm text-muted-foreground">Aucun modèle. Créez-en un pour pouvoir relancer une facture.</p>
      ) : (
        <ul className="divide-y divide-border rounded-lg border border-border">
          {templates.map((t) => (
            <li key={t.id} className="flex items-center justify-between gap-3 p-3">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{t.name}</p>
                <p className="truncate text-xs text-muted-foreground">{t.subject}</p>
              </div>
              <Button variant="ghost" size="icon" aria-label={`Supprimer le modèle ${t.name}`} disabled={pending} onClick={() => void remove(t.id)}>
                <Trash2 />
              </Button>
            </li>
          ))}
        </ul>
      )}
      <Dialog
        open={open}
        onOpenChange={setOpen}
        title="Nouveau modèle de relance"
        description="Le contenu HTML est envoyé tel quel au client."
        footer={
          <>
            <Button variant="ghost" type="button" onClick={() => setOpen(false)}>
              Annuler
            </Button>
            <Button type="submit" form="template-form" disabled={!valid || pending}>
              Créer le modèle
            </Button>
          </>
        }
      >
        <form id="template-form" onSubmit={(e) => void create(e)} className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="tpl-name">Nom</Label>
            <Input id="tpl-name" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="tpl-subject">Objet</Label>
            <Input id="tpl-subject" value={subject} onChange={(e) => setSubject(e.target.value)} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="tpl-html">Message (HTML)</Label>
            <textarea
              id="tpl-html"
              value={html}
              onChange={(e) => setHTML(e.target.value)}
              rows={6}
              className="rounded-md border border-input bg-background p-3 text-sm focus-visible:border-ring"
            />
          </div>
        </form>
      </Dialog>
    </div>
  )
}

/** Configuration de l'envoi des relances : mode d'envoi, comptes Gmail/Microsoft, SMTP, modèles. */
export function RemindersSettings() {
  const toast = useToast()
  const [mode, setMode] = useState<DeliveryMode | null>(null)
  const [modeLoaded, setModeLoaded] = useState(false)

  useEffect(() => {
    let active = true
    void getDeliveryMode()
      .then((m) => active && setMode(m))
      .catch((err: unknown) => active && toast.push(errorMessage(err, "Impossible de charger le mode d’envoi."), "error"))
      .finally(() => active && setModeLoaded(true))
    return () => {
      active = false
    }
  }, [toast])

  async function chooseMode(next: DeliveryMode) {
    try {
      setMode(await setDeliveryMode(next))
      toast.push("Mode d’envoi enregistré.", "success")
    } catch (err) {
      toast.push(errorMessage(err, "Impossible de configurer le mode d’envoi."), "error")
    }
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col gap-2">
        <strong className="text-sm">Mode d’envoi</strong>
        {modeLoaded && (
          <Segmented<DeliveryMode>
            label="Mode d’envoi des relances"
            value={mode ?? ("" as DeliveryMode)}
            onChange={(m) => void chooseMode(m)}
            options={[
              { value: "gmail", label: "Gmail" },
              { value: "microsoft", label: "Microsoft" },
              { value: "smtp", label: "SMTP" },
            ]}
          />
        )}
        <p className="text-xs text-muted-foreground">
          {mode ? "Les relances sont envoyées depuis le compte choisi, jamais depuis une adresse générique." : "Aucun mode choisi : aucune relance ne peut partir tant que ce n’est pas configuré."}
        </p>
      </div>
      <OAuthCard provider="gmail" />
      <OAuthCard provider="microsoft" />
      <SMTPForm />
      <Templates />
    </div>
  )
}
