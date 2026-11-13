import { useEffect, useState } from "react"

const SCRIPT: { role: "user" | "agent"; text: string }[] = [
  { role: "user", text: "Fais-moi un devis pour M. Dubois, rénovation complète d'une salle de bain, main d'œuvre + matériaux, environ 4500€ HT." },
  { role: "agent", text: "Avec plaisir ! Pour finaliser le devis, j'ai besoin de quelques précisions : l'email de M. Dubois pour l'envoi du devis, et comment souhaitez-vous décomposer les 4500€ HT — par exemple Main d'œuvre 2500€ / Matériaux 2000€ ?" },
  { role: "user", text: "Oui exactement, son email c'est m.dubois@exemple.fr. Délai de validité : 30 jours." },
  { role: "agent", text: "Devis créé ! Voici le lien à lui envoyer :" },
]

const DEFAULT_LINK = "/d/demo"

export function DemoChat({ documentUrl = DEFAULT_LINK }: { documentUrl?: string }) {
  const [visible, setVisible] = useState(0)
  const [typing, setTyping] = useState(false)

  useEffect(() => {
    let cancelled = false
    let i = 0
    const showNext = () => {
      if (cancelled) return
      if (i >= SCRIPT.length) return
      const isAgent = SCRIPT[i].role === "agent"
      if (isAgent) setTyping(true)
      const delay = isAgent ? 900 : 400
      setTimeout(() => {
        if (cancelled) return
        setTyping(false)
        setVisible((v) => v + 1)
        i++
        showNext()
      }, delay)
    }
    const start = setTimeout(showNext, 600)
    return () => {
      cancelled = true
      clearTimeout(start)
    }
  }, [])

  return (
    <div className="rounded-2xl border border-border/70 bg-card/80 p-4 shadow-[0_30px_80px_-40px_rgba(15,40,60,0.45)] backdrop-blur">
      <div className="mb-3 flex items-center gap-2 text-xs text-muted-foreground">
        <span className="size-2 rounded-full bg-emerald-500" />
        Devis Generator · assistant
      </div>
      <div className="flex flex-col gap-3">
        {SCRIPT.slice(0, visible).map((m, idx) => (
          <div key={idx} className={`flex ${m.role === "user" ? "justify-end" : "justify-start"}`}>
            <div
              className={`max-w-[85%] rounded-2xl px-3 py-2 text-sm leading-relaxed ${
                m.role === "user" ? "bg-primary text-primary-foreground" : "bg-muted text-foreground"
              }`}
            >
              {m.text}
              {idx === SCRIPT.length - 1 && visible > idx && (
                <a
                  href={documentUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="mt-1 block font-medium text-brand underline"
                >
                  Ouvrir le devis démo →
                </a>
              )}
            </div>
          </div>
        ))}
        {typing && (
          <div className="flex justify-start">
            <div className="rounded-2xl bg-muted px-3 py-2 text-sm text-muted-foreground">
              <span className="inline-flex gap-1">
                <span className="animate-bounce">.</span>
                <span className="animate-bounce [animation-delay:0.15s]">.</span>
                <span className="animate-bounce [animation-delay:0.3s]">.</span>
              </span>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
