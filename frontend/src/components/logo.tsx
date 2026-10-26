import { cn } from "@/lib/utils"

/** Pictogramme du produit (même dessin que public/favicon.svg) : ne pas redessiner. */
export function LogoMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 64 64" fill="none" aria-hidden="true" className={cn("size-7 shrink-0", className)}>
      <rect width="64" height="64" rx="14" fill="#1b3a4b" />
      <path d="M18 16h22c2.2 0 4 1.8 4 4v28c0 1.1-.9 2-2 2H20c-2.2 0-4-1.8-4-4V20c0-2.2 1.8-4 4-4z" fill="#e8f4f2" />
      <path d="M40 16v8c0 1.1.9 2 2 2h8" stroke="#2f6f6a" strokeWidth="2" />
      <path d="M22 28h16M22 34h14M22 40h10" stroke="#1b3a4b" strokeWidth="2" strokeLinecap="round" />
      <circle cx="46" cy="46" r="10" fill="#2f9e94" />
      <path d="M42.5 46.2l2.6 2.6 4.8-5.2" stroke="#e8f4f2" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}
