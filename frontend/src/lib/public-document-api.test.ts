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
