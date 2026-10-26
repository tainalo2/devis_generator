import type { ComponentProps } from "react"

import { cn } from "@/lib/utils"

/** Liste déroulante native, mêmes bordures et focus que `Input`. */
export function Select({ className, ...props }: ComponentProps<"select">) {
  return (
    <select
      className={cn(
        "flex h-9 w-full cursor-pointer rounded-md border border-input bg-background px-3 text-sm transition-colors",
        "disabled:cursor-not-allowed disabled:opacity-50",
        "focus-visible:border-ring",
        className,
      )}
      {...props}
    />
  )
}
