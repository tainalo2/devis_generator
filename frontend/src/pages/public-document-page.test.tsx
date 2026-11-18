import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter, Route, Routes } from "react-router-dom"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { PublicDocumentPage } from "@/pages/public-document-page"
import * as api from "@/lib/api"

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return { ...actual, fetchPublicDocumentPDF: vi.fn(), fetchPublicSignatureState: vi.fn(), signPublicDocument: vi.fn() }
})

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/d/signed-token"]}>
      <Routes>
        <Route path="/d/:token" element={<PublicDocumentPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  vi.mocked(api.fetchPublicDocumentPDF).mockReset()
  vi.mocked(api.fetchPublicSignatureState).mockReset()
  vi.mocked(api.signPublicDocument).mockReset()
  vi.mocked(api.fetchPublicDocumentPDF).mockResolvedValue(new Blob(["%PDF-test"], { type: "application/pdf" }))
  vi.mocked(api.fetchPublicSignatureState).mockResolvedValue({ signed: false, signable: false })
  vi.stubGlobal("URL", {
    ...URL,
    createObjectURL: vi.fn(() => "blob:public-document"),
    revokeObjectURL: vi.fn(),
  })
})

describe("Public document page", () => {
  it("renders the PDF returned for the signed token", async () => {
    vi.mocked(api.fetchPublicDocumentPDF).mockResolvedValue(new Blob(["%PDF-test"], { type: "application/pdf" }))

    renderPage()

    expect(screen.getByText("Chargement du document…")).toBeInTheDocument()
    const frame = await screen.findByTitle("Document partagé")
    expect(frame).toHaveAttribute("data", "blob:public-document")
    expect(api.fetchPublicDocumentPDF).toHaveBeenCalledWith("signed-token")
  })

  it("shows a readable error with one generic message for invalid, expired or unknown links", async () => {
    vi.mocked(api.fetchPublicDocumentPDF).mockRejectedValue(new Error(api.PUBLIC_LINK_UNAVAILABLE))

    renderPage()

    expect(await screen.findByRole("heading", { name: "Document indisponible" })).toBeInTheDocument()
    expect(screen.getByText(api.PUBLIC_LINK_UNAVAILABLE)).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByText("Chargement du document…")).not.toBeInTheDocument())
  })

  it("offers the signature only on a signable quote and posts the declared name", async () => {
    vi.mocked(api.fetchPublicSignatureState).mockResolvedValue({ signed: false, signable: true })
    vi.mocked(api.signPublicDocument).mockResolvedValue({ signed_at: "2026-11-18T10:00:00Z", signer_name: "Jeanne Client" })
    const user = userEvent.setup()
    renderPage()

    await user.click(await screen.findByRole("button", { name: "Signer ce devis" }))
    const submit = screen.getByRole("button", { name: "Signer le devis" })
    expect(submit).toBeDisabled()
    await user.type(screen.getByLabelText("Votre nom et prénom"), "  Jeanne Client ")
    await user.click(submit)

    await waitFor(() => expect(api.signPublicDocument).toHaveBeenCalledWith("signed-token", "Jeanne Client"))
    expect(await screen.findByText(/Devis signé par Jeanne Client/)).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "Signer ce devis" })).not.toBeInTheDocument()
  })

  it("keeps the dialog open and shows the server message when signing fails", async () => {
    vi.mocked(api.fetchPublicSignatureState).mockResolvedValue({ signed: false, signable: true })
    vi.mocked(api.signPublicDocument).mockRejectedValue(new Error("Ce devis est déjà signé."))
    const user = userEvent.setup()
    renderPage()

    await user.click(await screen.findByRole("button", { name: "Signer ce devis" }))
    await user.type(screen.getByLabelText("Votre nom et prénom"), "Jeanne Client")
    await user.click(screen.getByRole("button", { name: "Signer le devis" }))

    expect(await screen.findByRole("alert")).toHaveTextContent("Ce devis est déjà signé.")
    expect(screen.queryByText(/Devis signé par/)).not.toBeInTheDocument()
  })

  it("shows no signature button for a document that is not signable and a banner when already signed", async () => {
    renderPage()
    await screen.findByTitle("Document partagé")
    expect(screen.queryByRole("button", { name: "Signer ce devis" })).not.toBeInTheDocument()
  })

  it("shows who signed and when for an already signed quote", async () => {
    vi.mocked(api.fetchPublicSignatureState).mockResolvedValue({ signed: true, signable: false, signed_at: "2026-11-18T10:00:00Z", signer_name: "Jeanne Client" })
    renderPage()
    expect(await screen.findByText(/Devis signé par Jeanne Client/)).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "Signer ce devis" })).not.toBeInTheDocument()
  })
})
