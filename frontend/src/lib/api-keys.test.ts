import { afterEach, describe, expect, it, vi } from "vitest"

import { createAPIKey, listAPIKeys, revokeAPIKey } from "@/lib/api"

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("API key client", () => {
  it("lists API keys returned by the authenticated endpoint", async () => {
    const apiKeys = [
      {
        id: "key-1",
        name: "MCP préproduction",
        prefix: "abcdefgh",
        scopes: ["read", "write"],
        created_at: "2026-11-13T08:00:00Z",
        last_used_at: "2026-11-14T11:34:56Z",
      },
    ]
    const fetchMock = vi.fn().mockResolvedValue(Response.json({ api_keys: apiKeys }))
    vi.stubGlobal("fetch", fetchMock)

    await expect(listAPIKeys()).resolves.toEqual(apiKeys)
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/auth/api-keys",
      expect.objectContaining({ credentials: "include" }),
    )
  })

  it("creates a read/write key and returns its one-time secret", async () => {
    const created = {
      api_key: {
        id: "key-2",
        name: "MCP local",
        prefix: "ijklmnop",
        scopes: ["read", "write"],
        created_at: "2026-11-14T12:00:00Z",
      },
      key: "secret-visible-once",
    }
    const fetchMock = vi.fn().mockResolvedValue(Response.json(created, { status: 201 }))
    vi.stubGlobal("fetch", fetchMock)

    await expect(createAPIKey("MCP local")).resolves.toEqual(created)
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/auth/api-keys",
      expect.objectContaining({ method: "POST", body: JSON.stringify({ name: "MCP local", scopes: ["read", "write"] }) }),
    )
  })

  it("revokes the selected key", async () => {
    const fetchMock = vi.fn().mockResolvedValue(Response.json({ status: "revoked" }))
    vi.stubGlobal("fetch", fetchMock)

    await expect(revokeAPIKey("key/with spaces")).resolves.toBeUndefined()
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/auth/api-keys/key%2Fwith%20spaces",
      expect.objectContaining({ method: "DELETE" }),
    )
  })
})
