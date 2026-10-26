import { useEffect, useState } from "react"
import { Link, Navigate, useParams } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { DocumentView } from "@/components/document-view"
import { DocumentEditor } from "@/components/editor/document-editor"
import { DashedEmpty } from "@/components/page"
import { Seo } from "@/components/seo"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { getDocument } from "@/lib/api"
import { documentName } from "@/lib/documents"
import type { Document } from "@/lib/types"

export function DocumentDetailPage() {
  const { id } = useParams()
  const { user, loading } = useAuth()
  const [doc, setDoc] = useState<Document | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!user || !id) return
    let cancelled = false
    void getDocument(id)
      .then((d) => !cancelled && setDoc(d))
      .catch((err: unknown) => !cancelled && setError(err instanceof Error ? err.message : "Document introuvable."))
    return () => {
      cancelled = true
    }
  }, [user, id])

  if (!loading && !user) return <Navigate to="/connexion" replace />

  if (error) {
    return (
      <div className="mx-auto w-full max-w-5xl px-4 py-8 md:px-6">
        <DashedEmpty title="Document introuvable" description={error} action={<Button render={<Link to="/documents" />}>Retour aux documents</Button>} />
      </div>
    )
  }
  // Après création, l'URL change avant le rechargement : on attend le bon document.
  if (!doc || doc.id !== id) {
    return (
      <div className="p-8">
        <Spinner label="Chargement du document" />
      </div>
    )
  }

  return (
    <>
      <Seo title={documentName(doc)} path={`/documents/${doc.id}`} noIndex />
      {doc.status === "draft" ? (
        <DocumentEditor key={doc.id} draft={doc} guest={false} onIssued={setDoc} />
      ) : (
        <DocumentView key={doc.id} doc={doc} onChange={setDoc} />
      )}
    </>
  )
}
