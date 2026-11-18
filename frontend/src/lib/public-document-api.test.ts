import { afterEach, describe, expect, it, vi } from "vitest"

import { fetchPublicDocumentPDF, PUBLIC_LINK_UNAVAILABLE } from "@/lib/api"

afterEach(() => vi.unstubAllGlobals())

describe("fetchPublicDocumentPDF", () => {
  it.each([
    [404, "lien invalide"],
    [410, "lien expire"],
    [404, "Lien indisponible."],
  ])("maps status %i to the same generic message", async (status, error) => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({ error }), { status })))
    await expect(fetchPublicDocumentPDF("tok")).rejects.toThrow(PUBLIC_LINK_UNAVAILABLE)
  })
})

describe("signature api", () => {
  it("posts only the declared name and maps unusable links to the generic message", async () => {
    const fetchMock = vi.fn(async () => new Response(JSON.stringify({ signed_at: "2026-11-18T10:00:00Z", signer_name: "Jeanne" }), { status: 201 }))
    vi.stubGlobal("fetch", fetchMock)
    const { signPublicDocument } = await import("@/lib/api")
    await signPublicDocument("tok/en", "Jeanne")
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe("/api/v1/public/documents/tok%2Fen/sign")
    expect(init.method).toBe("POST")
    expect(JSON.parse(init.body as string)).toEqual({ name: "Jeanne" })

    vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({ error: "Lien indisponible." }), { status: 404 })))
    await expect(signPublicDocument("tok", "Jeanne")).rejects.toThrow(PUBLIC_LINK_UNAVAILABLE)
  })

  it("surfaces the server message on 409", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({ error: "Ce devis est déjà signé." }), { status: 409 })))
    const { signPublicDocument } = await import("@/lib/api")
    await expect(signPublicDocument("tok", "Jeanne")).rejects.toThrow("Ce devis est déjà signé.")
  })
})
