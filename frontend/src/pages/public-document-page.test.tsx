import { render, screen, waitFor } from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router-dom"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { PublicDocumentPage } from "@/pages/public-document-page"
import * as api from "@/lib/api"

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return { ...actual, fetchPublicDocumentPDF: vi.fn() }
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

  it("shows a readable error when the token is invalid or expired", async () => {
    vi.mocked(api.fetchPublicDocumentPDF).mockRejectedValue(new Error("Ce lien est invalide ou a expiré."))

    renderPage()

    expect(await screen.findByRole("heading", { name: "Document indisponible" })).toBeInTheDocument()
    expect(screen.getByText("Ce lien est invalide ou a expiré.")).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByText("Chargement du document…")).not.toBeInTheDocument())
  })
})
