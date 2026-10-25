# Architecture Vibed (socle)

Monorepo dans le même dépôt que le site vanilla historique (laissé intact à la racine).

## Structure

```
backend/                 Module Go (API + serveur HTTP)
  cmd/server/            Point d'entrée
  internal/config/       Variables d'environnement
  internal/db/           Pool pgx + migrations embarquées
  internal/httpapi/      Routes chi (/healthz, /api/v1, SPA)
  internal/legal/        Interface Provider/Store (sources officielles à brancher)
frontend/                React + TypeScript + Vite + Tailwind + shadcn/ui
docs/                    Documentation Vibed
compose.yaml             Postgres + app (dev)
Dockerfile.vibed         Image multi-stage (ne remplace pas Dockerfile nginx)
```

## API minimale

| Méthode | Chemin | Rôle |
|---------|--------|------|
| GET | `/healthz` | Santé process + base |
| GET | `/api/v1/healthz` | Même payload, préfixe versionné |
| GET | `/*` | Front statique (`frontend/dist`) + fallback SPA |

## Package `legal`

Contrat uniquement : `Provider` (fetch source officielle), `Store` (cache PostgreSQL avec `source` + `fetched_at`), table `legal_variables`. Aucune source officielle n'est branchée dans ce socle.

## Legacy

`index.html`, `main.js`, `style.css`, `src/`, `Dockerfile`, `docker/nginx.conf` et le workflow `docker-release.yml` (tags `v*`) restent en place.
