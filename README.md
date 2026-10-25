# Devis Generator : VIBED

Branche de développement de **Devis Generator : VIBED** — backend Go + PostgreSQL + front React, dans le même dépôt que le site vanilla historique (conservé à la racine, non déplacé).

Site officiel actuel : [devisgenerator.fr](https://www.devisgenerator.fr/)

## Licence

- **VIBED** (code sous `backend/`, `frontend/`, `docs/`, `Dockerfile.vibed`, etc.) : [Apache-2.0](LICENSE.APACHE-2.0) (aussi `LICENSE.vibed`)
- **Site vanilla historique** (racine : `index.html`, `main.js`, …) : [GPL-3.0](LICENSE) — inchangée

Détail : [docs/LICENSING.md](docs/LICENSING.md).

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

- Liveness : [http://localhost:8080/healthz](http://localhost:8080/healthz)
- Readiness : [http://localhost:8080/readyz](http://localhost:8080/readyz)

Image d’essai GHCR : `ghcr.io/tainalo2/devisgenerator-vibed` (tags `vibed-v*` via `.github/workflows/vibed-release.yml`).

## Variables d’environnement

| Variable | Rôle | Défaut |
|----------|------|--------|
| `DATABASE_URL` | Connexion PostgreSQL (obligatoire) | — |
| `APP_ENV` | Environnement (`development` / `production`) | `development` |
| `ADDR` | Adresse d’écoute HTTP | `0.0.0.0:8080` |
| `SHUTDOWN_TIMEOUT` | Arrêt gracieux | `10s` |
| `SESSION_SECRET` | Sessions / CSRF (≥ 32 car. en prod) | défaut dev |
| `ENCRYPTION_KEY` | AES-256-GCM, 32 octets **base64** | défaut dev |

En **production**, l’appli refuse de démarrer si `SESSION_SECRET` / `ENCRYPTION_KEY` sont absents ou faibles.

Optionnelles : `STATIC_DIR`, `STARTUP_TIMEOUT`, `SESSION_TTL`, `COOKIE_SECURE`, `ARGON2_*`, `LEGAL_*`, et site public :

| Variable | Rôle |
|----------|------|
| `DONATE_URL` | Lien discret d’appel aux dons (aucun prestataire intégré) |
| `LEGAL_PUBLISHER_*` | Éditeur (mentions) — vide = « à compléter » |
| `LEGAL_HOST_*` | Hébergeur (mentions / confidentialité) |

Voir `.env.example`.

## Développement local (sans image app)

1. Postgres : `docker compose up -d db`
2. Backend : `cd backend && go run ./cmd/server`
3. Frontend : `cd frontend && npm ci && npm run dev`

Vite : [http://localhost:5173](http://localhost:5173) (proxy `/api`, `/healthz`, `/readyz`).

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

```bash
cd backend && go build ./... && go test ./...
cd frontend && npm run build
```

## Structure

Voir [docs/architecture.md](docs/architecture.md) et [docs/DECISIONS.md](docs/DECISIONS.md).

## Fonctionnalités VIBED (branche)

- Landing publique + mentions légales / confidentialité (champs éditeur à compléter)
- Compte optionnel + mode invité (PDF sans conservation)
- Tableau de bord, clients, documents (brouillon → émission), aperçu PDF
- TVA franchise 293 B ou multi-taux ; sources légales officielles
- Thème clair/sombre, shell sidebar desktop / navigation mobile

Suite : SMTP reset MDP, Factur-X V2, compléter identité éditeur/hébergeur et `DONATE_URL`.
