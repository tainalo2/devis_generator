import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router-dom"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { AuthContext, type AuthContextValue } from "@/components/use-auth"
import { ToastContext } from "@/components/use-toast"
import { AccountPage } from "@/pages/account-page"
import * as api from "@/lib/api"

vi.mock("next-themes", () => ({
  useTheme: () => ({ theme: "system", setTheme: vi.fn() }),
}))

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>()
  return {
    ...actual,
    listAPIKeys: vi.fn(),
    createAPIKey: vi.fn(),
    revokeAPIKey: vi.fn(),
  }
})

const key: api.APIKey = {
  id: "key-1",
  name: "MCP préproduction",
  prefix: "abcdefgh",
  scopes: ["read", "write"],
  created_at: "2026-11-13T08:00:00Z",
  last_used_at: "2026-11-14T11:34:56Z",
}

const auth: AuthContextValue = {
  user: {
    id: "user-1",
    email: "personne@example.test",
    display_name: "Personne Test",
    is_admin: false,
    created_at: "2026-01-01T00:00:00Z",
  },
  loading: false,
  refresh: vi.fn(),
  login: vi.fn(),
  register: vi.fn(),
  logout: vi.fn(),
  exportData: vi.fn(),
  removeAccount: vi.fn(),
}

function renderPage() {
  return render(
    <MemoryRouter>
      <AuthContext.Provider value={auth}>
        <ToastContext.Provider value={{ push: vi.fn() }}>
          <AccountPage />
        </ToastContext.Provider>
      </AuthContext.Provider>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  vi.mocked(api.listAPIKeys).mockReset().mockResolvedValue([key])
  vi.mocked(api.createAPIKey).mockReset()
  vi.mocked(api.revokeAPIKey).mockReset().mockResolvedValue()
})

describe("Account API keys", () => {
  it("lists only masked key metadata and the MCP setup link", async () => {
    renderPage()

    expect(await screen.findByText("MCP préproduction")).toBeInTheDocument()
    expect(screen.getByText("abcdefgh••••••••")).toBeInTheDocument()
    expect(screen.getByText(/Créée le 13 novembre 2026/)).toBeInTheDocument()
    expect(screen.getByText(/Dernière utilisation le 14 novembre 2026/)).toBeInTheDocument()
    expect(screen.queryByText("secret-visible-once")).not.toBeInTheDocument()
    expect(screen.getByRole("link", { name: /installer le serveur MCP/i })).toHaveAttribute(
      "href",
      "https://github.com/tainalo2/devis_generator/blob/main/docs/mcp.md",
    )
  })

  it("shows a newly created secret once and never adds it to the key list", async () => {
    vi.mocked(api.createAPIKey).mockResolvedValue({
      api_key: { ...key, id: "key-2", name: "MCP local", prefix: "ijklmnop" },
      key: "secret-visible-once",
    })
    const user = userEvent.setup()
    renderPage()

    await screen.findByText("MCP préproduction")
    await user.click(screen.getByRole("button", { name: /créer une nouvelle clé/i }))
    await user.type(screen.getByLabelText("Nom de la clé"), "MCP local")
    await user.click(screen.getByRole("button", { name: /^Créer la clé$/i }))

    const dialog = await screen.findByRole("dialog")
    expect(within(dialog).getByText("secret-visible-once")).toBeInTheDocument()
    expect(within(dialog).getAllByText(/ne sera plus jamais affichée/i)).not.toHaveLength(0)
    expect(vi.mocked(api.createAPIKey)).toHaveBeenCalledWith("MCP local")
    expect(screen.getAllByText("MCP local")).toHaveLength(1)

    await user.click(within(dialog).getByRole("button", { name: "J’ai copié la clé" }))
    expect(screen.queryByText("secret-visible-once")).not.toBeInTheDocument()
  })

  it("keeps the dialog open while creation is pending so the one-time secret cannot be lost", async () => {
    let resolveCreation!: (value: api.CreatedAPIKey) => void
    vi.mocked(api.createAPIKey).mockReturnValue(
      new Promise((resolve) => {
        resolveCreation = resolve
      }),
    )
    const user = userEvent.setup()
    renderPage()

    await screen.findByText("MCP préproduction")
    await user.click(screen.getByRole("button", { name: /créer une nouvelle clé/i }))
    await user.type(screen.getByLabelText("Nom de la clé"), "MCP en attente")
    await user.click(screen.getByRole("button", { name: /^Créer la clé$/i }))
    await user.click(screen.getByRole("button", { name: "Fermer" }))

    expect(screen.getByRole("dialog")).toBeInTheDocument()

    resolveCreation({
      api_key: { ...key, id: "key-pending", name: "MCP en attente", prefix: "qrstuvwx" },
      key: "pending-secret",
    })
    expect(await screen.findByText("pending-secret")).toBeInTheDocument()
  })

  it("revokes an active key and marks it revoked", async () => {
    const user = userEvent.setup()
    renderPage()

    const name = await screen.findByText("MCP préproduction")
    const row = name.closest("li")
    expect(row).not.toBeNull()
    await user.click(within(row!).getByRole("button", { name: "Révoquer" }))

    await waitFor(() => expect(api.revokeAPIKey).toHaveBeenCalledWith("key-1"))
    expect(within(row!).getByText("Révoquée")).toBeInTheDocument()
    expect(within(row!).queryByRole("button", { name: "Révoquer" })).not.toBeInTheDocument()
  })
})
