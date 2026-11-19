import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { RemindersSettings } from "@/components/reminders-settings"
import { ToastContext } from "@/components/use-toast"
import * as api from "@/lib/reminders-api"

vi.mock("@/lib/reminders-api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/reminders-api")>()
  return {
    ...actual,
    getDeliveryMode: vi.fn(),
    setDeliveryMode: vi.fn(),
    getOAuthStatus: vi.fn(),
    startOAuth: vi.fn(),
    disconnectOAuth: vi.fn(),
    getSMTPConfig: vi.fn(),
    saveSMTPConfig: vi.fn(),
    deleteSMTPConfig: vi.fn(),
    listReminderTemplates: vi.fn(),
    createReminderTemplate: vi.fn(),
    deleteReminderTemplate: vi.fn(),
    redirectTo: vi.fn(),
  }
})

const push = vi.fn()
const template = { id: "t1", name: "Relance courtoise", subject: "Facture en attente", html: "<p>Bonjour</p>", created_at: "", updated_at: "" }

function renderSettings() {
  return render(
    <ToastContext.Provider value={{ push }}>
      <RemindersSettings />
    </ToastContext.Provider>,
  )
}

beforeEach(() => {
  vi.resetAllMocks()
  push.mockReset()
  vi.mocked(api.getDeliveryMode).mockResolvedValue(null)
  vi.mocked(api.getOAuthStatus).mockResolvedValue("disconnected")
  vi.mocked(api.getSMTPConfig).mockResolvedValue(null)
  vi.mocked(api.listReminderTemplates).mockResolvedValue([])
})

describe("RemindersSettings", () => {
  it("says that nothing can be sent while no delivery mode is chosen, then saves the chosen mode", async () => {
    vi.mocked(api.setDeliveryMode).mockResolvedValue("smtp")
    const user = userEvent.setup()
    renderSettings()

    expect(await screen.findByText(/Aucun mode choisi/)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "SMTP" }))

    await waitFor(() => expect(api.setDeliveryMode).toHaveBeenCalledWith("smtp"))
    expect(await screen.findByText(/jamais depuis une adresse générique/)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "SMTP" })).toHaveAttribute("aria-pressed", "true")
  })

  it("shows each OAuth provider state and starts the connection by redirecting to the authorization URL", async () => {
    vi.mocked(api.getOAuthStatus).mockImplementation(async (p) => (p === "gmail" ? "disconnected" : "unavailable"))
    vi.mocked(api.startOAuth).mockResolvedValue("https://accounts.google.com/auth?x=1")
    const user = userEvent.setup()
    renderSettings()

    expect(await screen.findByText(/OAuth non configuré/)).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /Connecter Microsoft/ })).not.toBeInTheDocument()
    await user.click(await screen.findByRole("button", { name: "Connecter Gmail" }))

    await waitFor(() => expect(api.redirectTo).toHaveBeenCalledWith("https://accounts.google.com/auth?x=1"))
  })

  it("offers to disconnect a connected account", async () => {
    vi.mocked(api.getOAuthStatus).mockImplementation(async (p) => (p === "gmail" ? "connected" : "disconnected"))
    vi.mocked(api.disconnectOAuth).mockResolvedValue()
    const user = userEvent.setup()
    renderSettings()

    await user.click(await screen.findByRole("button", { name: "Déconnecter" }))
    await waitFor(() => expect(api.disconnectOAuth).toHaveBeenCalledWith("gmail"))
    expect(await screen.findByRole("button", { name: "Connecter Gmail" })).toBeInTheDocument()
  })

  it("keeps the SMTP save button disabled until the form is complete, then sends it", async () => {
    vi.mocked(api.saveSMTPConfig).mockResolvedValue({ host: "smtp.example.test", port: 587, username: "user", from_address: "moi@example.test", tls_mode: "starttls", password_configured: true })
    const user = userEvent.setup()
    renderSettings()

    const form = await screen.findByRole("form", { name: "Configuration SMTP" })
    const save = within(form).getByRole("button", { name: "Enregistrer et tester" })
    expect(save).toBeDisabled()
    await user.type(within(form).getByLabelText("Serveur"), "smtp.example.test")
    await user.type(within(form).getByLabelText("Identifiant"), "user")
    await user.type(within(form).getByLabelText("Mot de passe"), "secret")
    await user.type(within(form).getByLabelText("Adresse d’expédition"), "moi@example.test")
    expect(save).toBeEnabled()
    await user.click(save)

    await waitFor(() =>
      expect(api.saveSMTPConfig).toHaveBeenCalledWith({ host: "smtp.example.test", port: 587, username: "user", password: "secret", from_address: "moi@example.test", tls_mode: "starttls" }),
    )
    expect(within(form).getByLabelText("Mot de passe")).toHaveValue("")
    expect(await within(form).findByRole("button", { name: "Supprimer" })).toBeInTheDocument()
  })

  it("lists templates, creates one and deletes one", async () => {
    vi.mocked(api.listReminderTemplates).mockResolvedValue([template])
    vi.mocked(api.createReminderTemplate).mockResolvedValue({ ...template, id: "t2", name: "Relance ferme" })
    vi.mocked(api.deleteReminderTemplate).mockResolvedValue()
    const user = userEvent.setup()
    renderSettings()

    expect(await screen.findByText("Relance courtoise")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "Nouveau modèle" }))
    await user.type(await screen.findByLabelText("Nom"), "Relance ferme")
    await user.type(screen.getByLabelText("Objet"), "Dernier rappel")
    await user.type(screen.getByLabelText("Message (HTML)"), "Merci de régler.")
    await user.click(screen.getByRole("button", { name: "Créer le modèle" }))

    await waitFor(() => expect(api.createReminderTemplate).toHaveBeenCalledWith({ name: "Relance ferme", subject: "Dernier rappel", html: "Merci de régler." }))
    expect(await screen.findByText("Relance ferme")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "Supprimer le modèle Relance courtoise" }))
    await waitFor(() => expect(api.deleteReminderTemplate).toHaveBeenCalledWith("t1"))
    await waitFor(() => expect(screen.queryByText("Relance courtoise")).not.toBeInTheDocument())
  })
})
