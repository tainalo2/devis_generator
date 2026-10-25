# Architecture — Devis Generator : VIBED

Monorepo dans le même dépôt que le site vanilla historique (laissé intact à la racine).

Décisions produit : [DECISIONS.md](DECISIONS.md).

## Structure

```
backend/
  cmd/server/
  internal/config/       Env (dont SESSION_SECRET, ENCRYPTION_KEY)
  internal/auth/         argon2id, tokens
  internal/authstore/    users / sessions / export / delete
  internal/crypto/       AES-256-GCM versionné
  internal/db/           pgx + migrations embarquées
  internal/httpapi/      chi : health, auth, SPA
  internal/legal/        stub sources officielles
frontend/                React + TS + Vite + Tailwind + shadcn (FR)
docs/
compose.yaml
Dockerfile.vibed
LICENSE.vibed            Apache-2.0
LICENSE                  GPL-3.0 (historique)
```

## API

| Méthode | Chemin | Rôle |
|---------|--------|------|
| GET | `/healthz` | Liveness |
| GET | `/readyz` | Readiness (Postgres) |
| POST | `/api/v1/auth/register` | Inscription |
| POST | `/api/v1/auth/login` | Connexion |
| POST | `/api/v1/auth/logout` | Déconnexion |
| GET | `/api/v1/auth/me` | Utilisateur courant |
| GET | `/api/v1/auth/export` | Export JSON (RGPD) |
| DELETE | `/api/v1/auth/me` | Suppression compte |
| POST | `/api/v1/auth/password-reset/request` | Stub reset (pas d’e-mail) |
| GET | `/*` | SPA |

Migrations au boot avec verrou advisory (multi-réplicas OK). Migration `000002` additive (sessions, reset tokens, colonnes chiffrées, CASCADE).

## Legacy

Site vanilla + `Dockerfile` nginx + `docker-release.yml` (`v*`, hors `vibed-v*`) inchangés sous GPL-3.0.
