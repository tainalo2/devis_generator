# Décisions produit — Devis Generator : VIBED

Décisions validées par le propriétaire. Ce fichier est la référence courte.

## Objectif

**Plateforme gratuite pour signer les factures en France.**

Aujourd’hui, l’application produit des devis et des factures PDF, avec un compte optionnel. Le XML Factur-X (profil EN 16931) se prépare à côté du PDF. La suite est une Factur-X téléchargeable, puis la signature électronique du fichier, puis sa transmission — par une plateforme agréée déjà immatriculée, ou par une immatriculation DGFiP si cet outil assure lui-même l’envoi.

## Identité

- **Nom officiel** : `Devis Generator : VIBED`
- **Licence (code Vibed)** : Apache-2.0 — voir `LICENSE.APACHE-2.0` / `LICENSE.vibed` ([LICENSING.md](LICENSING.md))
- **Licence (site vanilla historique à la racine)** : GPL-3.0 — fichier `LICENSE` inchangé
- **Langue d’interface** : français uniquement
- **UX** : desktop d’abord, mobile pleinement utilisable (pas mobile-first)
- **Site public** : landing + mentions légales + confidentialité — **pas** de pages Histoire / Road-map
- **Modèle économique** : gratuit, financé par dons (`DONATE_URL` optionnelle, aucun prestataire de paiement intégré)

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
| **TEDB** | Taux de TVA FR (SOAP) | Envelope = `IVatRetrievalService` + enfants `:types` (le targetNamespace WSDL `VatRetrievalService` seul échoue en XSD). Timeout / retry. Repli : dernière `verified` / saisie `manual` visible dans `/api/v1/legal/status` ; les taux FR de référence restent 20 / 10 / 5,5 / 2,1 % |
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

## Pages légales

Rédigées d’après le fonctionnement réel (données, chiffrement, conservation tant que le compte existe, export/suppression, cookies strictement nécessaires). Éditeur et hébergeur via variables d’environnement — **à compléter** et **à faire valider juridiquement**.

## Export comptable (V1)

**Décision (architecte, epic Export comptable `t_f4964981`/`t_1bed5d9d`) : CSV structuré simple, pas de FEC ni connecteur dédié en V1.**

Options écartées pour le premier livrable :
- **FEC** (Fichier des Écritures Comptables) : format réglementaire FR mais pensé pour une comptabilité en partie double (comptes, journaux, lettrage) — surdimensionné pour un simple export factures/paiements, coûteux à maintenir correctement (risque d'un FEC mal formé = pire qu'aucun FEC).
- **Connecteur direct** (Pennylane ou équivalent) : séduisant mais verrouille sur un outil, demande une intégration/API-maintenance par fournisseur, et aucun client n'a encore exprimé ce besoin précis.
- **Pas d'obligation légale** : les TPE/PME cibles n'ont pas d'obligation de facturation électronique chaînée (Factur-X obligatoire en émission pour micro-entrepreneurs seulement à partir du 1ᵉʳ sept. 2027, cf. ci-dessus) ; un export lisible/importable suffit pour alimenter leur compta ou leur expert-comptable.
- Contrainte d'hébergement minimal : pas de nouvelle dépendance lourde, pas de job de synchronisation externe à maintenir.

**Retenu : CSV UTF-8, un fichier = une période/un filtre de statut, encodage avec BOM pour compatibilité Excel FR.**

Séparateur `;` (convention FR/Excel), décimales avec `.` (montants en euros, 2 décimales) pour rester import-friendly aussi bien en logiciel compta FR qu'en Excel/Google Sheets.

### Schéma des colonnes

| Colonne | Exemple | Notes |
|---|---|---|
| `numero_document` | `F-2026-0001` | Numéro métier (voir Numérotation ci-dessus) |
| `type_document` | `facture` | `devis` \| `facture` \| `facture_acompte` \| `avoir` — export limité par défaut aux documents **émis** (`issued`) |
| `date_emission` | `2026-01-15` | ISO 8601 |
| `client_nom` | `Dupont SARL` | Raison sociale / nom client au moment de l'émission (`legal_snapshot`) |
| `client_siret` | `12345678900012` | Vide si non renseigné |
| `montant_ht` | `1000.00` | Euros, 2 décimales |
| `taux_tva` | `20.0` | Point de base ; `0` si franchise art. 293 B |
| `montant_tva` | `200.00` | Euros |
| `montant_ttc` | `1200.00` | Euros |
| `statut_paiement` | `paye` | `en_attente` \| `paye` \| `en_retard` \| `annule` |
| `date_paiement` | `2026-01-20` | Vide si non payé |
| `mode_paiement` | `virement` | Si connu (Stripe, virement, chèque...) ; vide sinon |
| `document_origine` | `D-2026-0003` | Pour acomptes/avoirs : numéro du document source ; vide sinon |

### Filtrage par défaut

Export sur une plage de dates (optionnelle), filtrable par statut de paiement et type de document. Inclut factures **et** acomptes payés par défaut (cf. Export-T2).

### Évolution ultérieure (hors V1)

Si la demande client existe : ajouter un export **FEC** en option (bouton distinct, pas en remplacement du CSV) et/ou un connecteur Pennylane, mais seulement après retour utilisateur confirmant le besoin — ne pas anticiper.

## Suite

SMTP reset MDP ; Factur-X téléchargeable puis signature électronique ; brancher Insee/PISTE si comptes fournis ; renseigner identité éditeur/hébergeur et `DONATE_URL`.
