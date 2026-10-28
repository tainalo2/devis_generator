import { Menu as MenuPrimitive } from "@base-ui/react/menu"
import {
  Building2,
  ChevronDown,
  CircleUserRound,
  FileText,
  LayoutDashboard,
  LogOut,
  Menu,
  Plus,
  Scale,
  Users,
  X,
  type LucideIcon,
} from "lucide-react"
import { useEffect, useState } from "react"
import { Link, NavLink, useLocation, useNavigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { LogoMark } from "@/components/logo"
import { ThemeToggle } from "@/components/theme-toggle"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { menuItemClass, menuPopupClass } from "@/lib/styles"
import { cn } from "@/lib/utils"

type NavItem = { to: string; label: string; icon: LucideIcon; end?: boolean }

const mainLinks: NavItem[] = [
  { to: "/tableau-de-bord", label: "Tableau de bord", icon: LayoutDashboard },
  { to: "/documents", label: "Documents", icon: FileText },
  { to: "/clients", label: "Clients", icon: Users },
]

const accountLinks: NavItem[] = [
  { to: "/entreprise", label: "Entreprise", icon: Building2 },
  { to: "/sources-legales", label: "Sources légales", icon: Scale },
  { to: "/compte", label: "Paramètres du compte", icon: CircleUserRound },
]

const guestLinks: NavItem[] = [
  { to: "/documents/nouveau", label: "Créer un devis", icon: FileText, end: true },
  { to: "/sources-legales", label: "Sources légales", icon: Scale },
]

const navClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "inline-flex h-8 items-center rounded-md px-3 text-sm font-medium whitespace-nowrap transition-colors",
    isActive
      ? "bg-secondary text-secondary-foreground"
      : "text-muted-foreground hover:bg-muted/80 hover:text-foreground",
  )

const mobileItemClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "flex h-12 items-center gap-3 rounded-lg px-3 text-[15px] font-medium transition-colors [&_svg]:size-[18px] [&_svg]:shrink-0",
    isActive
      ? "bg-secondary text-secondary-foreground [&_svg]:text-brand"
      : "text-foreground hover:bg-muted/80 [&_svg]:text-muted-foreground",
  )

/** « Nouveau document » ne doit pas rendre « Documents » actif. */
function isDocumentsActive(pathname: string) {
  return pathname.startsWith("/documents") && pathname !== "/documents/nouveau"
}

function Brand({ onClick }: { onClick?: () => void }) {
  return (
    <Link to="/" onClick={onClick} className="flex min-w-0 shrink-0 items-center gap-2 rounded-md lg:gap-2.5">
      <LogoMark />
      <span className="font-heading text-base font-semibold tracking-tight whitespace-nowrap lg:text-[17px]">
        Devis Generator : VIBED
      </span>
    </Link>
  )
}

function AccountMenu({ onLogout }: { onLogout: () => void }) {
  return (
    <MenuPrimitive.Root>
      <MenuPrimitive.Trigger render={<Button variant="outline" />}>
        <CircleUserRound />
        Mon compte
        <ChevronDown />
      </MenuPrimitive.Trigger>
      <MenuPrimitive.Portal>
        <MenuPrimitive.Positioner side="bottom" align="end" sideOffset={8} className="z-50">
          <MenuPrimitive.Popup className={cn(menuPopupClass, "w-60")}>
            {accountLinks.map((l) => (
              <MenuPrimitive.LinkItem key={l.to} className={menuItemClass} render={<Link to={l.to} />}>
                <l.icon />
                {l.label}
              </MenuPrimitive.LinkItem>
            ))}
            <Separator className="my-1" />
            <MenuPrimitive.Item className={menuItemClass} onClick={onLogout}>
              <LogOut />
              Se déconnecter
            </MenuPrimitive.Item>
          </MenuPrimitive.Popup>
        </MenuPrimitive.Positioner>
      </MenuPrimitive.Portal>
    </MenuPrimitive.Root>
  )
}

export function SiteHeader() {
  const { user, loading, logout } = useAuth()
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const [open, setOpen] = useState(false)
  const close = () => setOpen(false)

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false)
    }
    window.addEventListener("keydown", onKey)
    return () => window.removeEventListener("keydown", onKey)
  }, [open])

  const onLogout = async () => {
    close()
    await logout()
    navigate("/")
  }

  const desktopLinks = user ? mainLinks : guestLinks

  return (
    <header className="sticky top-0 z-40 border-b border-border bg-background/85 backdrop-blur">
      <div className="mx-auto flex h-15 w-full max-w-7xl items-center gap-6 pr-2 pl-4 lg:h-16 lg:px-6">
        <Brand onClick={close} />

        <nav aria-label={user ? "Navigation principale" : "Navigation publique"} className="hidden min-w-0 flex-1 items-center gap-1 lg:flex">
          {!loading &&
            desktopLinks.map((l) => (
              <NavLink
                key={l.to}
                to={l.to}
                end={l.end}
                className={({ isActive }) =>
                  navClass({ isActive: l.to === "/documents" ? isDocumentsActive(pathname) : isActive })
                }
              >
                {l.label}
              </NavLink>
            ))}
        </nav>

        <div className="ml-auto hidden shrink-0 items-center gap-2 lg:flex">
          {loading ? null : user ? (
            <>
              <Button render={<Link to="/documents/nouveau" />}>
                <Plus />
                Nouveau document
              </Button>
              <ThemeToggle />
              <Separator orientation="vertical" className="h-5 self-center" />
              <AccountMenu onLogout={onLogout} />
            </>
          ) : (
            <>
              <Button variant="ghost" render={<Link to="/connexion" />}>
                Connexion
              </Button>
              <Button render={<Link to="/inscription" />}>Créer un compte</Button>
              <Separator orientation="vertical" className="h-5 self-center" />
              <ThemeToggle />
            </>
          )}
        </div>

        <div className="ml-auto flex shrink-0 items-center gap-0.5 lg:hidden">
          <ThemeToggle className="size-11" />
          <Button
            variant="outline"
            size="icon"
            className="size-11"
            aria-label={open ? "Fermer le menu" : "Ouvrir le menu"}
            aria-expanded={open}
            aria-controls="menu-mobile"
            onClick={() => setOpen((v) => !v)}
          >
            {open ? <X /> : <Menu />}
          </Button>
        </div>
      </div>

      {open && (
        <div className="fixed inset-x-0 top-15 bottom-0 z-40 flex flex-col lg:hidden">
          <div id="menu-mobile" className="flex flex-col gap-3 border-b border-border bg-card px-3 pt-3 pb-4 shadow-lg">
            {user ? (
              <>
                <nav aria-label="Navigation principale" className="flex flex-col gap-0.5">
                  {mainLinks.map((l) => (
                    <NavLink
                      key={l.to}
                      to={l.to}
                      onClick={close}
                      className={({ isActive }) =>
                        mobileItemClass({ isActive: l.to === "/documents" ? isDocumentsActive(pathname) : isActive })
                      }
                    >
                      <l.icon />
                      {l.label}
                    </NavLink>
                  ))}
                </nav>
                <Separator />
                <nav aria-label="Compte" className="flex flex-col gap-0.5">
                  {accountLinks.map((l) => (
                    <NavLink key={l.to} to={l.to} onClick={close} className={mobileItemClass}>
                      <l.icon />
                      {l.label}
                    </NavLink>
                  ))}
                  <button type="button" className={cn(mobileItemClass({ isActive: false }), "text-left")} onClick={onLogout}>
                    <LogOut />
                    Se déconnecter
                  </button>
                </nav>
                <Separator />
                <Button size="lg" className="h-11 w-full" render={<Link to="/documents/nouveau" onClick={close} />}>
                  <Plus />
                  Nouveau document
                </Button>
              </>
            ) : (
              <>
                <nav aria-label="Navigation publique" className="flex flex-col gap-0.5">
                  {guestLinks.map((l) => (
                    <NavLink key={l.to} to={l.to} end={l.end} onClick={close} className={mobileItemClass}>
                      <l.icon />
                      {l.label}
                    </NavLink>
                  ))}
                </nav>
                <Separator />
                <div className="flex flex-col gap-2">
                  <Button size="lg" className="h-11 w-full" render={<Link to="/inscription" onClick={close} />}>
                    Créer un compte
                  </Button>
                  <Button variant="outline" size="lg" className="h-11 w-full" render={<Link to="/connexion" onClick={close} />}>
                    Connexion
                  </Button>
                </div>
              </>
            )}
          </div>
          <button
            type="button"
            aria-label="Fermer le menu"
            tabIndex={-1}
            className="flex-1 cursor-pointer bg-foreground/30"
            onClick={close}
          />
        </div>
      )}
    </header>
  )
}
