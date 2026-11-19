import { afterEach, describe, expect, it, vi } from "vitest"

import {
  deleteSMTPConfig,
  getDeliveryMode,
  getOAuthStatus,
  getSMTPConfig,
  listReminderTemplates,
  saveSMTPConfig,
  sendReminder,
  startOAuth,
} from "@/lib/reminders-api"

afterEach(() => vi.unstubAllGlobals())

function stub(status: number, body: unknown) {
  const fn = vi.fn(async () => new Response(JSON.stringify(body), { status }))
  vi.stubGlobal("fetch", fn)
  return fn
}

describe("reminders api", () => {
  it("maps 404 to «not configured» for the delivery mode and the SMTP config", async () => {
    stub(404, { error: "Mode d’envoi non configuré." })
    expect(await getDeliveryMode()).toBeNull()
    stub(404, { error: "Configuration SMTP introuvable." })
    expect(await getSMTPConfig()).toBeNull()
  })

  it("surfaces the server message on other errors", async () => {
    stub(502, { error: "Le test SMTP a échoué." })
    await expect(saveSMTPConfig({ host: "h", port: 587, username: "u", password: "p", from_address: "a@b.c", tls_mode: "starttls" })).rejects.toThrow("Le test SMTP a échoué.")
    stub(500, {})
    await expect(deleteSMTPConfig()).rejects.toThrow("Impossible de supprimer la configuration SMTP.")
  })

  it("reads the SMTP config from the {config} envelope and never needs a password back", async () => {
    stub(200, { config: { host: "smtp.example.test", port: 587, username: "u", from_address: "a@example.test", tls_mode: "starttls", password_configured: true } })
    const cfg = await getSMTPConfig()
    expect(cfg?.password_configured).toBe(true)
    expect(cfg).not.toHaveProperty("password")
  })

  it("reports OAuth as unavailable on 503 and connected/disconnected otherwise", async () => {
    stub(503, { error: "OAuth Gmail indisponible." })
    expect(await getOAuthStatus("gmail")).toBe("unavailable")
    stub(200, { connected: true })
    expect(await getOAuthStatus("microsoft")).toBe("connected")
    stub(200, { connected: false })
    expect(await getOAuthStatus("gmail")).toBe("disconnected")
  })

  it("only accepts https authorization URLs", async () => {
    stub(200, { authorization_url: "https://accounts.google.com/o/oauth2/v2/auth?x=1" })
    expect(await startOAuth("gmail")).toMatch(/^https:\/\/accounts\.google\.com/)
    for (const bad of ["javascript:alert(1)", "http://evil.example", "data:text/html,x", ""]) {
      stub(200, { authorization_url: bad })
      await expect(startOAuth("gmail")).rejects.toThrow("Adresse d’autorisation invalide.")
    }
  })

  it("lists templates and tolerates a null list", async () => {
    stub(200, { templates: null })
    expect(await listReminderTemplates()).toEqual([])
    stub(200, { templates: [{ id: "t1", name: "N", subject: "S", html: "<p>x</p>", created_at: "", updated_at: "" }] })
    expect((await listReminderTemplates())[0].id).toBe("t1")
  })

  it("sends a reminder with an Idempotency-Key header and the ids", async () => {
    const fn = stub(200, { reminder: { id: "r1", status: "sent", recipient: "client@example.test" } })
    const r = await sendReminder("inv/1", "tpl-1", "key-123")
    expect(r.status).toBe("sent")
    const [url, init] = fn.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe("/api/v1/documents/inv%2F1/reminders")
    expect(new Headers(init.headers).get("Idempotency-Key")).toBe("key-123")
    expect(JSON.parse(init.body as string)).toEqual({ invoice_id: "inv/1", template_id: "tpl-1" })
  })

  it("surfaces rate limit and delivery failure messages", async () => {
    stub(429, { error: "Limite horaire de relances atteinte." })
    await expect(sendReminder("i", "t", "k")).rejects.toThrow("Limite horaire de relances atteinte.")
    stub(502, { reminder: { id: "r" }, error: "Échec de livraison." })
    await expect(sendReminder("i", "t", "k")).rejects.toThrow("Échec de livraison.")
  })
})
