# Décisions produit — Devis Generator : VIBED

Décisions validées par le propriétaire. Ce fichier est la référence courte ; l’implémentation complète arrivera par itérations.

## Identité

- **Nom officiel** : `Devis Generator : VIBED`
- **Licence (code Vibed)** : Apache-2.0 — voir `LICENSE.vibed`
- **Licence (site vanilla historique à la racine)** : GPL-3.0 — fichier `LICENSE` inchangé
- **Langue d’interface** : français uniquement
- **UX** : desktop d’abord, mobile utilisable

## Public cible

Micro-entrepreneurs, autres indépendants et petites sociétés.

## Documents (V1)

Types : **devis**, **facture**, **facture d’acompte**, **avoir**.

Numérotation **continue par type et par année**, format exemple : `F-2026-0001`.

Préfixes prévus (à confirmer à l’implémentation du service de numérotation) :

| Type | Préfixe |
|------|---------|
| Devis | `D` |
| Facture | `F` |
| Facture d’acompte | `FA` |
| Avoir | `A` |

## TVA

- Régime **franchise en base (art. 293 B)** supporté
- Et **TVA complète** avec **taux multiples par ligne** de document
- Variables légales (taux, seuils…) via le package `legal` + cache PostgreSQL (sources officielles à brancher plus tard)

## Comptes et authentification

- Comptes **optionnels** : mode **invité** (sans compte) + compte **e-mail / mot de passe**
- Aucune page métier ne force la connexion pour l’instant
- Mots de passe : **argon2id** (paramètres configurables via `ARGON2_*`)
- Sessions : cookie `vibed_session` **HttpOnly** + **Secure** (prod) + **SameSite=Lax**, token aléatoire stocké **hashé** en PostgreSQL (révocable)
- CSRF : cookie `vibed_csrf` (lisible JS) + en-tête `X-CSRF-Token` (double submit) sur les mutations authentifiées
- Limitation de débit en mémoire **par processus** sur login / inscription / reset (avec 2 réplicas ≈ 2× la limite)
- Messages d’erreur **sans énumération** d’e-mails
- RGPD : données conservées tant que le compte existe ; **effacement complet** à la suppression (cascade) ; **export JSON** proposé avant suppression
- Réinitialisation de mot de passe : **stub API** prêt (`POST /api/v1/auth/password-reset/request`) — **aucun envoi d’e-mail** tant qu’aucun SMTP n’est choisi

Variables d’environnement à fournir à l’hébergeur (prod) :

| Variable | Rôle |
|----------|------|
| `SESSION_SECRET` | Secret sessions/CSRF (≥ 32 caractères, pas le défaut dev) |
| `ENCRYPTION_KEY` | Clé AES-256-GCM, **32 octets en base64** |

## Chiffrement applicatif

- Package `internal/crypto` : **AES-256-GCM**, clé versionnée (octet d’id dans le blob) pour rotation
- Colonnes `clients.iban_enc` / `clients.address_enc` prêtes ; le CRUD documents/clients les utilisera

## PDF

Génération **côté serveur** (pas uniquement dans le navigateur).

## Suite

CRUD clients/documents, branchement chiffrement sur le CRUD, SMTP pour reset MDP, sources `legal`, PDF.
