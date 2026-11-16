import { Mail, Plus, Trash2, X } from "lucide-react"
import { useEffect, useState, type ReactNode } from "react"

import { Button } from "@/components/ui/button"
import { Dialog } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { useToast } from "@/components/use-toast"
import {
  listCollaborators,
  listPendingInvitations,
  inviteCollaborator,
  revokeCollaborator,
  cancelInvitation,
  type Collaborator,
  type Invitation,
} from "@/lib/api"
import { cn } from "@/lib/utils"

function Row({
  id,
  title,
  description,
  danger,
  children,
  boxClassName,
}: {
  id: string
  title: string
  description: string
  danger?: boolean
  children: ReactNode
  boxClassName?: string
}) {
  return (
    <section aria-labelledby={id} className="grid gap-x-8 gap-y-3 border-t border-border py-6 md:grid-cols-[240px_minmax(0,1fr)]">
      <div className="flex flex-col gap-1">
        <h2
          id={id}
          className={cn("font-heading text-lg leading-[26px] font-semibold tracking-tight", danger && "text-destructive")}
        >
          {title}
        </h2>
        <p className="text-[13px] leading-[18px] text-muted-foreground">{description}</p>
      </div>
      <div className={cn("flex flex-col gap-3.5 rounded-xl border border-border bg-card p-4", boxClassName)}>
        {children}
      </div>
    </section>
  )
}

export function CollaboratorsSection() {
  const toast = useToast()
  const [collaborators, setCollaborators] = useState<Collaborator[]>([])
  const [invitations, setInvitations] = useState<Invitation[]>([])
  const [loading, setLoading] = useState(true)
  const [inviteDialogOpen, setInviteDialogOpen] = useState(false)
  const [inviteEmail, setInviteEmail] = useState("")
  const [invitePending, setInvitePending] = useState(false)

  // Load collaborators and invitations on mount
  useEffect(() => {
    void loadData()
  }, [])

  async function loadData() {
    setLoading(true)
    try {
      const [collabs, invites] = await Promise.all([listCollaborators(), listPendingInvitations()])
      setCollaborators(collabs || [])
      setInvitations(invites || [])
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Chargement impossible.", "error")
    } finally {
      setLoading(false)
    }
  }

  async function onInvite() {
    const email = inviteEmail.trim()
    if (!email) {
      toast.push("Veuillez entrer une adresse email.", "error")
      return
    }

    setInvitePending(true)
    try {
      await inviteCollaborator(email)
      toast.push(`Invitation envoyée à ${email}.`, "success")
      setInviteEmail("")
      setInviteDialogOpen(false)
      void loadData()
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Invitation impossible.", "error")
    } finally {
      setInvitePending(false)
    }
  }

  async function onRevoke(collaboratorId: string) {
    if (!confirm("Êtes-vous sûr ? Ce collaborateur perdra l'accès au compte.")) return

    try {
      await revokeCollaborator(collaboratorId)
      toast.push("Collaborateur révoqué.", "success")
      void loadData()
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Révocation impossible.", "error")
    }
  }

  async function onCancelInvitation(invitationId: string) {
    if (!confirm("Êtes-vous sûr ? Cette invitation ne sera plus valide.")) return

    try {
      await cancelInvitation(invitationId)
      toast.push("Invitation annulée.", "success")
      void loadData()
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Annulation impossible.", "error")
    }
  }

  return (
    <Row
      id="s-collaborators"
      title="Collaborateurs"
      description="Partagez le compte avec d'autres utilisateurs."
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <p className="max-w-xl text-sm">
          Les collaborateurs ont accès à vos documents et clients. Les propriétaires peuvent gérer les collaborateurs et
          modifier la configuration du compte.
        </p>
        <Button onClick={() => setInviteDialogOpen(true)} disabled={invitePending}>
          <Plus />
          Inviter
        </Button>
      </div>

      {/* Collaborators List */}
      {loading ? (
        <p className="text-sm text-muted-foreground">Chargement…</p>
      ) : collaborators.length === 0 ? (
        <p className="rounded-lg border border-dashed border-border p-4 text-sm text-muted-foreground">
          Aucun collaborateur. Invitez quelqu'un pour partager le compte.
        </p>
      ) : (
        <ul className="divide-y divide-border rounded-lg border border-border">
          {collaborators.map((collab) => (
            <li key={collab.id} className="flex flex-wrap items-center justify-between gap-3 p-3">
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <strong className="text-sm">{collab.email}</strong>
                  <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground capitalize">
                    {collab.role === "owner" ? "Propriétaire" : "Collaborateur"}
                  </span>
                </div>
                <p className="text-xs text-muted-foreground">Depuis le {new Date(collab.created_at).toLocaleDateString("fr-FR")}</p>
              </div>
              {collab.role !== "owner" && (
                <Button
                  variant="outline"
                  size="sm"
                  disabled={invitePending}
                  onClick={() => void onRevoke(collab.id)}
                >
                  <Trash2 className="size-4" />
                  Révoquer
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}

      {/* Pending Invitations */}
      {invitations.length > 0 && (
        <div className="rounded-lg border border-dashed border-blue-200 bg-blue-50 p-3 dark:border-blue-900 dark:bg-blue-950">
          <p className="mb-2 text-xs font-semibold text-blue-900 dark:text-blue-100">Invitations en attente</p>
          <ul className="space-y-2">
            {invitations.map((invite) => (
              <li key={invite.id} className="flex items-center justify-between gap-2">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-1 text-xs">
                    <Mail className="size-3" />
                    <span className="text-blue-900 dark:text-blue-100">{invite.email}</span>
                  </div>
                  <p className="text-xs text-blue-800 dark:text-blue-200">
                    Expire le {new Date(invite.expires_at).toLocaleDateString("fr-FR")}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => void onCancelInvitation(invite.id)}
                  disabled={invitePending}
                >
                  <X className="size-4" />
                </Button>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Invite Dialog */}
      <Dialog
        open={inviteDialogOpen}
        onOpenChange={(open) => {
          if (!open && invitePending) return
          setInviteDialogOpen(open)
          if (!open) setInviteEmail("")
        }}
        title="Inviter un collaborateur"
        description="Entrez l'adresse email de la personne à inviter."
        icon={<Mail />}
      >
        <Label>
          <span>Adresse email</span>
          <Input
            autoFocus
            type="email"
            value={inviteEmail}
            placeholder="collaborateur@example.com"
            onChange={(event) => setInviteEmail(event.target.value)}
          />
        </Label>
        <p className="text-xs text-muted-foreground">
          Un email d'invitation avec un lien d'acceptation sera envoyé.
        </p>
        <div className="flex justify-end gap-2">
          <Button
            variant="outline"
            disabled={invitePending}
            onClick={() => {
              setInviteDialogOpen(false)
              setInviteEmail("")
            }}
          >
            Annuler
          </Button>
          <Button disabled={invitePending || !inviteEmail.trim()} onClick={() => void onInvite()}>
            Envoyer l'invitation
          </Button>
        </div>
      </Dialog>
    </Row>
  )
}
