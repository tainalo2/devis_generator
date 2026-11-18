import { render, screen } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { SignatureProofBanner } from "@/components/signature-proof-banner"
import * as api from "@/lib/api"

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return { ...actual, getSignatureProof: vi.fn() }
})

const proof = {
  document_id: "d1",
  signed_at: "2026-11-18T10:00:00Z",
  document_hash: "a".repeat(64),
  signer_name: "Jeanne Client",
  signer_ip: "203.0.113.7",
  signer_user_agent: "UA/1",
  integrity_ok: true,
}

beforeEach(() => vi.mocked(api.getSignatureProof).mockReset())

describe("SignatureProofBanner", () => {
  it("shows the proof and confirms integrity", async () => {
    vi.mocked(api.getSignatureProof).mockResolvedValue(proof)
    render(<SignatureProofBanner documentId="d1" />)
    const region = await screen.findByRole("region", { name: "Preuve de signature" })
    expect(region).toHaveTextContent("Signé électroniquement par Jeanne Client")
    expect(region).toHaveTextContent("a".repeat(64))
    expect(region).toHaveTextContent("203.0.113.7")
    expect(region).toHaveTextContent("Intégrité vérifiée")
  })

  it("warns loudly when the content no longer matches the signed hash", async () => {
    vi.mocked(api.getSignatureProof).mockResolvedValue({ ...proof, integrity_ok: false })
    render(<SignatureProofBanner documentId="d1" />)
    expect(await screen.findByText(/le contenu ne correspond plus à celui signé/)).toBeInTheDocument()
  })
})
