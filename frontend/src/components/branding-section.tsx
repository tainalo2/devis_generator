import { useEffect, useState, type FormEvent } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { useToast } from "@/components/use-toast"
import { uploadBrandingLogo, getBrandingLogo, saveBrandingColors, getBrandingColors, clearBranding } from "@/lib/api"
import type { ColorScheme } from "@/lib/types"
import { cn } from "@/lib/utils"

export function BrandingSection() {
  const toast = useToast()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [logoPreview, setLogoPreview] = useState<string | null>(null)
  const [logoFile, setLogoFile] = useState<File | null>(null)
  const [logoUploading, setLogoUploading] = useState(false)
  const [colors, setColors] = useState<ColorScheme>({
    primary_color: "#2f9e94",
    accent_color: "#1a5f4a",
    text_color: "#1a202c",
  })
  const [originalColors, setOriginalColors] = useState<ColorScheme | null>(null)
  const [originalLogo, setOriginalLogo] = useState<string | null>(null)

  // Load branding data on mount
  useEffect(() => {
    void (async () => {
      try {
        setLoading(true)
        const logoRes = await getBrandingLogo()
        if (logoRes.logo_data) {
          setLogoPreview(logoRes.logo_data)
          setOriginalLogo(logoRes.logo_data)
        }
        const colorsRes = await getBrandingColors()
        if (colorsRes) {
          setColors(colorsRes)
          setOriginalColors(colorsRes)
        }
      } catch (err) {
        const msg = err instanceof Error ? err.message : "Chargement du branding impossible."
        toast.push(msg, "error")
      } finally {
        setLoading(false)
      }
    })()
  }, [toast])

  async function handleLogoChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.currentTarget.files?.[0]
    if (!file) return

    // Validate file type
    if (!["image/png", "image/jpeg"].includes(file.type)) {
      toast.push("Format accepté : PNG ou JPG.", "error")
      return
    }

    // Validate file size (2MB)
    if (file.size > 2 * 1024 * 1024) {
      toast.push("Le fichier ne doit pas dépasser 2 MB.", "error")
      return
    }

    // Create preview
    const reader = new FileReader()
    reader.onload = (event) => {
      const result = event.target?.result as string
      setLogoPreview(result)
      setLogoFile(file)
    }
    reader.readAsDataURL(file)
  }

  async function handleUploadLogo() {
    if (!logoFile) return
    try {
      setLogoUploading(true)
      const res = await uploadBrandingLogo(logoFile)
      setLogoFile(null)
      toast.push("Logo uploadé avec succès.", "success")
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Upload du logo impossible."
      toast.push(msg, "error")
      // Restore previous preview
      setLogoPreview(originalLogo || null)
    } finally {
      setLogoUploading(false)
    }
  }

  async function handleSaveColors(e: FormEvent) {
    e.preventDefault()
    try {
      setSaving(true)
      await saveBrandingColors(colors)
      setOriginalColors(colors)
      toast.push("Couleurs enregistrées.", "success")
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Enregistrement impossible."
      toast.push(msg, "error")
    } finally {
      setSaving(false)
    }
  }

  async function handleReset() {
    if (!confirm("Réinitialiser le branding (logo et couleurs) ? Cette action ne peut pas être annulée.")) {
      return
    }
    try {
      setSaving(true)
      await clearBranding()
      setLogoPreview(null)
      setOriginalLogo(null)
      setColors({
        primary_color: "#2f9e94",
        accent_color: "#1a5f4a",
        text_color: "#1a202c",
      })
      setOriginalColors(null)
      toast.push("Branding réinitialisé.", "success")
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Réinitialisation impossible."
      toast.push(msg, "error")
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <Spinner label="Chargement du branding" />
  }

  const colorChanged =
    JSON.stringify(colors) !== JSON.stringify(originalColors || { primary_color: "", accent_color: "", text_color: "" })
  const hasChanges = !!logoFile || colorChanged

  return (
    <form onSubmit={(e) => void handleSaveColors(e)} className="flex flex-col gap-6">
      {/* Logo Upload */}
      <div className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Logo de l'entreprise</h3>
        <div className="flex gap-4">
          {/* Preview */}
          <div className="flex h-20 w-20 items-center justify-center rounded-lg border-2 border-dashed border-border bg-muted">
            {logoPreview ? (
              <img src={logoPreview} alt="Aperçu du logo" className="max-h-full max-w-full" />
            ) : (
              <span className="text-xs text-muted-foreground">Pas de logo</span>
            )}
          </div>

          {/* Upload Controls */}
          <div className="flex flex-1 flex-col gap-2">
            <Label htmlFor="logo-input" className="text-xs text-muted-foreground">
              PNG ou JPG, max 2 MB
            </Label>
            <Input
              id="logo-input"
              type="file"
              accept="image/png,image/jpeg"
              onChange={(e) => void handleLogoChange(e)}
              disabled={logoUploading}
              className="text-xs"
            />
            {logoFile && (
              <Button
                type="button"
                size="sm"
                onClick={(e) => {
                  e.preventDefault()
                  void handleUploadLogo()
                }}
                disabled={logoUploading}
              >
                {logoUploading ? "Upload en cours..." : "Uploader le logo"}
              </Button>
            )}
          </div>
        </div>
        <p className="text-xs text-muted-foreground">Le logo s'affichera sur vos devis et factures.</p>
      </div>

      {/* Color Pickers */}
      <div className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold">Palette de couleurs</h3>
        <div className="grid gap-3 md:grid-cols-3">
          <ColorInput
            label="Couleur principale"
            value={colors.primary_color}
            onChange={(primary_color) => setColors((c) => ({ ...c, primary_color }))}
          />
          <ColorInput
            label="Couleur secondaire"
            value={colors.accent_color}
            onChange={(accent_color) => setColors((c) => ({ ...c, accent_color }))}
          />
          <ColorInput
            label="Couleur du texte"
            value={colors.text_color}
            onChange={(text_color) => setColors((c) => ({ ...c, text_color }))}
          />
        </div>
        <p className="text-xs text-muted-foreground">Les couleurs sont appliquées à l'en-tête, aux tableaux et aux liens.</p>
      </div>

      {/* Preview */}
      {(logoPreview || colorChanged) && (
        <div className="rounded-lg border border-border p-4">
          <h4 className="mb-3 text-sm font-semibold">Aperçu</h4>
          <div
            className="rounded-md p-4"
            style={{
              backgroundColor: colors.primary_color,
              color: colors.text_color,
            }}
          >
            <div className="flex items-center gap-3">
              {logoPreview && (
                <img src={logoPreview} alt="Logo" className="h-10 w-auto" />
              )}
              <div>
                <p className="text-sm font-semibold">Devis n°D-2026-0001</p>
                <p className="text-xs opacity-75">Exemple de document avec votre branding</p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="flex gap-2">
        <Button type="submit" disabled={!hasChanges || saving}>
          {saving ? "Enregistrement..." : "Enregistrer les modifications"}
        </Button>
        {(originalLogo || originalColors) && (
          <Button
            type="button"
            variant="outline"
            onClick={(e) => {
              e.preventDefault()
              void handleReset()
            }}
            disabled={saving}
          >
            Réinitialiser
          </Button>
        )}
      </div>
    </form>
  )
}

/**
 * Color input component with native color picker
 */
function ColorInput({ label, value, onChange }: { label: string; value: string; onChange: (color: string) => void }) {
  const [input, setInput] = useState(value)

  const normalizeHex = (hex: string): string => {
    // Remove # if present, convert to lowercase, return with #
    const clean = hex.replace(/^#/, "").toLowerCase()
    return /^[0-9a-f]{6}$/.test(clean) ? `#${clean}` : value // Fall back to original if invalid
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const hex = normalizeHex(e.target.value)
    setInput(hex)
    onChange(hex)
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const hex = e.target.value
    setInput(hex)
    const normalized = normalizeHex(hex)
    if (/^#[0-9a-f]{6}$/.test(normalized)) {
      onChange(normalized)
    }
  }

  return (
    <Label className="flex flex-col gap-2">
      <span className="text-sm">{label}</span>
      <div className="flex gap-2">
        <input
          type="color"
          value={value}
          onChange={handleChange}
          className="h-9 w-16 cursor-pointer rounded border border-input"
        />
        <Input
          type="text"
          value={input}
          onChange={handleInputChange}
          placeholder="#000000"
          maxLength={7}
          className="font-mono text-sm"
        />
      </div>
    </Label>
  )
}
