import { PenLine } from "lucide-react"
import { useState, type FormEvent } from "react"

import { Button } from "@/components/ui/button"
import { Dialog } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { signPublicDocument, type PublicSignResult } from "@/lib/api"

type Props = {
  token: string
  open: boolean
  onOpenChange: (open: boolean) => void
  onSigned: (result: PublicSignResult) => void
}

/** Modale de signature : le signataire déclare son nom ; heure, IP et empreinte sont captées côté serveur. */
export function SignatureDialog({ token, open, onOpenChange, onSigned }: Props) {
  const [name, setName] = useState("")
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const valid = name.trim().length >= 2

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!valid || pending) return
    setPending(true)
    setError(null)
    try {
      onSigned(await signPublicDocument(token, name.trim()))
    } catch (err) {
      setError(err instanceof Error ? err.message : "Signature impossible.")
    } finally {
      setPending(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Signer ce devis"
      icon={<PenLine className="size-5" aria-hidden />}
      description="Signature électronique simple : en signant, vous acceptez ce devis."
      footer={
        <>
          <Button variant="ghost" type="button" onClick={() => onOpenChange(false)}>
            Annuler
          </Button>
          <Button type="submit" form="signature-form" disabled={!valid || pending}>
            {pending ? "Signature…" : "Signer le devis"}
          </Button>
        </>
      }
    >
      <form id="signature-form" onSubmit={(e) => void submit(e)} className="flex flex-col gap-3">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="signer-name">Votre nom et prénom</Label>
          <Input id="signer-name" value={name} maxLength={200} autoComplete="name" onChange={(e) => setName(e.target.value)} />
        </div>
        <p className="text-xs text-muted-foreground">
          Sont enregistrés à la signature : la date et l’heure, votre adresse IP, votre navigateur et l’empreinte du document. Le nom saisi n’est pas vérifié.
        </p>
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
      </form>
    </Dialog>
  )
}
