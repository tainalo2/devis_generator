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

## Signature électronique (V1)

**Décision (architecte, epic Signature `t_dee0608a` / Sig-T1 `t_0c1c888c`) : signature électronique simple horodatée, mécanisme interne, pas d'intégration tierce obligatoire en V1.**

### Mécanisme retenu

Sur le lien stable `/d/{token}` : bouton **« Signer ce devis »** → au clic, capture et persistance de :

- `signed_at` — horodatage serveur (UTC, pas horodatage client, pour éviter la manipulation)
- `document_hash` — SHA-256 du contenu figé du document (`legal_snapshot`/`issuer_snapshot` au moment de la signature)
- `signer_ip` — IP de la requête
- `signer_user_agent` — user-agent du navigateur
- `signer_name_declared` — identité déclarée (nom saisi par le signataire, non vérifiée)

Pas de certificat qualifié, pas de tiers de confiance. Correspond à la **« signature électronique simple »** au sens eIDAS (règlement UE 910/2014, art. 3(10)) : recevable comme preuve devant un juge (art. 25), mais sans la présomption légale de fiabilité attachée à la signature qualifiée — en cas de litige, la charge de la preuve de l'intégrité/l'authenticité repose sur l'éditeur/l'émetteur, pas sur une présomption automatique.

### Option tierce écartée pour V1

DocuSign/Yousign (même en mode **BYO clé API**, sur le modèle `internal/paymentbyo` — clé fournie par l'utilisateur, aucun coût pour l'éditeur) est **écarté pour le premier livrable** :

- Le cas d'usage cible (devis TPE/micro-entreprise, accord commercial simple) n'exige pas de valeur probatoire renforcée dans l'immense majorité des cas.
- Contrainte « open source/gratuit, coût d'hébergement minimal » : la signature simple horodatée ne demande ni service tiers ni clé API, donc zéro dépendance et zéro coût même en mode BYO (qui demande tout de même une intégration/maintenance côté éditeur).
- Aucune demande utilisateur identifiée justifiant ce coût d'intégration — même principe de prudence que pour l'export comptable (« ne pas anticiper », cf. ci-dessus).

**Évolution ultérieure (hors V1)** : si un besoin confirmé de valeur probatoire plus forte apparaît (montants élevés, contentieux anticipé), ajouter une option tierce BYO clé API (Yousign en priorité, API REST simple) **en plus** de la signature simple — jamais en remplacement — sur le même modèle que `paymentbyo` (clé API saisie par l'utilisateur dans son compte, stockée chiffrée via le vault existant). Ne pas construire avant retour utilisateur concret.

### Cadrage pour les tickets d'implémentation

- **Sig-T2** (modèle de données) : table dédiée (ou colonnes sur `documents`) portant `signed_at`, `document_hash` (SHA-256 hex, pas de HMAC — c'est un hash d'intégrité, aucun secret requis côté signature elle-même puisque l'accès au lien est déjà protégé par `DOC_LINK_SECRET`), `signer_ip`, `signer_user_agent`, `signer_name_declared`. Un devis signé devient **immuable**, même règle que `issued` pour facture/acompte/avoir.
- **Sig-T3** (UI) : capture au clic sur `/d/{token}`, pas de case "je certifie" complexe — rester simple, conforme à une signature électronique *simple*. Email de confirmation à l'émetteur.
- **Sig-T4** (émetteur) : badge + preuve consultable/téléchargeable (horodatage, hash, IP) — pas de génération de certificat, juste l'affichage structuré des champs ci-dessus.
- **Sig-T5** (doc self-hosters) : expliciter clairement que c'est une signature **simple** eIDAS, pas qualifiée ; ce que le mécanisme prouve (horodatage + intégrité du contenu + traçabilité technique) et ne prouve pas (identité vérifiée du signataire) ; mentionner l'option tierce future sans promettre une valeur juridique non vérifiée.

## Suite

SMTP reset MDP ; Factur-X téléchargeable puis signature électronique ; brancher Insee/PISTE si comptes fournis ; renseigner identité éditeur/hébergeur et `DONATE_URL`.
