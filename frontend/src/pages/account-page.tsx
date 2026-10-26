import { Download } from "lucide-react"
import { useTheme } from "next-themes"
import { useState, type ReactNode } from "react"
import { Link, Navigate, useNavigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { PasswordInput } from "@/components/auth-ui"
import { PageHeader } from "@/components/page"
import { Seo } from "@/components/seo"
import { useToast } from "@/components/toast-provider"
import { Button } from "@/components/ui/button"
import { Dialog } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Segmented } from "@/components/ui/segmented"
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

export function AccountPage() {
  const { user, loading, logout, exportData, removeAccount } = useAuth()
  const { theme, setTheme } = useTheme()
  const navigate = useNavigate()
  const toast = useToast()
  const [pending, setPending] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [confirmText, setConfirmText] = useState("")
  const [password, setPassword] = useState("")

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
        open={confirmOpen}
        onOpenChange={(o) => {
          setConfirmOpen(o)
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
          <Input autoComplete="off" value={confirmText} onChange={(e) => setConfirmText(e.target.value)} />
        </Label>
        <Label>
          <span>Mot de passe</span>
          <PasswordInput autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} />
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
