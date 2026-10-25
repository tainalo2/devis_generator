import type { ComponentProps } from "react"

import { cn } from "@/lib/utils"

export function Label({ className, ...props }: ComponentProps<"label">) {
  return (
    <label
      className={cn("block space-y-1.5 text-sm font-medium text-foreground", className)}
      {...props}
    />
  )
}
