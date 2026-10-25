# Architecture — Devis Generator : VIBED

Monorepo dans le même dépôt que le site vanilla historique (laissé intact à la racine).

Décisions produit : [DECISIONS.md](DECISIONS.md).

## Structure

```
backend/                 Module Go (API + serveur HTTP)
  cmd/server/            Point d'entrée
  internal/config/       Variables d'environnement
  internal/db/           Pool pgx + migrations embarquées
  internal/httpapi/      Routes chi (/healthz, /api/v1, SPA)
  internal/legal/        Interface Provider/Store (sources officielles à brancher)
frontend/                React + TypeScript + Vite + Tailwind + shadcn/ui (FR)
docs/                    Documentation VIBED
compose.yaml             Postgres + app (dev)
Dockerfile.vibed         Image multi-stage (ne remplace pas Dockerfile nginx)
LICENSE.vibed            Apache-2.0 (code VIBED)
LICENSE                  GPL-3.0 (site historique, inchangé)
```

## API minimale

| Méthode | Chemin | Rôle |
|---------|--------|------|
| GET | `/healthz` | Liveness (200, sans Postgres) |
| GET | `/readyz` | Readiness (Ping Postgres) |
| GET | `/api/v1/healthz` | Idem liveness, préfixe versionné |
| GET | `/api/v1/readyz` | Idem readiness, préfixe versionné |
| GET | `/*` | Front statique (`frontend/dist`) + fallback SPA |

Au démarrage : retries vers Postgres, puis migrations golang-migrate (verrou consultatif PostgreSQL par défaut, sûr multi-réplicas).


## Schéma (socle)

- Types document : `quote`, `invoice`, `deposit_invoice`, `credit_note`
- TVA : `vat_regime` sur le document + `tax_rate_bps` / `tax_cents` par ligne
- Numérotation : table `document_number_sequences` (émetteur × type × année)
- Comptes optionnels : `users.password_hash` ; invité = `user_id` NULL

## Package `legal`

Contrat uniquement : `Provider` / `Store`, table `legal_variables`. Sources officielles à brancher plus tard.

## Legacy

`index.html`, `main.js`, `style.css`, `src/`, `Dockerfile`, `docker/nginx.conf` et le workflow `docker-release.yml` (tags `v*`) restent en place sous GPL-3.0.
