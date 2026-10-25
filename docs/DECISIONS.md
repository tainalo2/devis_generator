# Décisions produit — Devis Generator : VIBED

Décisions validées par le propriétaire. Ce fichier est la référence courte.

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

- Brouillon éditable ; **émis** (`issued`) = **immuable** (correction via avoir)
- Conversion devis → facture ; acompte / avoir liés au document d’origine
- PDF simple V1 (**pas** Factur-X — prévu V2)
- Valeurs légales **figées** dans le document à l’émission (`issuer_snapshot`, `legal_snapshot`)

### Numérotation

Continue **par type et par année**, sans trou, transactionnelle (`INSERT … ON CONFLICT … RETURNING`) :

| Type | Préfixe | Exemple |
|------|---------|---------|
| Devis | `D` | `D-2026-0001` |
| Facture | `F` | `F-2026-0001` |
| Facture d’acompte | `FA` | `FA-2026-0001` |
| Avoir | `A` | `A-2026-0001` |

## TVA (V1)

- Franchise art. **293 B** (mention obligatoire, TVA = 0)
- Ou TVA normale avec taux **par ligne** : 0 / 2,1 / 5,5 / 10 / 20 % (points de base)
- **Pas** d’autoliquidation ni exonérations UE/export pour l’instant
- Montants en **centimes** ; arrondis half-up documentés dans `internal/billing`

## Comptes

Optionnels (invité + email/mot de passe). En invité, le PDF est généré sans conservation serveur (`POST /api/v1/guest/pdf`).

## Chiffrement

AES-256-GCM (`ENCRYPTION_KEY`) pour IBAN / adresses clients et profil émetteur.

## Sources légales (`internal/legal`)

### Choix de schéma

- Table append-only **`legal_values`** (historique : source, `fetched_at`, statut `verified|stale|error|manual`)
- Table **`legal_variables`** conservée comme **projection** « dernière valeur utile » (lectures rapides / compatibilité)
- Ne pas muter l’historique : en cas d’échec upstream, on **append** `error` et on **ne touche pas** la projection → repli sur la dernière valeur vérifiée / manuelle

### Sources branchées (aucune clé API)

| Source | Usage | Limites |
|--------|--------|---------|
| **Recherche d’entreprises** (DINUM) | Autocomplétion SIREN/SIRET/nom | ≈ 7 req/s → intervalle ≥ 200 ms |
| **VIES** (Commission UE) | Validation n° TVA intracom | Bloque les re-checks du même n° → **cache 24 h** + rate limit |
| **TEDB** | Taux de TVA FR (SOAP) | Timeout / retry mesuré |
| **BCE SDMX** | Taux directeur (MRR) | Pour dériver les pénalités de retard |

### Emplacements (non branchés — compte requis)

- Insee Sirene
- Légifrance / PISTE

### Exclue

- **API Entreprise** (administrations uniquement)

### Valeurs manuelles (surveillance)

Seuils franchise en base et micro-entreprise, taux d’intérêt légal, indemnité forfaitaire **40 €**, mentions obligatoires, calendrier facturation électronique (**Factur-X** obligatoire pour l’émission par les micro-entrepreneurs à partir du **1er sept. 2027**).

### Pénalités de retard (dérivées)

Règle provisoire documentée (à **faire valider juridiquement**) :

`taux effectif = max(3 × taux d’intérêt légal, taux BCE + 10)`

Stockée sous `derived.late_penalty_rate`. Les mentions figées à l’émission utilisent cette valeur lorsqu’elle est vérifiée ; sinon repli `defaults-v1`.

### Rafraîchissement

Job périodique (défaut 24 h) protégé par **`pg_try_advisory_lock`** (sûr avec 2 pods). Timeouts HTTP configurables. Endpoints statut / recherche / VIES accessibles aussi en **invité** avec rate limit anti-abus.

## PDF

- Bibliothèque : **`github.com/phpdave11/gofpdf`** (MIT, compatible Apache-2.0)
- Mentions : `legal.Service.MentionsForIssue` (cache vérifié / manuel) sinon `defaults-v1`
- Factures : conditions de paiement, pénalités de retard, indemnité forfaitaire **40 €**

## Variables d’environnement (prod)

| Variable | Rôle |
|----------|------|
| `SESSION_SECRET` | Sessions / CSRF (≥ 32 caractères) |
| `ENCRYPTION_KEY` | AES-256-GCM, 32 octets base64 |
| `DATABASE_URL`, `APP_ENV`, `ADDR`, `SHUTDOWN_TIMEOUT` | Runtime |

Optionnelles légales (aucun secret) : `LEGAL_REFRESH_ENABLED`, `LEGAL_REFRESH_INTERVAL`, `LEGAL_HTTP_TIMEOUT`, `LEGAL_RECHERCHE_BASE_URL`, `LEGAL_VIES_URL`, `LEGAL_TEDB_URL`, `LEGAL_BCE_URL`.

## Suite

Landing / pages légales / refonte graphique ; SMTP reset MDP ; Factur-X V2 ; brancher Insee/PISTE si comptes fournis.
