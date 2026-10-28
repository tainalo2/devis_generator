# Guide pour les agents

Lire ce fichier avant de modifier le dépôt. Le sens pour un lecteur humain est dans [humain.md](humain.md). Le détail d’ingénierie est dans [technique.md](technique.md). Les décisions courtes sont dans [DECISIONS.md](DECISIONS.md).

## Objectif

Construire une plateforme gratuite pour signer les factures en France : facture Factur-X téléchargeable, signature électronique attribuable à l’émetteur, puis transmission.

Écrire les textes, les interfaces et les commits au présent de ce qui existe. La signature, le PDF/A-3 et l’immatriculation DGFiP sont des objectifs. Les nommer comme tels.

## État à respecter

- `facturx.Build` produit uniquement le XML CII EN 16931. Il refuse les devis. Il ne signe rien.
- `pdfgen` produit le PDF téléchargé. Ne pas l’appeler depuis `facturx`, et ne pas y embarquer le XML tant qu’il n’y a pas un PDF/A-3 avec XMP.
- Les phrases du site qui disent que Factur-X n’est pas inclus restent vraies tant que le fichier téléchargé n’est pas une Factur-X.
- Les montants stockés sont des centimes. Les totaux affichés ou exportés passent par `billing.Compute`.
- Un document `issued` est immuable. Une correction est un avoir.
- Interface et documentation produit en français.
- Branche de travail : `vibed`. Le site vanilla à la racine est GPL-3.0 et se laisse en place. Le code VIBED (`backend/`, `frontend/`, `docs/`) est Apache-2.0.

## Où agir

| Sujet | Endroit |
|-------|---------|
| XML Factur-X | `backend/internal/facturx/` |
| PDF | `backend/internal/pdfgen/` |
| Totaux | `backend/internal/billing/calc.go` |
| Mentions 293 B | `backend/internal/legal/defaults.go` |
| Modèle document, émetteur, client | `backend/internal/biz/models.go` |
| Aperçu écran, à garder aligné avec le PDF | `frontend/src/components/editor/preview.tsx` |
| Textes publics Factur-X | `frontend/src/pages/landing-page.tsx`, `frontend/src/pages/legal-sources-page.tsx` |
| Page de présentation | `docs/index.html` |

## Vérifier

```bash
cd backend && go test ./internal/facturx/ ./internal/pdfgen/
```

Les tests Factur-X ne font aucun appel réseau. Après un changement d’écran, d’éditeur ou de données affichées, parcourir le flux dans le navigateur.

## À ne pas faire

- Inventer un certificat, une signature, un agrément ou un envoi DGFiP.
- Copier un secret réel dans un commit. Les valeurs de `.env.example` sont des défauts de développement.
- Déplacer ou relicencier le site historique de la racine.
- Réécrire le PDF pour « faire Factur-X » sans le conteneur PDF/A-3 : le résultat serait un fichier qui en a le nom et pas la forme.
