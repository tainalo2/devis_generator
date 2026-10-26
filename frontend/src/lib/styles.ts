/** Classes partagées hors composants (Tailwind). */

export const textLinkClass = "text-sm font-medium text-brand underline-offset-4 hover:underline"

/** Tableau de données : en-têtes discrets, filets, survol. */
export const table = {
  root: "w-full border-collapse text-sm",
  th: "border-b border-border px-4 py-2.5 text-left text-xs font-medium whitespace-nowrap text-muted-foreground",
  td: "border-b border-border px-4 py-3 whitespace-nowrap [tr:last-child_&]:border-b-0",
  row: "transition-colors hover:bg-muted/60",
}

/** Élément de menu déroulant (Base UI Menu). */
export const menuItemClass =
  "flex h-9 cursor-default items-center gap-2.5 rounded-md px-2.5 text-sm whitespace-nowrap text-foreground outline-none select-none data-highlighted:bg-muted [&_svg]:size-4 [&_svg]:shrink-0 [&_svg]:text-muted-foreground"

export const menuPopupClass =
  "flex flex-col gap-0.5 rounded-lg border border-border bg-popover p-1.5 text-popover-foreground shadow-lg outline-none"
