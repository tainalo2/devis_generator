import { Eye, EyeOff } from "lucide-react"
import { useState, type ComponentProps, type ReactNode } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

export function AuthCard({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-4.5 rounded-2xl border border-border bg-card p-6 shadow-[0_30px_80px_-40px_rgba(15,40,60,0.35)] sm:p-8">
      <div className="flex flex-col gap-1.5">
        <h1 className="font-heading text-[26px] leading-8 font-semibold tracking-tight">{title}</h1>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
      {children}
    </div>
  )
}

export function FormError({ children }: { children: ReactNode }) {
  return (
    <div role="alert" className="rounded-lg bg-destructive/10 px-3 py-2.5 text-sm text-destructive dark:bg-destructive/20">
      {children}
    </div>
  )
}

/** Champ mot de passe avec bouton « Afficher ». */
export function PasswordInput({ className, ...props }: Omit<ComponentProps<"input">, "type">) {
  const [shown, setShown] = useState(false)
  return (
    <span className="relative block">
      <Input {...props} type={shown ? "text" : "password"} className={cn("pr-11", className)} />
      <span className="absolute top-0.5 right-0.5">
        <Button
          variant="ghost"
          size="icon"
          aria-label={shown ? "Masquer le mot de passe" : "Afficher le mot de passe"}
          aria-pressed={shown}
          onClick={() => setShown((s) => !s)}
        >
          {shown ? <EyeOff /> : <Eye />}
        </Button>
      </span>
    </span>
  )
}

export const authLink = "font-medium text-brand underline-offset-4 hover:underline"
