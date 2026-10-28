# Documentation technique

Public : quelqu’un qui lance, relit ou fait évoluer le code. Le sens du produit est dans [humain.md](humain.md). Les agents lisent [ia.md](ia.md).

## Objectif

Plateforme gratuite pour signer les factures en France.

Ordre de construction :

1. **Facture structurée téléchargeable.** `internal/facturx` écrit le XML CII EN 16931. Le PDF de `internal/pdfgen` reste un PDF simple (gofpdf). L’étape suivante embarque `factur-x.xml` dans un PDF/A-3 avec les métadonnées XMP. Tant que ce conteneur n’existe pas, l’interface continue de dire que Factur-X n’est pas inclus.
2. **Signature électronique du fichier.** Signature du PDF Factur-X, attribuable à l’émetteur, avec une trace d’intégrité. Le niveau eIDAS et le fournisseur de certificats ne sont pas choisis. Aucune clé de signature n’est dans le dépôt.
3. **Transmission.** Soit un branchement vers une plateforme agréée déjà immatriculée (API AFNOR XP Z12-013, formats XP Z12-012, cas d’usage XP Z12-014), soit un dossier d’immatriculation DGFiP si cet opérateur envoie lui-même les factures et les données (e-reporting, annuaire). Le dépôt actuel ne contient ni l’API ni le dossier.

## Dépôt

Monorepo. La branche de travail VIBED est `vibed`.

| Zone | Licence | Rôle |
|------|---------|------|
| `backend/`, `frontend/`, `docs/`, `Dockerfile.vibed` | Apache-2.0 | Application |
| Racine `index.html`, `main.js`, `Dockerfile` | GPL-3.0 | Site vanilla historique, laissé en place |

Détail : [LICENSING.md](LICENSING.md). Décisions courtes : [DECISIONS.md](DECISIONS.md). Carte du code : [architecture.md](architecture.md).

## Lancer

```bash
cp .env.example .env
docker compose up --build
```

Sans Docker : Postgres, puis `cd backend && go run ./cmd/server`, puis `cd frontend && npm ci && npm run dev`. L’interface Vite est sur [http://localhost:5173](http://localhost:5173), l’API sur le port 8080. Les secrets de `.env.example` sont des valeurs de développement.

```bash
cd backend && go test ./...
cd frontend && npm run build
```

## Factur-X déjà en place

`facturx.Build(doc, issuer, client) ([]byte, error)` retourne le XML `CrossIndustryInvoice`.

- Guideline : `urn:cen.eu:en16931:2017#compliant#urn:factur-x.eu:1p0:en16931`.
- Types : facture `380`, acompte `386`, avoir `381`. Un devis est une erreur.
- Montants en euros à deux décimales, à partir des centimes. Ils doivent coïncider avec `billing.Compute`. Quantités en milli-unités. Unité absente du modèle : `C62`.
- SIREN schéma `0002`, SIRET schéma `0009`, TVA schéma `VA`. Pays libre normalisé, défaut `FR`.
- Franchise 293 B : catégorie `E`, motif `legal.DefaultMentions().Franchise293B`. Taux 0 hors franchise : `Z`. Taux normal : `S`, et le n° de TVA vendeur est exigé.
- IBAN émetteur, s’il est renseigné : moyen de paiement `30`.

Le paquet n’est pas appelé par `pdfgen` ni par HTTP. Tests sans réseau : `go test ./internal/facturx/`. Le rendu PDF se vérifie avec `go test ./internal/pdfgen/`. Écarts (adresse électronique, références de commande, numéro de facture d’origine sur l’avoir, CIUS français, Schematron, PDF/A-3) : [factur-x.md](factur-x.md).

## Règles de calcul et de document

- Tous les montants persistés sont des centimes entiers. L’arrondi est celui de `billing.Compute` (demi vers l’extérieur).
- Un document émis est immuable. La correction passe par un avoir.
- La numérotation est continue par type et par année (`F-2026-0001`, préfixes `D`, `F`, `FA`, `A`).
- L’IBAN et les adresses clients sont chiffrés. L’émetteur figé sur un document émis aussi.
- Le PDF suit la mise en page de `frontend/src/components/editor/preview.tsx`, avec les polices Go embarquées pour l’UTF-8.

## Présentation

`docs/index.html` est la page publique du dépôt (GitHub Pages, branche `vibed`, dossier `/docs`). Elle décrit l’objectif et l’état réel. Elle ne remplace pas [devisgenerator.fr](https://www.devisgenerator.fr/).
