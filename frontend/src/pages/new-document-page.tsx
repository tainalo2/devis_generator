import { useSearchParams } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { DocumentEditor } from "@/components/editor/document-editor"
import type { EditorKind } from "@/components/editor/model"
import { Seo } from "@/components/seo"
import { Spinner } from "@/components/ui/spinner"

const kindParam: Record<string, EditorKind> = { devis: "quote", facture: "invoice", acompte: "deposit_invoice" }

export function NewDocumentPage() {
  const { user, loading } = useAuth()
  const [params] = useSearchParams()
  const guest = !user
  const requested = kindParam[params.get("type") ?? ""]
  const kind = guest && requested === "deposit_invoice" ? "quote" : requested

  return (
    <>
      <Seo title="Nouveau document" path="/documents/nouveau" />
      {loading ? (
        <div className="p-8">
          <Spinner label="Chargement de l’éditeur" />
        </div>
      ) : (
        <DocumentEditor
          key={guest ? "invite" : "compte"}
          guest={guest}
          initialKind={kind}
          initialQuoteId={params.get("devis") ?? undefined}
          initialClientId={params.get("client") ?? undefined}
        />
      )}
    </>
  )
}
