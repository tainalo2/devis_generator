import { Check, ChevronDown } from "lucide-react"
import { useEffect, useState, type ReactNode } from "react"

import { cn } from "@/lib/utils"

export type LegalSection = { id: string; label: string; title?: string; content: ReactNode }

/** Texte long des pages légales (style `prose-sm`, interligne 1.625). */
const prose =
  "text-sm leading-[1.625] [&_a]:font-medium [&_a]:text-brand [&_a]:underline-offset-4 [&_a:hover]:underline [&_code]:rounded [&_code]:bg-muted/50 [&_code]:px-1 [&_code]:font-mono [&_code]:text-xs [&_dd]:m-0 [&_dl]:mb-2.5 [&_dl]:grid [&_dl]:grid-cols-[auto_1fr] [&_dl]:gap-x-4 [&_dl]:gap-y-1.5 [&_dt]:text-muted-foreground [&_h2]:mt-8 [&_h2]:mb-2 [&_h2]:scroll-mt-24 [&_h2]:font-heading [&_h2]:text-xl [&_h2]:leading-7 [&_h2]:font-semibold [&_h2]:tracking-tight [&>section:first-child>h2]:mt-0 [&_li]:mb-1 [&_p]:mb-2.5 [&_p]:text-foreground/88 [&_ul]:mb-2.5 [&_ul]:list-disc [&_ul]:pl-5"

const tocLink = "rounded-md border-l-2 border-transparent px-2.5 py-1.5 text-[13px] leading-[18px] text-muted-foreground hover:bg-muted hover:text-foreground"

export function LegalLayout({
  title,
  subtitle,
  keyPoints,
  sections,
}: {
  title: string
  subtitle?: ReactNode
  keyPoints?: { title: string; text: string }[]
  sections: LegalSection[]
}) {
  const [current, setCurrent] = useState(sections[0]?.id)
  const ids = sections.map((s) => s.id).join(" ")

  useEffect(() => {
    const io = new IntersectionObserver(
      (entries) => {
        const top = entries.filter((e) => e.isIntersecting).sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)[0]
        if (top) setCurrent(top.target.id)
      },
      { rootMargin: "-80px 0px -65% 0px" },
    )
    ids.split(" ").forEach((id) => {
      const el = document.getElementById(id)
      if (el) io.observe(el)
    })
    return () => io.disconnect()
  }, [ids])

  const links = (withCurrent: boolean) =>
    sections.map((s) => (
      <a
        key={s.id}
        href={`#${s.id}`}
        aria-current={withCurrent && current === s.id ? "true" : undefined}
        className={cn(tocLink, withCurrent && current === s.id && "rounded-l-none border-primary font-medium text-foreground")}
      >
        {s.label}
      </a>
    ))

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-7 px-4 pt-6 pb-10 md:px-6 md:pt-10 md:pb-16">
      <header className="flex flex-col gap-1.5">
        <h1 className="font-heading text-[26px] leading-[1.2] font-semibold tracking-tight md:text-[34px]">{title}</h1>
        {subtitle && <p className="text-sm text-muted-foreground">{subtitle}</p>}
      </header>

      {keyPoints && (
        <div className="grid gap-3 md:grid-cols-3">
          {keyPoints.map((k) => (
            <div key={k.title} className="flex items-start gap-3 rounded-lg border border-border bg-card p-3.5">
              <i aria-hidden className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-brand/15 text-brand">
                <Check className="size-4" />
              </i>
              <span className="flex flex-col gap-0.5">
                <span className="text-sm font-semibold">{k.title}</span>
                <span className="text-[13px] leading-[18px] text-muted-foreground">{k.text}</span>
              </span>
            </div>
          ))}
        </div>
      )}

      <div className="flex flex-col items-start gap-5 md:flex-row md:gap-12">
        <nav aria-label="Sommaire" className="sticky top-24 hidden w-55 shrink-0 flex-col gap-0.5 md:flex">
          <span className="px-2.5 pb-1.5 text-[11px] leading-4 font-semibold tracking-[0.06em] text-muted-foreground uppercase">Sommaire</span>
          {links(true)}
        </nav>
        <details className="group w-full rounded-xl border border-border bg-card px-3.5 py-3 md:hidden">
          <summary className="flex cursor-pointer list-none items-center justify-between text-sm font-semibold [&::-webkit-details-marker]:hidden">
            Sommaire
            <ChevronDown className="size-4 transition-transform group-open:rotate-180" aria-hidden />
          </summary>
          <nav aria-label="Sommaire" className="flex flex-col gap-0.5 pt-2">
            {links(false)}
          </nav>
        </details>
        <article className={cn("max-w-170 min-w-0 flex-1", prose)}>
          {sections.map((s) => (
            <section key={s.id} aria-labelledby={s.id}>
              <h2 id={s.id}>{s.title ?? s.label}</h2>
              {s.content}
            </section>
          ))}
        </article>
      </div>
    </div>
  )
}
