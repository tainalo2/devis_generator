# Devis Generator : VIBED

Branche de développement de **Devis Generator : VIBED** — backend Go + PostgreSQL + front React, dans le même dépôt que le site vanilla historique (conservé à la racine, non déplacé).

Site officiel actuel : [devisgenerator.fr](https://www.devisgenerator.fr/)

## Licence

- **VIBED** (code sous `backend/`, `frontend/`, `docs/`, `Dockerfile.vibed`, etc.) : [Apache-2.0](LICENSE.vibed)
- **Site vanilla historique** (racine : `index.html`, `main.js`, …) : [GPL-3.0](LICENSE) — inchangée

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

- Liveness : [http://localhost:8080/healthz](http://localhost:8080/healthz) (200 sans dépendre de Postgres)
- Readiness : [http://localhost:8080/readyz](http://localhost:8080/readyz) (vérifie la base)

Image d’essai GHCR : `ghcr.io/tainalo2/devisgenerator-vibed` (tags `vibed-v*` via `.github/workflows/vibed-release.yml`).

## Variables d’environnement

| Variable | Rôle | Défaut |
|----------|------|--------|
| `DATABASE_URL` | Connexion PostgreSQL (obligatoire) | — |
| `APP_ENV` | Environnement (`development` / `production`) | `development` |
| `ADDR` | Adresse d’écoute HTTP (ex. `0.0.0.0:8080`) | `0.0.0.0:8080` |
| `SHUTDOWN_TIMEOUT` | Délai max d’arrêt gracieux | `10s` |
| `SESSION_SECRET` | Secret sessions / CSRF (≥ 32 car. en prod) | défaut dev (avertissement) |
| `ENCRYPTION_KEY` | Clé AES-256-GCM, 32 octets **base64** | défaut dev (avertissement) |

En **production**, l’appli refuse de démarrer si `SESSION_SECRET` / `ENCRYPTION_KEY` sont absents ou faibles.  
Optionnelles : `STATIC_DIR`, `STARTUP_TIMEOUT`, `SESSION_TTL`, `COOKIE_SECURE`, `ARGON2_*`. Voir `.env.example`.


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

Voir [docs/architecture.md](docs/architecture.md) et les décisions produit dans [docs/DECISIONS.md](docs/DECISIONS.md).

- `backend/` — API Go (chi, pgx, migrations, package `legal` stub)
- `frontend/` — React + TS + Vite + Tailwind + shadcn/ui (français uniquement)
- `Dockerfile.vibed` — image de la nouvelle stack (le `Dockerfile` nginx legacy reste pour les tags `v*`)
- `.github/workflows/vibed-ci.yml` — lint / test / build (sans publication d'image)

## Fonctionnalités VIBED (branche)

- Compte optionnel + mode invité (PDF sans conservation)
- Clients / documents (brouillon → émission numérotée)
- TVA franchise 293 B ou taux par ligne ; totaux en centimes
- PDF serveur (gofpdf) : compte ou invité

Voir [docs/DECISIONS.md](docs/DECISIONS.md). Suite : landing/pages légales, SMTP, sources `legal` officielles, Factur-X.

