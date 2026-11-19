import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router-dom"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { SendReminderDialog } from "@/components/send-reminder-dialog"
import { ToastContext } from "@/components/use-toast"
import * as api from "@/lib/reminders-api"

vi.mock("@/lib/reminders-api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/reminders-api")>()
  return { ...actual, listReminderTemplates: vi.fn(), getDeliveryMode: vi.fn(), sendReminder: vi.fn() }
})

const push = vi.fn()
const onClose = vi.fn()
const tpl = (id: string, name: string) => ({ id, name, subject: "S", html: "<p>x</p>", created_at: "", updated_at: "" })

function renderDialog() {
  return render(
    <MemoryRouter>
      <ToastContext.Provider value={{ push }}>
        <SendReminderDialog invoiceId="inv-1" invoiceName="Facture F-2026-0001" onClose={onClose} />
      </ToastContext.Provider>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  vi.resetAllMocks()
})

describe("SendReminderDialog", () => {
  it("sends the chosen template with an idempotency key and closes", async () => {
    vi.mocked(api.listReminderTemplates).mockResolvedValue([tpl("t1", "Courtoise"), tpl("t2", "Ferme")])
    vi.mocked(api.getDeliveryMode).mockResolvedValue("smtp")
    vi.mocked(api.sendReminder).mockResolvedValue({ id: "r1", invoice_id: "inv-1", template_id: "t2", type: "manual", recipient: "client@example.test", status: "sent", created_at: "", attempt_count: 1 })
    const user = userEvent.setup()
    renderDialog()

    await user.selectOptions(await screen.findByLabelText("Modèle"), "t2")
    await user.click(screen.getByRole("button", { name: "Envoyer la relance" }))

    await waitFor(() => expect(api.sendReminder).toHaveBeenCalledTimes(1))
    const [invoice, template, key] = vi.mocked(api.sendReminder).mock.calls[0]
    expect([invoice, template]).toEqual(["inv-1", "t2"])
    expect(key).toMatch(/^[0-9a-f-]{36}$/)
    expect(push).toHaveBeenCalledWith("Relance envoyée à client@example.test.", "success")
    expect(onClose).toHaveBeenCalled()
  })

  it("blocks sending and points to the settings when no delivery mode is configured", async () => {
    vi.mocked(api.listReminderTemplates).mockResolvedValue([tpl("t1", "Courtoise")])
    vi.mocked(api.getDeliveryMode).mockResolvedValue(null)
    renderDialog()

    expect(await screen.findByText(/Aucun mode d’envoi configuré/)).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "Configurer l’envoi des relances" })).toHaveAttribute("href", "/compte#s-relances")
    expect(screen.getByRole("button", { name: "Envoyer la relance" })).toBeDisabled()
  })

  it("blocks sending when there is no template", async () => {
    vi.mocked(api.listReminderTemplates).mockResolvedValue([])
    vi.mocked(api.getDeliveryMode).mockResolvedValue("gmail")
    renderDialog()

    expect(await screen.findByText(/Aucun modèle de relance/)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Envoyer la relance" })).toBeDisabled()
  })

  it("keeps the dialog open and shows the server message when the send fails, reusing the same key on retry", async () => {
    vi.mocked(api.listReminderTemplates).mockResolvedValue([tpl("t1", "Courtoise")])
    vi.mocked(api.getDeliveryMode).mockResolvedValue("smtp")
    vi.mocked(api.sendReminder)
      .mockImplementationOnce(() => new Promise((_, reject) => setTimeout(() => reject(new Error("Limite horaire de relances atteinte.")), 0)))
      .mockResolvedValueOnce({ id: "r1", invoice_id: "inv-1", template_id: "t1", type: "manual", recipient: "c@example.test", status: "sent", created_at: "", attempt_count: 1 })
    const user = userEvent.setup()
    renderDialog()

    await user.click(await screen.findByRole("button", { name: "Envoyer la relance" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("Limite horaire de relances atteinte.")
    expect(onClose).not.toHaveBeenCalled()

    await user.click(screen.getByRole("button", { name: "Envoyer la relance" }))
    await waitFor(() => expect(api.sendReminder).toHaveBeenCalledTimes(2))
    const calls = vi.mocked(api.sendReminder).mock.calls
    expect(calls[0][2]).toBe(calls[1][2])
  })
})
