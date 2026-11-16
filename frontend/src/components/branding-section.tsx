import { useCallback, useRef, useState, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { useToast } from "@/components/use-toast"
import { uploadBrandingLogo, getBrandingLogo, saveBrandingColors, getBrandingColors } from "@/lib/api"
import type { ColorScheme } from "@/lib/types"

const LOGO_MAX_SIZE = 2 * 1024 * 1024 // 2MB
const LOGO_FORMATS = ["image/png", "image/jpeg"]

export function BrandingSection() {
  const toast = useToast()
  const fileInputRef = useRef<HTMLInputElement>(null)
  
  const [logoUrl, setLogoUrl] = useState<string | null>(null)
  const [logoLoading, setLogoLoading] = useState(false)
  const [colors, setColors] = useState<ColorScheme>({
    primary_color: "#2f9e94",
    accent_color: "#6366f1",
    text_color: "#1f2937",
  })
  const [colorsLoading, setColorsLoading] = useState(false)

  // Load existing branding on mount
  useEffect(() => {
    void (async () => {
      try {
        const logoData = await getBrandingLogo()
        if (logoData?.logo_url) {
          setLogoUrl(logoData.logo_url)
        }
      } catch (err) {
        console.error("Failed to load branding logo:", err)
      }

      try {
        const colorData = await getBrandingColors()
        if (colorData) {
          setColors((prev: ColorScheme) => ({ ...prev, ...colorData }))
        }
      } catch (err) {
        console.error("Failed to load branding colors:", err)
      }
    })()
  }, [])

  const validateLogoFile = (file: File): boolean => {
    if (!LOGO_FORMATS.includes(file.type)) {
      toast.push("Le format du logo doit être PNG ou JPG.", "error")
      return false
    }
    if (file.size > LOGO_MAX_SIZE) {
      toast.push("Le logo ne doit pas dépasser 2 MB.", "error")
      return false
    }
    return true
  }

  const handleLogoUpload = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.currentTarget.files?.[0]
      if (!file || !validateLogoFile(file)) return

      setLogoLoading(true)
      try {
        const result = await uploadBrandingLogo(file)
        setLogoUrl(result.logo_url)
        toast.push("Logo enregistré avec succès.", "success")
      } catch (err) {
        toast.push(err instanceof Error ? err.message : "Erreur lors de l'envoi du logo.", "error")
      } finally {
        setLogoLoading(false)
        // Reset input so same file can be re-selected
        if (fileInputRef.current) {
          fileInputRef.current.value = ""
        }
      }
    },
    [toast]
  )

  const handleColorChange = useCallback((colorType: keyof ColorScheme, value: string) => {
    setColors((prev: ColorScheme) => ({ ...prev, [colorType]: value }))
  }, [])

  const handleSaveColors = useCallback(async () => {
    setColorsLoading(true)
    try {
      const payload: Record<string, string> = {}
      if (colors.primary_color) payload.primary_color = colors.primary_color
      if (colors.accent_color) payload.accent_color = colors.accent_color
      if (colors.text_color) payload.text_color = colors.text_color

      await saveBrandingColors(payload)
      toast.push("Couleurs enregistrées avec succès.", "success")
    } catch (err) {
      toast.push(err instanceof Error ? err.message : "Erreur lors de l'enregistrement des couleurs.", "error")
    } finally {
      setColorsLoading(false)
    }
  }, [colors, toast])

  return (
    <div className="flex flex-col gap-5">
      {/* Logo Upload */}
      <div className="flex flex-col gap-3">
        <div>
          <h3 className="text-sm font-semibold">Logo de l'entreprise</h3>
          <p className="text-xs text-muted-foreground">PNG ou JPG, maximum 2 MB. Affiché sur vos documents.</p>
        </div>

        <div className="flex gap-3">
          <input
            ref={fileInputRef}
            type="file"
            accept={LOGO_FORMATS.join(",")}
            onChange={(e) => void handleLogoUpload(e)}
            disabled={logoLoading}
            className="hidden"
          />
          <Button
            type="button"
            variant="outline"
            onClick={() => fileInputRef.current?.click()}
            disabled={logoLoading}
          >
            {logoLoading ? "Envoi..." : "Choisir un fichier"}
          </Button>

          {logoUrl && (
            <div className="flex items-center gap-2 rounded-md border border-border bg-muted p-2">
              <img src={logoUrl} alt="Logo" className="h-8 w-8 object-contain" />
              <span className="text-xs text-muted-foreground">Logo chargé</span>
            </div>
          )}
        </div>
      </div>

      {/* Color Scheme */}
      <div className="flex flex-col gap-3">
        <div>
          <h3 className="text-sm font-semibold">Palette de couleurs</h3>
          <p className="text-xs text-muted-foreground">Personnalisez l'apparence de vos documents.</p>
        </div>

        <div className="grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(150px,1fr))]">
          <Label className="flex flex-col gap-1.5">
            <span className="text-xs font-medium">Couleur principale</span>
            <div className="flex gap-2">
              <input
                type="color"
                value={colors.primary_color || "#2f9e94"}
                onChange={(e) => handleColorChange("primary_color", e.target.value)}
                className="h-10 w-12 cursor-pointer rounded border border-border"
              />
              <Input
                type="text"
                placeholder="#2f9e94"
                value={colors.primary_color || ""}
                onChange={(e) => handleColorChange("primary_color", e.target.value)}
                className="text-xs"
              />
            </div>
          </Label>

          <Label className="flex flex-col gap-1.5">
            <span className="text-xs font-medium">Couleur d'accent</span>
            <div className="flex gap-2">
              <input
                type="color"
                value={colors.accent_color || "#6366f1"}
                onChange={(e) => handleColorChange("accent_color", e.target.value)}
                className="h-10 w-12 cursor-pointer rounded border border-border"
              />
              <Input
                type="text"
                placeholder="#6366f1"
                value={colors.accent_color || ""}
                onChange={(e) => handleColorChange("accent_color", e.target.value)}
                className="text-xs"
              />
            </div>
          </Label>

          <Label className="flex flex-col gap-1.5">
            <span className="text-xs font-medium">Couleur du texte</span>
            <div className="flex gap-2">
              <input
                type="color"
                value={colors.text_color || "#1f2937"}
                onChange={(e) => handleColorChange("text_color", e.target.value)}
                className="h-10 w-12 cursor-pointer rounded border border-border"
              />
              <Input
                type="text"
                placeholder="#1f2937"
                value={colors.text_color || ""}
                onChange={(e) => handleColorChange("text_color", e.target.value)}
                className="text-xs"
              />
            </div>
          </Label>
        </div>

        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => void handleSaveColors()}
          disabled={colorsLoading}
          className="w-fit"
        >
          {colorsLoading ? "Enregistrement..." : "Enregistrer les couleurs"}
        </Button>
      </div>

      {/* Preview */}
      <div className="flex flex-col gap-2 rounded-lg border border-border bg-muted p-4">
        <span className="text-xs font-semibold text-muted-foreground">Aperçu</span>
        <div className="flex gap-3">
          <div
            className="h-12 w-12 rounded"
            style={{ backgroundColor: colors.primary_color || "#2f9e94" }}
            title="Couleur principale"
          />
          <div
            className="h-12 w-12 rounded"
            style={{ backgroundColor: colors.accent_color || "#6366f1" }}
            title="Couleur d'accent"
          />
          <div
            className="h-12 w-12 rounded"
            style={{ backgroundColor: colors.text_color || "#1f2937" }}
            title="Couleur du texte"
          />
          {logoUrl && (
            <div className="flex items-center rounded border border-border bg-background p-2">
              <img src={logoUrl} alt="Logo preview" className="h-10 w-10 object-contain" />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
