import { useSyncExternalStore } from "react"

/** Vrai tant que la requête média correspond (ex. "(min-width: 1024px)"). */
export function useMediaQuery(query: string) {
  return useSyncExternalStore(
    (cb) => {
      const m = window.matchMedia(query)
      m.addEventListener("change", cb)
      return () => m.removeEventListener("change", cb)
    },
    () => window.matchMedia(query).matches,
    () => false,
  )
}

/** Lecture/écriture localStorage tolérantes (navigation privée, stockage bloqué). */
export const localStore = {
  get<T>(key: string): T | null {
    try {
      const raw = window.localStorage.getItem(key)
      return raw ? (JSON.parse(raw) as T) : null
    } catch {
      return null
    }
  },
  set(key: string, value: unknown) {
    try {
      window.localStorage.setItem(key, JSON.stringify(value))
    } catch {
      // stockage indisponible : la donnée reste en mémoire pour la session
    }
  },
}
