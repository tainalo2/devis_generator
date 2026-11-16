# Modèle de données : Invitations et Collaborateurs (Multi-T2)

## Overview

Multi-T2 implémente un système de collaboration multi-utilisateurs permettant aux propriétaires de compte de partager l'accès à leurs documents et clients avec d'autres utilisateurs via des invitations par email.

Le système est fondé sur :
- **account_collaborators** : liaison durable compte ↔ utilisateur ↔ rôle
- **invitations** : invitations temporaires à usage unique par email

**Rôles décidés (V1)** : 2 rôles seulement
- `owner` : propriétaire du compte (accès complet, implicite à la création)
- `collaborator` : collaborateur invité (accès limité défini par Multi-T5)

Voir `docs/DECISIONS.md` § « Multi-utilisateurs par compte — rôles (V1) » pour la rationale.

## Architecture

### Table : account_collaborators

Lie les collaborateurs à un compte avec un rôle défini.

```sql
CREATE TABLE account_collaborators (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES users(id),
    user_id UUID NOT NULL REFERENCES users(id),
    role TEXT NOT NULL CHECK (role IN ('owner', 'collaborator')),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE(account_id, user_id)
);
```

**Sémantique** :
- `account_id` : propriétaire du compte (UUID du user initial)
- `user_id` : collaborateur (UUID d'un autre user)
- `role` : permission granted (`owner` ou `collaborator`)
- Chaque paire (account_id, user_id) ne peut exister qu'une fois

**Indexation** :
- `account_collaborators_account_id_idx` : requêtes par compte (lister collaborateurs)
- `account_collaborators_user_id_idx` : requêtes par user (lister ses comptes)
- `account_collaborators_role_idx` : filtrage par rôle (vérification propriétaire, audit)

### Table : invitations

Invitations temporaires, à usage unique, avec expiration.

```sql
CREATE TABLE invitations (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES users(id),
    email TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('collaborator')),
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    UNIQUE(account_id, email) WHERE expires_at > NOW()
);
```

**Sémantique** :
- `account_id` : compte partagé
- `email` : adresse email de l'invité (peut ne pas exister en users)
- `role` : permission si acceptée (`collaborator` seulement, jamais `owner`)
- `token` : identifiant aléatoire 32-byte hex (unhashed, court-vécu, email-verified)
- `expires_at` : expiration de l'invitation (par défaut +7j)
- `created_by` : auditage de qui a créé l'invitation
- Contrainte partielle sur (account_id, email) : une invocation active par email/compte
  → plusieurs invites au même email peuvent exister si les précédentes ont expiré

**Indexation** :
- `invitations_account_id_idx` : lister invitations en attente pour un compte
- `invitations_email_idx` : vérifier si un email est invité (sécurité/UX)
- `invitations_token_idx` : accepter invitation par token (clé de lien email)
- `invitations_expires_at_idx` : nettoyage des invitations expirées
- `invitations_created_by_idx` : auditage des invitations créées

## Rôles et Permissions

### owner
- Accès complet au compte
- Peut modifier la configuration sensible (clés API, Stripe, SMTP, coordonnées bancaires)
- Peut inviter/révoquer collaborateurs
- Peut supprimer le compte
- **Statut** : implicite, créé automatiquement quand l'utilisateur crée le compte
- Réapparaît explicitement en account_collaborators si Multi-T1 implémente la migration de compte

### collaborator
- Accès limité défini par Multi-T5 (rôle-based access control dans les endpoints)
- **Typiquement** : lecture/écriture documents et clients, sans accès configuration sensible
- Permissions exactes (documents/clients read vs write, accès analytics, etc.) à confirmer en Multi-T5

### Refus implicite (non-rôle)
- Utilisateur non-collaborateur : rejeté HTTP 403 / Unauthorized
- API key : rejeté sur endpoints sensibles (voir permissions.go)

## Flux : Invitation et Acceptation

### 1. Propriétaire envoie invitation

```
POST /api/v1/collaborators/invite
{
  "email": "alice@example.com"
}
```

Serveur (Multi-T3) :
1. Valide l'email (format, non-duplicate-active)
2. Génère token aléatoire 32-byte (hex)
3. Calcule expires_at = NOW() + 7 days
4. Insère en invitations (role = 'collaborator')
5. Envoie email : "Vous êtes invité : [lien]/invite/{token}"

### 2. Invité clique lien ou accepte via UI

```
POST /api/v1/collaborators/accept-invitation
{
  "token": "...",
  "password": "..." (si nouvel utilisateur)
}
```

Serveur (Multi-T3) :
1. Charge invitation par token
2. Vérifie expires_at > NOW()
3. Si email ∉ users :
   - Crée user (email, password, UUID)
   - Enregistre session
4. Sinon : charge user par email
5. Insère en account_collaborators (account_id, user_id, role='collaborator')
6. **Supprime** invitation (token unique, one-use)
7. Retourne session/token

### 3. Propriétaire révoque

```
DELETE /api/v1/collaborators/{collaborator_id}
```

Serveur (Multi-T3) :
1. Vérifie authenticated user = account owner
2. Refuse si last remaining owner
3. Supprime ligne account_collaborators
4. Révocation immédiate (pas de révocation graduelle)

## Intégration avec auth et permissions

### Middleware : requireOwner (httpapi/permissions.go)

Actuellement, **TODO** : vérifie que l'utilisateur est propriétaire du compte.
Pour Multi-T2 onwards, implémenter :

```go
// Check if user.ID is in account_collaborators for the account
// and has role 'owner'
query := `
  SELECT role FROM account_collaborators
  WHERE account_id = $1 AND user_id = $2 AND role = 'owner'
`
```

### Middleware : isolateSensitiveEndpoints

Endpoints sensibles (clés API, Stripe, SMTP, coordonnées bancaires) :
- Rejetés si accès via API key
- Rejetés si collaborateur (Multi-T5 implémente la vérification du rôle)

Multi-T2 crée la structure ; Multi-T5 implémente l'isolation.

### Endpoints collaborateurs (Multi-T3)

**Public (pré-auth)** :
- `POST /accept-invitation` — accepter invitation par token

**Auth (session)** :
- `GET /api/v1/collaborators` — lister collaborateurs du compte
- `POST /api/v1/collaborators/invite` — inviter
- `DELETE /api/v1/collaborators/{id}` — révoquer
- `GET /api/v1/collaborators/pending` — lister invitations en attente (owner)

## Sécurité et Auditabilité

### 1. Tokens d'invitation
- 32-byte aléatoire (hex string)
- Unique en base (UNIQUE constraint)
- Unhashed (court-vécu, email-verified, one-use)
- Expirent après 7 jours (configurable)
- Supprimés une fois acceptés

### 2. Isolation des données
- Chaque collaborateur ne voit que les documents/clients de son compte
- Requêtes doivent filtrer par account_id
- account_id inféré du propriétaire du document/client, vérifié contre session user

### 3. Auditabilité
- `created_by` en invitations → qui a envoyé l'invitation
- `created_at`, `updated_at` en account_collaborators
- Logs d'accès sensibles via les endpoints existants

### 4. Prévention de révocation en cascade
- Si collaborateur supprimé de account_collaborators, les documents qu'il a créés restent
- Pas de suppression en cascade de documents (FK sur user_id NULL allowed)
- Nettoyage opportuniste ou batch job si nécessaire

## Migration et Schéma

Fichier : `backend/internal/db/migrations/000025_collaborations_and_invitations.up.sql`

**Éléments créés** :
1. Table account_collaborators (3 indexes)
   - Rôles: `owner`, `collaborator`
2. Table invitations (5 indexes)
   - Rôles: `collaborator` seulement
3. Foreign keys avec CASCADE ON DELETE

**Down** : DROP TABLE IF EXISTS (sûr)

**Contraintes** :
- CHECK role ∈ ('owner', 'collaborator') pour account_collaborators
- CHECK role IN ('collaborator') pour invitations
- UNIQUE account_id + user_id
- Partial UNIQUE sur (account_id, email) WHERE expires_at > NOW()

## Prochaines étapes

1. **Multi-T1** (concurrent) : Authentification / sessions (en cours)
2. **Multi-T3** : Endpoints collaborateurs (invite, accept, revoke, list)
3. **Multi-T4** : Frontend UI (invitations, collaborateurs, gestion)
4. **Multi-T5** : Role-based access control (vérification rôle dans endpoints, isolation données)

## Pitfalls identifiés et mitigation

### Pitfall 1 : Partial UNIQUE ne fonctionne pas sur PostgreSQL < 15
**Mitigation** : Projet sur PostgreSQL 15+. Vérifier en CI.

### Pitfall 2 : Invitation expirée toujours acceptée
**Mitigation** : Vérifier `expires_at > NOW()` dans le handler accept, et rejeter HTTP 400 si expiré.

### Pitfall 3 : Token collision faible
**Mitigation** : Utiliser `crypto/rand` 32 bytes → 2^256 espace. Risque négligeable.

### Pitfall 4 : Suppression du propriétaire
**Mitigation** : Refuse de supprimer si dernier owner. Multi-T5 implémente la prévention.

### Pitfall 5 : Drift entre rôles en code et en base
**Mitigation** : CHECK constraint en SQL permet seulement ('owner', 'collaborator'). Reflétez en Go const au niveau du package (e.g., `const RoleOwner = "owner"`).

## Schéma ERD (texte)

```
users (exists)
  ├── id (PK)
  ├── email
  ├── ...

account_collaborators (NEW)
  ├── id (PK)
  ├── account_id (FK → users.id)
  ├── user_id (FK → users.id)
  ├── role (owner|collaborator)
  └── UNIQUE(account_id, user_id)

invitations (NEW)
  ├── id (PK)
  ├── account_id (FK → users.id)
  ├── email
  ├── role (collaborator)
  ├── token (UNIQUE)
  ├── expires_at
  └── created_by (FK → users.id)
```

## Exemple de données

```sql
-- User 1 owns account 1
INSERT INTO account_collaborators (account_id, user_id, role)
VALUES ('550e8400-e29b-41d4-a716-446655440000', '550e8400-e29b-41d4-a716-446655440000', 'owner');

-- Invite alice@example.com as collaborator
INSERT INTO invitations (account_id, email, role, token, expires_at, created_by)
VALUES (
  '550e8400-e29b-41d4-a716-446655440000',
  'alice@example.com',
  'collaborator',
  'a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1',
  NOW() + INTERVAL '7 days',
  '550e8400-e29b-41d4-a716-446655440000'
);

-- Alice accepts, creates user, and joins
INSERT INTO users (email, password_hash, display_name)
VALUES ('alice@example.com', '...', 'Alice');

INSERT INTO account_collaborators (account_id, user_id, role)
VALUES (
  '550e8400-e29b-41d4-a716-446655440000',
  '<alice-uuid>',
  'collaborator'
);

DELETE FROM invitations WHERE token = 'a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1';
```

## Lexique

- **Account** : compagnie/organisation représentée par un propriétaire (user)
- **Owner** : propriétaire du compte, créateur automatique
- **Collaborator** : utilisateur invité avec accès limité
- **Role** : permission grant ('owner' ou 'collaborator')
- **Invitation** : lien temporaire par email, accepté une seule fois, puis supprimé
- **Token** : clé cryptographique aléatoire pour l'invitation (32 bytes hex)
