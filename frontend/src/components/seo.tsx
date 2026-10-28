import { useEffect } from "react"

type Props = {
  title?: string
  description?: string
  path?: string
  noIndex?: boolean
}

const SITE = "Devis Generator : VIBED"
const DEFAULT_DESC =
  "Devis et factures pour micro-entrepreneurs, indépendants et petites sociétés. Gratuit, mode invité, mentions légales et sources officielles."

export function Seo({ title, description = DEFAULT_DESC, path = "/", noIndex }: Props) {
  const fullTitle = title ? `${title} · ${SITE}` : SITE

  useEffect(() => {
    document.title = fullTitle
    setMeta("description", description)
    setMeta("og:title", fullTitle, "property")
    setMeta("og:description", description, "property")
    setMeta("og:type", "website", "property")
    setMeta("og:locale", "fr_FR", "property")
    const origin = window.location.origin
    setMeta("og:url", `${origin}${path}`, "property")
    setLink("canonical", `${origin}${path}`)
    if (noIndex) setMeta("robots", "noindex,nofollow")
    else setMeta("robots", "index,follow")
  }, [fullTitle, description, path, noIndex])

  return null
}

function setMeta(name: string, content: string, attr: "name" | "property" = "name") {
  let el = document.head.querySelector(`meta[${attr}="${name}"]`) as HTMLMetaElement | null
  if (!el) {
    el = document.createElement("meta")
    el.setAttribute(attr, name)
    document.head.appendChild(el)
  }
  el.content = content
}

function setLink(rel: string, href: string) {
  let el = document.head.querySelector(`link[rel="${rel}"]`) as HTMLLinkElement | null
  if (!el) {
    el = document.createElement("link")
    el.rel = rel
    document.head.appendChild(el)
  }
  el.href = href
}
