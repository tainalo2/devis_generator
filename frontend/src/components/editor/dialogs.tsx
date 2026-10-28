import { Building2, User } from "lucide-react"
import { useState, type FormEvent } from "react"

import { ClientFields, IssuerFields } from "@/components/party-fields"
import { Button } from "@/components/ui/button"
import { Dialog } from "@/components/ui/dialog"
import type { Client, IssuerProfile } from "@/lib/types"

import { ZoneIcon } from "./zones"

type Base<T> = {
  open: boolean
  onOpenChange: (open: boolean) => void
  initial: T
  /** Enregistre (compte) ou applique localement (invité). Rejette en cas d'erreur. */
  onSubmit: (value: T) => Promise<void>
  guest: boolean
}

function useSubmit<T>(value: T, onSubmit: (v: T) => Promise<void>, close: () => void) {
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setPending(true)
    setError(null)
    try {
      await onSubmit(value)
      close()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Enregistrement impossible.")
    } finally {
      setPending(false)
    }
  }
  return { pending, error, submit }
}

function ClientForm({ initial, onSubmit, onOpenChange, guest }: Omit<Base<Client>, "open">) {
  const [value, setValue] = useState(initial)
  const { pending, error, submit } = useSubmit(value, onSubmit, () => onOpenChange(false))
  return (
    <form id="client-form" onSubmit={(e) => void submit(e)} className="contents">
      <ClientFields value={value} onChange={setValue} />
      {!guest && <p className="text-xs text-muted-foreground">Le client est ajouté à votre liste de clients.</p>}
      {error && <p className="text-sm text-destructive">{error}</p>}
      <SubmitFooter pending={pending} label={initial.id ? "Enregistrer" : "Ajouter le client"} onCancel={() => onOpenChange(false)} />
    </form>
  )
}

export function ClientDialog({ open, onOpenChange, ...rest }: Base<Client>) {
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      icon={<ZoneIcon icon={User} tone="parties" className="size-7 rounded-[7px]" />}
      title={rest.initial.id || rest.initial.name ? "Modifier le client" : "Nouveau client"}
      description="Recherchez l’entreprise pour pré-remplir ses informations, ou saisissez-les."
    >
      {open && <ClientForm onOpenChange={onOpenChange} {...rest} />}
    </Dialog>
  )
}

function IssuerForm({ initial, onSubmit, onOpenChange, guest }: Omit<Base<IssuerProfile>, "open">) {
  const [value, setValue] = useState(initial)
  const { pending, error, submit } = useSubmit(value, onSubmit, () => onOpenChange(false))
  return (
    <form onSubmit={(e) => void submit(e)} className="contents">
      <IssuerFields value={value} onChange={setValue} />
      <p className="text-xs text-muted-foreground">
        {guest
          ? "En mode invité, ces informations restent dans ce navigateur et ne sont pas envoyées à nos serveurs, sauf pour générer le PDF."
          : "Les modifications sont enregistrées dans la fiche Entreprise et reprises par les prochains documents. Un document déjà émis garde l’émetteur de son émission."}
      </p>
      {error && <p className="text-sm text-destructive">{error}</p>}
      <SubmitFooter pending={pending} label="Enregistrer" onCancel={() => onOpenChange(false)} />
    </form>
  )
}

export function IssuerDialog({ open, onOpenChange, ...rest }: Base<IssuerProfile>) {
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      icon={<ZoneIcon icon={Building2} tone="parties" className="size-7 rounded-[7px]" />}
      title="Modifier l’émetteur"
      description="Ces informations apparaissent en tête du document."
    >
      {open && <IssuerForm onOpenChange={onOpenChange} {...rest} />}
    </Dialog>
  )
}

/** Pied collant dans le corps défilant de la fenêtre, pour rester dans le formulaire. */
function SubmitFooter({ pending, label, onCancel }: { pending: boolean; label: string; onCancel: () => void }) {
  return (
    <div className="sticky -bottom-5 -mx-5 -mb-5 mt-1 flex justify-end gap-2 border-t border-border bg-card px-5 py-3">
      <Button variant="ghost" size="lg" onClick={onCancel}>
        Annuler
      </Button>
      <Button type="submit" size="lg" disabled={pending}>
        {label}
      </Button>
    </div>
  )
}
