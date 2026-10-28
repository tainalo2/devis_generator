import { Dialog as DialogPrimitive } from "@base-ui/react/dialog"
import { X } from "lucide-react"
import type { ReactNode } from "react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: ReactNode
  description?: ReactNode
  icon?: ReactNode
  children: ReactNode
  footer?: ReactNode
  className?: string
}

/** Fenêtre modale : en-tête, corps défilant, pied d'actions. Feuille basse sur mobile. */
export function Dialog({ open, onOpenChange, title, description, icon, children, footer, className }: Props) {
  return (
    <DialogPrimitive.Root open={open} onOpenChange={onOpenChange}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Backdrop className="fixed inset-0 z-50 bg-foreground/35" />
        <div className="pointer-events-none fixed inset-0 z-50 flex items-end justify-center px-2 pt-6 pb-2 sm:items-center sm:p-6">
          <DialogPrimitive.Popup
            className={cn(
              "pointer-events-auto flex max-h-full w-full max-w-[560px] flex-col overflow-hidden rounded-xl border border-border bg-popover text-popover-foreground shadow-[0_24px_48px_-12px_rgb(15_40_60/0.35)] outline-none sm:max-h-[calc(100vh-48px)]",
              className,
            )}
          >
            <header className="flex shrink-0 items-start justify-between gap-4 border-b border-border px-5 pt-5 pb-4">
              <div className="flex flex-col gap-1">
                <DialogPrimitive.Title className="flex items-center gap-2.5 font-heading text-xl font-semibold tracking-tight">
                  {icon}
                  {title}
                </DialogPrimitive.Title>
                {description && (
                  <DialogPrimitive.Description className="text-sm text-muted-foreground">{description}</DialogPrimitive.Description>
                )}
              </div>
              <DialogPrimitive.Close render={<Button variant="ghost" size="icon" aria-label="Fermer" />}>
                <X />
              </DialogPrimitive.Close>
            </header>
            <div className="flex min-h-0 flex-1 flex-col gap-3.5 overflow-y-auto px-5 pt-4 pb-5">{children}</div>
            {footer && (
              <footer className="flex shrink-0 justify-end gap-2 border-t border-border bg-card px-5 py-3">{footer}</footer>
            )}
          </DialogPrimitive.Popup>
        </div>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  )
}
