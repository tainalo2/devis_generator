# Devis Generator — Vibed

Branche de développement de la nouvelle version (**Vibed**) : backend Go + PostgreSQL + front React, dans le même dépôt que le site vanilla historique (conservé à la racine, non déplacé).

Site officiel actuel : [devisgenerator.fr](https://www.devisgenerator.fr/) · licence GPL-3.0.

## Prérequis

- Go 1.22+
- Node.js 22+
- Docker / Docker Compose (pour lancer l'ensemble)

## Démarrage rapide (Docker)

```bash
cp .env.example .env   # optionnel pour Compose (les valeurs sont déjà dans compose.yaml)
docker compose up --build
```

Puis ouvrir [http://localhost:8080](http://localhost:8080).  
Santé API : [http://localhost:8080/api/v1/healthz](http://localhost:8080/api/v1/healthz).

## Développement local (sans image app)

1. Postgres :

```bash
docker compose up -d db
```

2. Backend (depuis la racine du dépôt) :

```bash
cp .env.example .env
cd backend && go run ./cmd/server
```

3. Frontend (autre terminal) :

```bash
cd frontend && npm ci && npm run dev
```

Vite sert le UI sur [http://localhost:5173](http://localhost:5173) et proxyfie `/api` et `/healthz` vers `:8080`.

## Commandes utiles (`make`)

| Cible | Action |
|-------|--------|
| `make frontend-install` | `npm ci` |
| `make frontend-dev` | Vite |
| `make frontend-build` | Build front |
| `make backend-run` | Serveur Go |
| `make backend-test` | `go test ./...` |
| `make build` | Build front + binaire Go |
| `make compose-up` | `docker compose up --build` |
| `make test` | Tests backend + lint front |

Vérifications manuelles attendues :

```bash
cd backend && go build ./... && go test ./...
cd frontend && npm run build
```

## Structure

Voir [docs/architecture.md](docs/architecture.md).

- `backend/` — API Go (chi, pgx, migrations, package `legal` stub)
- `frontend/` — React + TS + Vite + Tailwind + shadcn/ui
- `Dockerfile.vibed` — image de la nouvelle stack (le `Dockerfile` nginx legacy reste pour les tags `v*`)
- `.github/workflows/vibed-ci.yml` — lint / test / build (sans publication d'image)

## Suite prévue

- Authentification / comptes
- CRUD clients et documents (devis / factures)
- Sources officielles pour le package `legal` + cache
- Génération PDF et mentions légales
- Décision sur le devenir du site vanilla à la racine
