import { useState } from "react"

import { CompanyAutocomplete } from "@/components/company-autocomplete"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Segmented } from "@/components/ui/segmented"
import { Select } from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"
import { VATValidator } from "@/components/vat-validator"
import { COUNTRIES, applyCompanyHit } from "@/lib/party"
import type { Address, Client, IssuerProfile } from "@/lib/types"

export function AddressFields({ value, onChange }: { value: Address; onChange: (a: Address) => void }) {
  const set = (p: Partial<Address>) => onChange({ ...value, ...p })
  const known = COUNTRIES.some((c) => c.value === value.country)
  return (
    <>
      <Label>
        <span>Adresse</span>
        <Input placeholder="Numéro et voie" autoComplete="address-line1" value={value.line1} onChange={(e) => set({ line1: e.target.value })} />
      </Label>
      <Label>
        <span>
          Complément <span className="font-normal text-muted-foreground">(facultatif)</span>
        </span>
        <Input autoComplete="address-line2" value={value.line2} onChange={(e) => set({ line2: e.target.value })} />
      </Label>
      <div className="grid grid-cols-[120px_minmax(0,1fr)] gap-3">
        <Label>
          <span>Code postal</span>
          <Input placeholder="75000" autoComplete="postal-code" value={value.postal_code} onChange={(e) => set({ postal_code: e.target.value })} />
        </Label>
        <Label>
          <span>Ville</span>
          <Input autoComplete="address-level2" value={value.city} onChange={(e) => set({ city: e.target.value })} />
        </Label>
      </div>
      <Label>
        <span>Pays</span>
        <Select value={value.country || "FR"} onChange={(e) => set({ country: e.target.value })}>
          {COUNTRIES.map((c) => (
            <option key={c.value} value={c.value}>
              {c.label}
            </option>
          ))}
          {!known && value.country && <option value={value.country}>{value.country}</option>}
        </Select>
      </Label>
    </>
  )
}

const grid2 = "grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))]"

type ClientDraft = Pick<Client, "name" | "email" | "siren" | "siret" | "vat_number" | "address">

/** Champs d'un client : professionnel (recherche d'entreprise, SIREN, TVA) ou particulier. */
export function ClientFields<T extends ClientDraft>({ value, onChange }: { value: T; onChange: (v: T) => void }) {
  const [type, setType] = useState<"pro" | "part">(value.siren || value.vat_number || !value.name ? "pro" : "part")
  const set = (p: Partial<ClientDraft>) => onChange({ ...value, ...p })

  return (
    <>
      <Segmented
        label="Type de client"
        stretch
        value={type}
        onChange={setType}
        options={[
          { value: "pro", label: "Professionnel" },
          { value: "part", label: "Particulier" },
        ]}
      />
      {type === "pro" ? (
        <>
          <CompanyAutocomplete onPick={(hit) => onChange(applyCompanyHit(value, hit, "name"))} />
          <div className="flex items-center gap-3">
            <Separator className="flex-1" />
            <span className="text-xs whitespace-nowrap text-muted-foreground">ou saisissez les informations</span>
            <Separator className="flex-1" />
          </div>
          <Label>
            <span>Raison sociale</span>
            <Input required value={value.name} onChange={(e) => set({ name: e.target.value })} />
          </Label>
          <div className={grid2}>
            <Label>
              <span>SIREN</span>
              <Input inputMode="numeric" placeholder="9 chiffres" value={value.siren} onChange={(e) => set({ siren: e.target.value })} />
            </Label>
            <Label>
              <span>SIRET</span>
              <Input inputMode="numeric" placeholder="14 chiffres" value={value.siret} onChange={(e) => set({ siret: e.target.value })} />
            </Label>
          </div>
          <VATValidator value={value.vat_number} onChange={(vat_number) => set({ vat_number })} />
        </>
      ) : (
        <Label>
          <span>Nom et prénom</span>
          <Input required autoComplete="name" value={value.name} onChange={(e) => set({ name: e.target.value })} />
        </Label>
      )}
      <AddressFields value={value.address} onChange={(address) => set({ address })} />
      <Label>
        <span>E-mail de facturation</span>
        <Input type="email" placeholder="nom@exemple.fr" autoComplete="email" value={value.email} onChange={(e) => set({ email: e.target.value })} />
      </Label>
    </>
  )
}

function Legend({ children }: { children: string }) {
  return <legend className="mb-0.5 text-[11px] leading-4 font-semibold tracking-[0.06em] text-muted-foreground uppercase">{children}</legend>
}

/** Champs de l'émetteur (fiche Entreprise). */
export function IssuerFields({ value, onChange, withSearch = true }: { value: IssuerProfile; onChange: (v: IssuerProfile) => void; withSearch?: boolean }) {
  const set = (p: Partial<IssuerProfile>) => onChange({ ...value, ...p })
  return (
    <>
      <fieldset className="flex flex-col gap-3">
        <Legend>Identité</Legend>
        {withSearch && <CompanyAutocomplete label="Retrouver votre entreprise" onPick={(hit) => onChange(applyCompanyHit(value, hit, "legal_name"))} />}
        <Label>
          <span>Raison sociale</span>
          <Input required autoComplete="organization" value={value.legal_name} onChange={(e) => set({ legal_name: e.target.value })} />
        </Label>
        <Label>
          <span>
            Nom commercial <span className="font-normal text-muted-foreground">(facultatif)</span>
          </span>
          <Input value={value.trade_name} onChange={(e) => set({ trade_name: e.target.value })} />
        </Label>
        <div className={grid2}>
          <Label>
            <span>SIRET</span>
            <Input inputMode="numeric" placeholder="14 chiffres" value={value.siret} onChange={(e) => set({ siret: e.target.value })} />
          </Label>
          <Label>
            <span>Code APE</span>
            <Input placeholder="0000A" value={value.ape_code} onChange={(e) => set({ ape_code: e.target.value })} />
          </Label>
        </div>
        <div className={grid2}>
          <Label>
            <span>SIREN</span>
            <Input inputMode="numeric" placeholder="9 chiffres" value={value.siren} onChange={(e) => set({ siren: e.target.value })} />
          </Label>
          <Label>
            <span>
              RCS ou RM <span className="font-normal text-muted-foreground">(si immatriculé)</span>
            </span>
            <Input placeholder="RCS Paris 000 000 000" value={value.rcs} onChange={(e) => set({ rcs: e.target.value })} />
          </Label>
        </div>
        <VATValidator
          value={value.vat_number}
          onChange={(vat_number) => set({ vat_number })}
          placeholder={value.default_vat_regime === "franchise_293b" ? "Non applicable en franchise en base" : "FR 00 000000000"}
        />
      </fieldset>
      <fieldset className="flex flex-col gap-3">
        <Legend>Coordonnées</Legend>
        <AddressFields value={value.address} onChange={(address) => set({ address })} />
        <div className={grid2}>
          <Label>
            <span>E-mail</span>
            <Input type="email" placeholder="nom@exemple.fr" autoComplete="email" value={value.email} onChange={(e) => set({ email: e.target.value })} />
          </Label>
          <Label>
            <span>Téléphone</span>
            <Input type="tel" placeholder="06 00 00 00 00" autoComplete="tel" value={value.phone} onChange={(e) => set({ phone: e.target.value })} />
          </Label>
        </div>
        <Label>
          <span>IBAN</span>
          <Input placeholder="FR76 0000 0000 0000 0000 0000 000" value={value.iban ?? ""} onChange={(e) => set({ iban: e.target.value })} />
        </Label>
        <p className="-mt-1.5 text-xs text-muted-foreground">L’IBAN et l’adresse sont chiffrés sur nos serveurs.</p>
      </fieldset>
    </>
  )
}
