# Guide de configuration des trois modes de paiement

Documentation complète pour déployer et gérer les trois modes de paiement : RIB, BYO (Bring Your Own) et Stripe Connect.

## Table des matières

1. [Matrice des modules](#1-matrice-des-modules)
2. [Flags de configuration](#2-flags-de-configuration)
3. [Gestion du trousseau de secrets](#3-gestion-du-trousseau-de-secrets)
4. [Mode RIB](#4-mode-rib)
5. [Mode BYO](#5-mode-byo)
6. [Mode Stripe Connect](#6-mode-stripe-connect)
7. [Modèle de statut unifié](#7-modèle-de-statut-unifié)
8. [Migration et rollback](#8-migration-et-rollback)

---

## 1. Matrice des modules

| Module | Statut par défaut | Coût | Dépendances externes | Visible par |
|--------|-------------------|------|----------------------|-------------|
| **RIB** | **Actif** | Gratuit | Aucune | Propriétaire/client (avec lien sécurisé) |
| **BYO** | Inactif | Gratuit | Stripe, Mollie ou PayPal (au choix) | Propriétaire uniquement |
| **Stripe Connect** | Inactif | Commissions Stripe | Compte Stripe Connect | Propriétaire + clients finaux |

### Comportements par défaut

- **RIB actif par défaut** : aucune configuration requise ; IBAN du propriétaire stocké chiffré en base de données.
- **BYO et Stripe Connect inactifs par défaut** : requièrent une activation explicite et une configuration du provider.

---

## 2. Flags de configuration

Les trois modes sont contrôlés par des flags booléens indépendants, définis via des variables d'environnement strictes au démarrage du serveur.

### Variables d'environnement

| Variable | Type | Défaut | Description |
|----------|------|--------|-------------|
| `PAYMENT_RIB_ENABLED` | booléen | `true` | Active le mode RIB (virement bancaire) |
| `PAYMENT_BYO_ENABLED` | booléen | `false` | Active le mode BYO (provider Stripe/Mollie/PayPal) |
| `PAYMENT_STRIPE_CONNECT_ENABLED` | booléen | `false` | Active Stripe Connect (plateforme de paiement) |

### Validation stricte

Les flags acceptent les valeurs suivantes (insensible à la casse) :
- **Vrai** : `1`, `true`, `yes`, `on`
- **Faux** : `0`, `false`, `no`, `off`
- **Par défaut** : si la variable est absente ou vide

Toute valeur invalide est rejetée au démarrage avec un message d'erreur.

### Exemple de configuration `.env`

```bash
# Développement : RIB actif, BYO et Connect inactifs
PAYMENT_RIB_ENABLED=true
PAYMENT_BYO_ENABLED=false
PAYMENT_STRIPE_CONNECT_ENABLED=false

# Production : tous les modes actifs
PAYMENT_RIB_ENABLED=true
PAYMENT_BYO_ENABLED=true
PAYMENT_STRIPE_CONNECT_ENABLED=true

# Variantes : test sans Stripe Connect
PAYMENT_RIB_ENABLED=true
PAYMENT_BYO_ENABLED=true
PAYMENT_STRIPE_CONNECT_ENABLED=false
```

### Les 8 combinaisons possibles

| RIB | BYO | Connect | Comportement |
|-----|-----|---------|--------------|
| ✓ | ✗ | ✗ | RIB uniquement (défaut) |
| ✓ | ✓ | ✗ | RIB + BYO (pas de marketplace) |
| ✓ | ✗ | ✓ | RIB + Stripe Connect |
| ✓ | ✓ | ✓ | Tous les modes actifs |
| ✗ | ✓ | ✗ | BYO uniquement (déploiement custom) |
| ✗ | ✓ | ✓ | BYO + Stripe Connect |
| ✗ | ✗ | ✓ | Stripe Connect uniquement (marketplace pure) |
| ✗ | ✗ | ✗ | **Aucun mode** → erreur `payment_mode_disabled` sur toute requête de paiement |

### Comportement lorsqu'un module est désactivé

Lorsqu'un module est désactivé (flag à `false`) :

1. **API** : toute requête dirigée vers ce mode retourne :
   ```json
   {
     "error": "payment_mode_disabled"
   }
   ```
   avec le code HTTP `404`.

2. **UI** : les boutons/formulaires pour ce mode sont masqués côté frontend (interroger `/api/v1/payments/capabilities` pour la liste active).

3. **Webhooks** : les webhooks du provider inactif sont ignorés (optionnel : rejeté avec 404).

4. **Base de données** : les enregistrements existants pour ce mode restent présents mais inaccessibles.

---

## 3. Gestion du trousseau de secrets

Le trousseau de secrets (`PAYMENT_SECRETS_KEYS`) protège tous les identifiants sensibles : IBANs, tokens OAuth, clés API, secrets webhook. Le chiffrement est AES-256-GCM avec versioning de clés.

### Génération du trousseau

**Générer une nouvelle clé AES-256 (32 octets) en base64 :**

```bash
# Méthode 1 : OpenSSL
openssl rand -base64 32

# Méthode 2 : Python
python3 -c "import secrets; import base64; print(base64.b64encode(secrets.token_bytes(32)).decode())"

# Exemple de sortie
X7vL9mK2pQr3tJ8nW5xZc4bD6eF1aG0hS2iP9kL4mN=
```

### Format du trousseau

Le trousseau est un dictionnaire JSON de **clé ID → clé brute (base64)**, stocké hors du dépôt.

**Structure (pas dans le repo) :**
```json
{
  "2024-11-01": "X7vL9mK2pQr3tJ8nW5xZc4bD6eF1aG0hS2iP9kL4mN=",
  "2024-10-01": "A1bC2dE3fG4hI5jK6lM7nO8pQ9rS0tU1vW2xY3zA4b=",
  "2024-09-01": "R5sT6uV7wX8yZ9aB0cD1eF2gH3iJ4kL5mN6oP7qR8sT="
}
```

**Clé active** : la clé ID utilisée pour tous les chiffrements nouveaux. Lors de la rotation, une nouvelle clé devient active, mais les anciennes clés restent disponibles pour le déchiffrement rétroactif.

### Stockage et permissions

- **Emplacement** : fichier ou secret manager (AWS Secrets Manager, HashiCorp Vault, etc.) hors du dépôt Git.
- **Permissions** : lecture seule pour le service de paiement en production.
- **Sauvegarde** : inclure le trousseau dans les sauvegardes chiffrées ; éviter les dépôts Git publics.

### Procédure de rotation sans perte

La rotation de clés est une opération non-disruptive : les anciens secrets restent déchiffrables.

**Étapes :**

1. **Générer une nouvelle clé** :
   ```bash
   NEW_KEY=$(openssl rand -base64 32)
   echo "Nouvelle clé: $NEW_KEY"
   ```

2. **Ajouter la nouvelle clé au trousseau** avec un nouvel ID (ex : date du jour) :
   ```json
   {
     "2024-11-13": "Y8pL2mK9qW3rT5sU7vX0yZ1aB4cD6eF9gH2iJ5kL=",
     "2024-11-01": "X7vL9mK2pQr3tJ8nW5xZc4bD6eF1aG0hS2iP9kL4mN=",
     ...
   }
   ```

3. **Définir la nouvelle clé comme active** (ID dans `PAYMENT_SECRETS_ACTIVE_KEY_ID`) :
   ```bash
   export PAYMENT_SECRETS_ACTIVE_KEY_ID=2024-11-13
   export PAYMENT_SECRETS_KEYS='{"2024-11-13": "Y8pL2mK9qW3rT5sU7vX0yZ1aB4cD6eF9gH2iJ5kL=", ...}'
   ```

4. **Redémarrer le service** ; tous les nouveaux secrets utilisent la clé `2024-11-13`.

5. **Conserver les anciennes clés** au moins 30 jours pour les opérations de rejeu (relecture des webhooks).

**Optionnel : regénération complète des secrets**

Si toutes les anciennes clés doivent être révoquées :
1. Exporter tous les secrets actuels (déchiffrés) dans une sauvegarde temporaire.
2. Générer un trousseau neuf avec une seule clé.
3. Re-chiffrer tous les secrets avec la nouvelle clé.
4. Supprimer la sauvegarde temporaire en toute sécurité.

### Récupération et test

**Vérifier que le trousseau est valide** :
```bash
# Pas d'API publique simple ; le test se fait à l'exécution :
# Si un secret ne peut pas être déchiffré, les logs incluront :
# "payment secret decryption failed" ou "invalid payment secret ciphertext"
```

### Symptômes d'une clé absente

1. **Démarrage du serveur** : erreur de validation au chargement de la configuration.
2. **Opération de paiement** : si une clé vieille est demandée mais absente du trousseau :
   - Log : `"decryption failed: key ... not found in keyring"`
   - API : réponse `500 Internal Server Error`
   - Webhook : rejeté avec `403 Forbidden` (clé de signature non trouvée).

3. **Audit** : vérifier les logs pour identifier quel secret n'était pas déchiffrable.

**Remédiation** : ajouter la clé manquante au trousseau et redémarrer.

---

## 4. Mode RIB

Le mode RIB permet au propriétaire de recevoir des virements bancaires directs. L'IBAN est stocké chiffré et visible uniquement sur demande via un lien sécurisé à usage unique.

### Configuration utilisateur

L'IBAN est défini dans le profil du propriétaire (page "Paramètres" ou API).

**Endpoint API** (à créer) :
```
PUT /api/v1/issuer/payment/iban
Content-Type: application/json

{
  "iban": "FR1420041010050500013M02606"
}
```

**Réponse en cas de succès** :
```json
{
  "masked_iban": "FR14 2004 ... **2606",
  "updated_at": "2024-11-13T10:30:00Z"
}
```

**Validation** :
- Longueur : 15-34 caractères.
- Format IBAN : vérifié par la somme de contrôle ISO 13616.
- Aucun espace ni tiret dans la saisie (ou trimmed avant validation).

### Lien stable et sécurisé

Chaque document avec le mode RIB actif expose un lien `/d/{secure_token}` :

**Format du lien** :
```
https://devis.exemple.fr/d/8f7e6d5c4b3a2f1e9d8c7b6a5f4e3d2c
```

**Propriétés du token** :
- UUID chiffré stocké en base de données.
- Valide jusqu'à l'expiration du document (ou jamais, si pas de date d'expiration configurée).
- Non reproductible : chaque document a son propre token.
- Non énumérable : pas de lien entre tokens consécutifs.

**Contenu de la page** :
```html
<!-- Titre et détails du document -->
<h1>Facture n° 2024-001</h1>
<p>Montant : 1 500,00 EUR</p>
<p>Statut : En attente de paiement</p>

<!-- Section RIB visible seulement via ce lien -->
<section class="payment-rib">
  <h2>Coordonnées bancaires</h2>
  <p>
    <strong>IBAN</strong> : FR14 2004 1010 0505 0001 3M0 2606
  </p>
  <p class="warning">
    ⚠ Cet IBAN n'apparaît jamais dans les PDF, exports ou emails.
    Conservez ce lien privé.
  </p>
</section>
```

### Marquage manuel du paiement

Deux approches pour marquer comme payé :

**1. Webhook bancaire (futur)** : si une intégration bancaire est disponible, le paiement peut être marqué automatiquement via un webhook qui détecte le virement entrant.

**2. Marquage manuel (immédiat)** : le propriétaire marque manuellement un paiement comme reçu.

**Endpoint API** (admin uniquement) :
```
PATCH /api/v1/documents/{doc_id}/payment-status
Authorization: Bearer <jwt_admin>
Content-Type: application/json

{
  "new_status": "paid",
  "reason": "Virement reçu le 2024-11-13"
}
```

**Réponse** :
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "payment_status": "paid",
  "marked_at": "2024-11-13T10:30:00Z",
  "marked_by": "email@proprietaire.fr"
}
```

### Avertissement explicite

**Important** : L'IBAN n'apparaît jamais dans :
- **PDF** : bien que le statut "payé" figure, pas d'IBAN en PDF.
- **Exports CSV/Excel** : les colonnes de paiement ne contiennent pas l'IBAN.
- **Emails** : les notifications de paiement ne contiennent pas l'IBAN.
- **Payloads d'API** (sauf sous `/d/{token}`) : l'IBAN n'est retourné que via le lien sécurisé.

**Affichage côté frontend** :
```javascript
// Sur /d/{secure_token}
const iban = response.data.payment_details.iban;
console.log("IBAN visible uniquement ici :", iban);

// Sur /api/v1/documents/{id}
// L'IBAN n'est JAMAIS retourné, même authentifié propriétaire.
// Redirection vers /d/{token} requise.
```

---

## 5. Mode BYO (Bring Your Own)

Le mode BYO permet au propriétaire de configurer son propre provider de paiement (Stripe, Mollie ou PayPal) et de recevoir les fonds directement sur son compte.

### Providers supportés

| Provider | Statut | Sandbox | Webhook | Permissions minimales |
|----------|--------|---------|---------|----------------------|
| **Stripe** | Stable | Oui (sk_test_, pk_test_) | OAuth 2.0 + signature HMAC | read:charges, read:customers |
| **Mollie** | Stable | Oui (test_{key}) | Signature HMAC-SHA256 | payments:read, payments:write |
| **PayPal** | Stable | Oui (stage URLs) | Webhook subscription + signature | PPCP scope |

### Configuration Stripe

**1. Créer une clé API Stripe test** :
- Aller sur [https://dashboard.stripe.com/apikeys](https://dashboard.stripe.com/apikeys)
- Copier la clé secrète de test (`sk_test_...`)
- Copier la clé publique de test (`pk_test_...`)

**2. Enregistrer les endpoint webhooks** :
- Dashboard Stripe → Webhooks → Ajouter endpoint
- URL : `https://votre-domaine.fr/api/v1/payments/webhooks/stripe`
- Événements à suivre : `charge.updated`, `charge.succeeded`, `charge.failed`, `charge.refunded`
- Clé de signature : générée par Stripe (whsec_...)

**3. Configuration application** :
```bash
export PAYMENT_BYO_ENABLED=true
export STRIPE_SECRET_KEY=sk_test_...
export STRIPE_PUBLISHABLE_KEY=pk_test_...
export STRIPE_WEBHOOK_SECRET=whsec_...
```

**4. Tester** (mode sandbox) :
```bash
curl -X POST https://localhost:8080/api/v1/payments/byo/validate \
  -H "Authorization: Bearer <jwt>" \
  -H "Content-Type: application/json" \
  -d '{"provider": "stripe"}'
```

**Réponse** :
```json
{
  "provider": "stripe",
  "valid": true,
  "account": {
    "id": "acct_1234567890",
    "email": "proprietaire@exemple.fr"
  }
}
```

### Configuration Mollie

**1. Créer un compte Mollie** :
- [https://www.mollie.com/dashboard](https://www.mollie.com/dashboard)
- Copier la clé API test : `test_...`

**2. Enregistrer le webhook** :
- Paramètres → Webhooks → Ajouter webhook
- URL : `https://votre-domaine.fr/api/v1/payments/webhooks/mollie`
- Événements : `payment.created`, `payment.updated`

**3. Configuration** :
```bash
export PAYMENT_BYO_ENABLED=true
export MOLLIE_API_KEY=test_...
export MOLLIE_WEBHOOK_URL=https://votre-domaine.fr/api/v1/payments/webhooks/mollie
```

### Configuration PayPal

**1. Créer une application PayPal** :
- [https://developer.paypal.com/dashboard/applications](https://developer.paypal.com/dashboard/applications)
- Business Sandbox account
- Créer une app REST API
- Copier : Client ID, Secret

**2. Enregistrer le webhook** :
- App Settings → Webhooks
- URL : `https://votre-domaine.fr/api/v1/payments/webhooks/paypal`
- Event types : `CHECKOUT.ORDER.COMPLETED`, `CHECKOUT.ORDER.PAYMENT-CAPTURE.FAILED`

**3. Configuration** :
```bash
export PAYMENT_BYO_ENABLED=true
export PAYPAL_CLIENT_ID=...
export PAYPAL_CLIENT_SECRET=...
export PAYPAL_WEBHOOK_URL=https://votre-domaine.fr/api/v1/payments/webhooks/paypal
```

### Webhook endpoints

| Provider | URL | Méthode |
|----------|-----|---------|
| Stripe | `POST /api/v1/payments/webhooks/stripe` | Signature HMAC-SHA256 |
| Mollie | `POST /api/v1/payments/webhooks/mollie` | Signature HMAC-SHA256 |
| PayPal | `POST /api/v1/payments/webhooks/paypal` | Signature JWT |

**Vérification de signature** : chaque webhook est authentifié avant traitement. Les webhooks non signés ou mal signés sont rejetés avec `403 Forbidden`.

### Test vs. Live

**Avant déploiement en production** :

| Étape | Clés | URL | Mode |
|-------|------|-----|------|
| Développement local | `sk_test_`, `test_`, Sandbox | localhost/api/v1/payments | Test |
| Staging | `sk_test_`, `test_`, Sandbox | staging.exemple.fr | Test |
| Production | `sk_live_`, `live_`, Production | www.exemple.fr | Live |

**Bascule en production** :
```bash
# Avant
STRIPE_SECRET_KEY=sk_test_51...

# Après
STRIPE_SECRET_KEY=sk_live_51...
```

**Important** : les webhooks doivent être ré-enregistrés pour les URLs de production.

### Permissions minimales

Chaque provider nécessite des permissions strictes (principe du moindre privilège) :

**Stripe** :
- `read:charges` : lire l'état des paiements.
- `read:customers` : lire les données client (optionnel).
- **Refuser** : `write:customers`, `delete:*`, tout accès administrateur.

**Mollie** :
- `payments:read` : lire l'état des paiements.
- `payments:write` : créer et mettre à jour les paiements.
- **Refuser** : `customers:write`, `refunds:write` (sauf si remboursement implémenté).

**PayPal** :
- `PAYPAL_SCOPE` = `openid profile email` pour OAuth.
- Webhooks en lecture seule : pas de modification d'ordre via la clé webhook.

### Rotation et révocation de clés

**Rotation programmée (tous les 90 jours)** :

1. Générer une nouvelle clé chez le provider.
2. Ajouter la nouvelle clé dans le coffre-fort de l'application.
3. Mettre à jour la variable d'environnement (ex : `STRIPE_SECRET_KEY`).
4. Redémarrer l'application.
5. Révoquer l'ancienne clé chez le provider après 24h de transition.

**Révocation d'urgence** :
```bash
# Si une clé est compromise
STRIPE_SECRET_KEY=sk_live_NEW_KEY  # Remplacer immédiatement
# Redémarrer et refuser toute requête avec l'ancienne clé
# Aller sur Stripe → Account Settings → API Keys → Revoke
```

### Dépannage

| Symptôme | Cause probable | Solution |
|----------|----------------|----------|
| Webhook rejeté (403) | Signature invalide | Vérifier clé webhook et format du payload |
| Paiement non marqué en base | Webhook non reçu | Vérifier URL webhook, logs provider, redémarrer webhook |
| Erreur "Provider not configured" | Clé vide ou invalide | Vérifier `STRIPE_SECRET_KEY` etc. |
| Paiement refusé en production | Mode test vs. live | Vérifier que clés live sont utilisées (`sk_live_...`) |

---

## 6. Mode Stripe Connect

Stripe Connect transforme l'application en plateforme de paiement, permettant aux propriétaires d'utiliser leurs propres comptes Stripe pour recevoir les fonds directement.

### Concepts

- **Plateforme** : l'application Devis Generator (qui encaisse une commission).
- **Compte Stripe** : le compte de la plateforme (créé par l'équipe d'ops).
- **Compte connecté** : le compte Stripe du propriétaire individuel.
- **Direct charge** : frais appliqués au compte du propriétaire, commission versée à la plateforme.

### Préalables de création de plateforme

Avant de pouvoir utiliser Stripe Connect :

1. **Créer un compte Stripe professionnel** :
   - [https://stripe.com/connect](https://stripe.com/connect)
   - Type : "En tant que plateforme"
   - Secteur : Facturation / Invoicing

2. **Valider votre identité** :
   - KYC : données d'identité, adresse
   - Documentations : extrait K-bis ou statuts

3. **Configurer les frais applicables** :
   - Voir section "Commission" ci-dessous.

4. **Obtenir le Stripe Account ID** :
   - Dashboard → Settings → Account ID
   - Format : `acct_1234567890`

### Express vs. Standard

Deux modes d'onboarding pour les propriétaires :

| Critère | Express | Standard |
|---------|---------|----------|
| **Durée d'intégration** | 5-10 min (formulaire rapide) | 30-60 min (audit complet) |
| **Approbation** | Automatique (généralement) | Manuel (24-48h) |
| **Fonctionnalités** | Basique | Complet (virements multiples, etc.) |
| **Commission Stripe** | 0,5% supplémentaire | 0% (facturation standard) |
| **Cas d'usage** | Freelances, PME | Professionnels établis |

**Défaut** : Express (plus rapide).

### Express : URLs et Account Links

**1. Créer un compte connecté Express** :
```bash
curl https://api.stripe.com/v1/accounts \
  -H "Authorization: Bearer sk_live_..." \
  -d "type=express" \
  -d "country=FR" \
  -d "email=proprietaire@exemple.fr"
```

**Réponse** :
```json
{
  "id": "acct_1ExampleConnect",
  "type": "express",
  "country": "FR",
  "email": "proprietaire@exemple.fr",
  "charges_enabled": false
}
```

**2. Générer un Account Link (URL d'onboarding)** :
```bash
curl https://api.stripe.com/v1/account_links \
  -H "Authorization: Bearer sk_live_..." \
  -d "account=acct_1ExampleConnect" \
  -d "type=account_onboarding" \
  -d "return_url=https://devis.exemple.fr/dashboard" \
  -d "refresh_url=https://devis.exemple.fr/connect/retry"
```

**Réponse** :
```json
{
  "object": "account_link",
  "created": 1699861200,
  "expires_at": 1699947600,
  "url": "https://connect.stripe.com/onboarding/acct_1ExampleConnect?client_id=...",
  "account": "acct_1ExampleConnect"
}
```

**3. Rediriger le propriétaire** :
```html
<a href="https://connect.stripe.com/onboarding/..." target="_blank">
  Activer Stripe Connect
</a>
```

**URL de retour** (`return_url`) :
- Le propriétaire est redirigé après avoir complété l'onboarding.
- Vérifier que `charges_enabled=true` avant d'accepter les paiements.

**URL de rafraîchissement** (`refresh_url`) :
- Si l'onboarding échoue ou est abandonné, le propriétaire peut réessayer.
- Recréer un nouveau Account Link avec les mêmes paramètres.

### Standard : OAuth et Account Links

**1. Créer un compte connecté Standard** :
```bash
curl https://api.stripe.com/v1/accounts \
  -H "Authorization: Bearer sk_live_..." \
  -d "type=standard" \
  -d "country=FR" \
  -d "email=proprietaire@exemple.fr"
```

**2. Rediriger vers l'OAuth Stripe** :

```
https://connect.stripe.com/oauth/authorize?
  client_id=ca_...&
  state=random_uuid&
  redirect_uri=https://devis.exemple.fr/connect/callback&
  suggested_capabilities=payment_method_via_code,card_payments
```

**3. Callback et enregistrement du compte** :
```bash
curl https://api.stripe.com/oauth/token \
  -d "grant_type=authorization_code" \
  -d "code=ac_..." \
  -d "client_id=ca_..." \
  -d "client_secret=sk_live_..."
```

**Réponse** :
```json
{
  "access_token": "sk_live_...",
  "stripe_user_id": "acct_1ExampleStandard",
  "refresh_token": "rt_..."
}
```

### Webhooks Stripe Connect

Les webhooks doivent être enregistrés pour les événements Connect :

**Événements à suivre** :
- `charge.succeeded` : paiement autorisé et capturé.
- `charge.failed` : paiement rejeté.
- `charge.refunded` : remboursement effectué.
- `account.updated` : informations du compte mises à jour (KYC).

**Endpoint webhook** :
```
POST https://devis.exemple.fr/api/v1/payments/webhooks/stripe-connect
```

**Enregistrement** :
- Dashboard Stripe → Webhooks → Ajouter endpoint
- Événements : sélectionner ceux listés ci-dessus.
- Clé de signature : `whsec_...`

### Commission et frais

La plateforme encaisse une commission directement du paiement :

**Modèle de facturation** :
```
Montant facture : 1 000 EUR

Flux :
1. Client paie 1 000 EUR via Stripe Connect (compte du propriétaire).
2. Commission plateforme (ex : 5%) : 50 EUR vers compte plateforme.
3. Propriétaire reçoit : 950 EUR.

Stripe applique sa commission (2,4% + 0,30 EUR) :
- 950 EUR → 950 * 0,976 - 0,30 = 925,52 EUR (frais Stripe).
```

**Configuration de la commission** :
```go
// Lors de la création du charge via l'API Stripe
chargeParams := &stripe.ChargeParams{
  Amount:                stripe.Int64(100000), // 1 000 EUR en centimes
  Currency:              stripe.String(string(currency.EUR)),
  Customer:              stripe.String(customerID),
  OnBehalfOf:            stripe.String(connectAccountID),
  TransferData: &stripe.ChargeTransferDataParams{
    Destination: stripe.String(connectAccountID),
    Amount:      stripe.Int64(95000), // 950 EUR après commission 5%
  },
  ApplicationFeeAmount:  stripe.Int64(5000), // 50 EUR commission
}
```

**Commission minimale recommandée** : 2-5% (de préférence configurable par client).

### Test vs. Live

**Mode test** :
- Clés : `sk_test_`, `pk_test_`
- Comptes connectés test : `acct_1...test`
- Webhooks : URLs de test
- Commandes de test Stripe : utiliser les numéros de carte de test Stripe

**Mode live** :
- Clés : `sk_live_`, `pk_live_`
- Comptes connectés réels : `acct_1...`
- Webhooks : URLs de production (HTTPS obligatoire)
- Paiements réels : les fonds sont transférés immédiatement

**Configuration** :
```bash
# Test
STRIPE_SECRET_KEY=sk_test_51...
STRIPE_CONNECT_MODE=sandbox

# Live
STRIPE_SECRET_KEY=sk_live_51...
STRIPE_CONNECT_MODE=production
```

### Checklist opérateur avant production

Avant de lancer Stripe Connect en production :

#### KYC / AML (Connaissance du client)

- [ ] Identité du propriétaire vérifiée (pièce d'identité)
- [ ] Adresse physique confirmée
- [ ] Source de fonds déclarée (facturation légitime)
- [ ] Pas de red-flag : liste noire sanctions, OFAC, etc.

#### Fiscalité

- [ ] Numéro SIRET/SIREN confirmé
- [ ] Régime TVA déterminé (franchise, normal, réel)
- [ ] Acceptation des CGU (contribution à l'impôt sur le revenu ou IS)

#### Support et escalade

- [ ] Équipe support identifiée et contactable
- [ ] Procédure de signalement de fraude
- [ ] Plan d'action en cas de paiement contesté (chargeback)

#### Conformité légale

- [ ] CGV et politique de confidentialité acceptées
- [ ] Clause de commission affichée sur les documents
- [ ] Avertissement : "Stripe est un intermédiaire financier, pas votre banque"

### Acknowledgement de conformité

**Important** : Stripe Connect est une solution technologique, **pas une solution juridique**. La plateforme reste responsable de :

1. **Prévention de la fraude** : vérifier les propriétaires et les montants.
2. **Conformité réglementaire** : respecter les lois de chaque juridiction (RGPD, DSP2, etc.).
3. **Signalement de transactions suspectes** : déclarer aux autorités compétentes si nécessaire.
4. **Gestion des litiges** : établir une procédure de résolution des litiges avec les clients.

**Stripe ne supprime pas ces obligations**. L'utilisation de Stripe Connect ne vous exonère pas de la responsabilité légale.

---

## 7. Modèle de statut unifié

Tous les modes de paiement utilisent le même modèle de statut, indépendant du provider.

### États de paiement

```
unpaid → pending → paid
```

| État | Signification | Transition | Notes |
|------|---------------|-----------|-------|
| `unpaid` | Non payé | Initial, manuel | Défaut pour tout document. |
| `pending` | En attente | Auto (webhook), manuel | Paiement initié mais non confirmé. |
| `paid` | Payé | Auto (webhook), manuel | Paiement confirmé, remise de compte possible. |

### Interface NormalizedPaymentEvent

Chaque événement de paiement (webhook, marquage manuel) suit une structure unifiée :

```json
{
  "provider": "stripe|mollie|paypal|rib|manual",
  "external_event_id": "ch_1234|evt_123|PAYID-...|manual-token",
  "document_id": "550e8400-e29b-41d4-a716-446655440000",
  "new_status": "pending|paid",
  "occurred_at": "2024-11-13T10:30:00Z"
}
```

### Événements idempotents

Chaque événement est identifié par (`provider`, `external_event_id`) :
- Si l'événement a déjà été traité, il est ignoré (rejeu sûr).
- Les statuts antérieurs sont respectés : `unpaid → pending → paid` (pas de retour).

### Action manuelle

**Endpoint API** (admin uniquement) :
```
PATCH /api/v1/documents/{doc_id}/payment-status
Authorization: Bearer <jwt_admin>
Content-Type: application/json

{
  "new_status": "paid",
  "reason": "Paiement reçu par chèque le 2024-11-13"
}
```

**Audit** :
```json
{
  "document_id": "550e8400-e29b-41d4-a716-446655440000",
  "transition": {
    "from": "pending",
    "to": "paid",
    "triggered_at": "2024-11-13T10:30:00Z",
    "triggered_by": "admin@exemple.fr",
    "reason": "Paiement reçu par chèque le 2024-11-13",
    "manual": true
  }
}
```

### Observabilité

**Logs structurés** :
```go
slog.Info("payment transition",
  "document_id", docID,
  "from", oldStatus,
  "to", newStatus,
  "provider", provider,
  "event_id", eventID,
  "timestamp", now)
```

**Métriques** :
- `payments_transition_total{from="unpaid",to="pending",provider="stripe"}` (counter)
- `payments_transition_duration_seconds{provider="stripe"}` (histogram)
- `payment_event_ingestion_errors_total{provider="stripe"}` (counter)

**Alertes** :
- Taux d'erreur d'ingestion > 1% en 5 min.
- Webhook rejeté > 10 fois pour le même provider en 1 h.
- Transition manuelle déclenchée en dehors des heures de travail (investigation).

### Runbook d'incident

#### Symptôme : Paiement marqué "pending" mais devrait être "paid"

**Diagnostic** :
1. Vérifier les logs webhook du provider.
2. Aller sur le tableau de bord du provider (Stripe, Mollie) et vérifier que le paiement est bien `succeeded`.
3. Vérifier si le webhook a été reçu (query : `SELECT * FROM payment_events WHERE document_id = ?`).

**Résolution** :
```bash
# Option 1 : rejeu du webhook
curl -X POST /api/v1/admin/payments/webhook-replay \
  -H "Authorization: Bearer <jwt_admin>" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "stripe",
    "external_event_id": "ch_1234...",
    "document_id": "550e8400-..."
  }'

# Option 2 : marquage manuel
curl -X PATCH /api/v1/documents/{doc_id}/payment-status \
  -H "Authorization: Bearer <jwt_admin>" \
  -H "Content-Type: application/json" \
  -d '{"new_status": "paid", "reason": "Rejeu webhook échoué ; marqué manuellement"}'
```

#### Symptôme : Webhook rejeté (403 Forbidden)

**Diagnostic** :
1. Vérifier la signature du webhook dans les logs : `webhook_signature_validation_failed`.
2. Confirmer que la clé webhook est correcte dans les env vars.

**Résolution** :
```bash
# Regénérer la clé webhook chez le provider
# Ex : Stripe Dashboard → Webhooks → Clé de signature
export STRIPE_WEBHOOK_SECRET=whsec_new_...
systemctl restart devis-generator
```

#### Symptôme : Paiement perdu (ni "pending" ni "paid")

**Diagnostic** :
1. Query : `SELECT * FROM documents WHERE id = ?` → vérifier `payment_status`.
2. Query : `SELECT * FROM payment_events WHERE document_id = ?` → voir l'historique.

**Résolution** :
```sql
-- Vérifier manuellement chez le provider
-- Puis, si confirmé payé :
UPDATE documents SET payment_status = 'paid' 
WHERE id = ? AND payment_status != 'paid';

-- Créer un événement d'audit
INSERT INTO payment_events (provider, event_id, document_id, old_status, new_status, created_at)
VALUES ('manual_recovery', 'recovery_' || gen_random_uuid(), ?, 'pending', 'paid', now());
```

### Rejeu d'événements

**Endpoint API** (admin uniquement) :
```
POST /api/v1/admin/payments/event-replay
Authorization: Bearer <jwt_admin>
Content-Type: application/json

{
  "provider": "stripe",
  "external_event_id": "ch_1234...",
  "new_status": "paid"
}
```

**Idempotence** :
```
1er rejeu : "unpaid" → "paid" ✓
2e rejeu : "paid" → "paid" (pas de changement, OK) ✓
Transition invalide ("paid" → "unpaid") : rejetée ✗
```

---

## 8. Migration et rollback

Procédures sûres pour basculer entre les modes ou ajouter de nouveaux modes.

### Migration : Activation d'un nouveau mode

**Scénario** : vous aviez RIB seul, vous activez BYO.

**Étapes** :

1. **Préparer** :
   ```bash
   # Générer les clés BYO (Stripe, Mollie ou PayPal)
   export PAYMENT_BYO_ENABLED=true
   export STRIPE_SECRET_KEY=sk_test_...
   export STRIPE_WEBHOOK_SECRET=whsec_...
   ```

2. **Tester en staging** :
   ```bash
   # Déployer sur staging avec les nouvelles clés
   docker compose -f compose.staging.yaml up -d
   
   # Tester les endpoints BYO
   curl https://staging.exemple.fr/api/v1/payments/capabilities
   # Réponse : {"active": ["rib", "byo"]}
   
   # Créer un test de paiement BYO
   ```

3. **Valider les webhooks** :
   ```bash
   # Envoyer un webhook de test depuis Stripe
   # Vérifier que le statut du document passe de "unpaid" à "pending"
   ```

4. **Déployer en production** :
   ```bash
   # Mettre à jour les variables d'environnement en production
   PAYMENT_BYO_ENABLED=true
   STRIPE_SECRET_KEY=sk_live_...
   STRIPE_WEBHOOK_SECRET=whsec_live_...
   
   # Redémarrer l'application (avec minimal downtime)
   systemctl restart devis-generator
   # ou : kubectl rollout restart deployment/devis-generator
   ```

5. **Notifier** :
   - Email aux propriétaires : "Nouveau mode BYO disponible"
   - Afficher un badge "Nouveau" dans l'UI

### Rollback : Désactivation d'un mode

**Scénario** : BYO a un problème critique, vous devez le désactiver.

**Étapes** :

1. **Identifier l'impact** :
   ```sql
   -- Quel nombre de documents attendent un paiement BYO ?
   SELECT COUNT(*) FROM documents 
   WHERE payment_mode = 'byo' AND payment_status != 'paid';
   ```

2. **Notifier les clients** :
   - Email : "Mode BYO actuellement indisponible ; les paiements précédents restent valides"
   - Masquer les formulaires BYO côté UI

3. **Désactiver** :
   ```bash
   export PAYMENT_BYO_ENABLED=false
   systemctl restart devis-generator
   ```

4. **Rejeu des événements** (après correction) :
   ```bash
   # Une fois le problème corrigé
   export PAYMENT_BYO_ENABLED=true
   export STRIPE_SECRET_KEY=sk_live_... (clé correcte)
   systemctl restart devis-generator
   
   # Rejeu des webhooks non traités
   curl -X POST /api/v1/admin/payments/event-replay \
     -H "Authorization: Bearer <jwt_admin>" \
     -d '[...]'
   ```

### Exemples `.env` sûrs avec placeholders

**Développement** :
```bash
# .env.development
PAYMENT_RIB_ENABLED=true
PAYMENT_BYO_ENABLED=true
PAYMENT_STRIPE_CONNECT_ENABLED=false

STRIPE_SECRET_KEY=sk_test_example_development_key_replace_me
STRIPE_PUBLISHABLE_KEY=pk_test_example_development_key_replace_me
STRIPE_WEBHOOK_SECRET=whsec_test_example_webhook_replace_me

# Secrets chiffrement RIB/OAuth/connectés
PAYMENT_SECRETS_KEYS='{"dev": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}'
PAYMENT_SECRETS_ACTIVE_KEY_ID=dev
```

**Staging** (clés de test Stripe) :
```bash
# .env.staging (non pushé, secret manager)
PAYMENT_RIB_ENABLED=true
PAYMENT_BYO_ENABLED=true
PAYMENT_STRIPE_CONNECT_ENABLED=true

STRIPE_SECRET_KEY=sk_test_production_like_key_from_stripe_dashboard
STRIPE_PUBLISHABLE_KEY=pk_test_production_like_key_from_stripe_dashboard
STRIPE_WEBHOOK_SECRET=whsec_test_staging_webhook_secret

PAYMENT_SECRETS_KEYS='{"2024-11-13": "X7vL9mK2pQr3tJ8nW5xZc4bD6eF1aG0hS2iP9kL4mN="}'
PAYMENT_SECRETS_ACTIVE_KEY_ID=2024-11-13
```

**Production** (clés live, strictement secrets) :
```bash
# Variables d'env en secret manager (AWS Secrets Manager, HashiCorp Vault, etc.)
# Jamais dans le dépôt Git
# ⚠ Exemple de structure seulement :

PAYMENT_RIB_ENABLED=true
PAYMENT_BYO_ENABLED=true
PAYMENT_STRIPE_CONNECT_ENABLED=true

STRIPE_SECRET_KEY=sk_live_51_production_live_key_do_not_commit
STRIPE_PUBLISHABLE_KEY=pk_live_51_production_live_key_do_not_commit
STRIPE_WEBHOOK_SECRET=whsec_live_production_webhook_secret_do_not_commit

# Trousseau de secrets (optionnel : stocké aussi dans secret manager)
PAYMENT_SECRETS_KEYS='{"2024-11-13": "Y8pL2mK9qW3rT5sU7vX0yZ1aB4cD6eF9gH2iJ5kL=", "2024-10-01": "..."}'
PAYMENT_SECRETS_ACTIVE_KEY_ID=2024-11-13
```

**Validations** :
```bash
# Vérifier que les clés ne sont pas pushées
git grep -i "sk_live_\|sk_test_\|whsec_" -- '*.md' '*.go' | grep -v "examples\|docs" && echo "❌ Secrets trouvés!" || echo "✓ Aucun secret en clair"

# Vérifier les placeholders dans les configs
grep "replace_me\|CHANGE_ME\|example" .env.*.template
```

---

## Intégration dans README et index

Ce guide doit être lié depuis :

1. **README.md** (section "Configuration de paiement") :
   ```markdown
   ## Variables d'environnement — Paiement
   
   Pour configurer les trois modes de paiement (RIB, BYO, Stripe Connect),
   consulter [docs/PAIEMENT.md](docs/PAIEMENT.md).
   ```

2. **docs/index.html** (page de présentation) :
   ```html
   <section id="paiement">
     <h3>Modes de paiement</h3>
     <p>
       Trois modes flexibles et sécurisés :
       <a href="PAIEMENT.md">RIB, BYO (votre provider), et Stripe Connect</a>.
     </p>
   </section>
   ```

3. **docs/technique.md** (doc technique existante) :
   - Référence la section "Paiement" de PAIEMENT.md
   - Lien vers les exemples `.env`

---

## Cohérence avec `.env.example` et OpenAPI

### `.env.example`

Ajouter les variables de paiement au fichier `.env.example` :

```bash
# ============================================================================
# CONFIGURATION DU PAIEMENT — modes RIB / BYO / Stripe Connect
# ============================================================================

# Activer / désactiver les modes de paiement
# Valeurs : 1, true, yes, on (pour actif) | 0, false, no, off (pour inactif)
PAYMENT_RIB_ENABLED=true
PAYMENT_BYO_ENABLED=false
PAYMENT_STRIPE_CONNECT_ENABLED=false

# Mode BYO — Clés Stripe (remplacer par les vôtres ; test mode par défaut)
# STRIPE_SECRET_KEY=sk_test_YOUR_SECRET_KEY_HERE
# STRIPE_PUBLISHABLE_KEY=pk_test_YOUR_PUBLISHABLE_KEY_HERE
# STRIPE_WEBHOOK_SECRET=whsec_test_YOUR_WEBHOOK_SECRET_HERE

# Mode Stripe Connect (optionnel)
# STRIPE_CONNECT_ACCOUNT_ID=acct_YOUR_EXPRESS_ACCOUNT_ID_HERE

# Trousseau de secrets pour RIB / OAuth (JSON des clés AES-256 en base64)
# PAYMENT_SECRETS_KEYS='{"2024-11-13": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}'
# PAYMENT_SECRETS_ACTIVE_KEY_ID=2024-11-13
```

### OpenAPI / Swagger

Ajouter les endpoints de paiement à la spec OpenAPI :

```yaml
/api/v1/payments/capabilities:
  get:
    summary: "Obtenir les modes de paiement actifs"
    responses:
      200:
        content:
          application/json:
            schema:
              type: object
              properties:
                active:
                  type: array
                  items:
                    type: string
                    enum: ["rib", "byo", "stripe_connect"]
            examples:
              default:
                value: { "active": ["rib"] }
              all_enabled:
                value: { "active": ["rib", "byo", "stripe_connect"] }

/api/v1/documents/{id}/payment-status:
  patch:
    summary: "Mettre à jour le statut de paiement (admin)"
    parameters:
      - in: path
        name: id
        required: true
        schema: { type: string, format: uuid }
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            properties:
              new_status:
                type: string
                enum: ["unpaid", "pending", "paid"]
              reason:
                type: string
                description: "Raison du changement (optionnel, audit)"
    responses:
      200:
        content:
          application/json:
            schema:
              type: object
              properties:
                id: { type: string, format: uuid }
                payment_status: { type: string }
                marked_at: { type: string, format: date-time }
```

---

## Commandes et vérifications

### Vérifier la configuration au démarrage

```bash
# Logs au démarrage
docker logs devis-generator | grep -i "payment"

# Exemple de sortie correcte
# INFO payment configuration loaded: rib=true byo=false connect=false
# INFO payment registry initialized with 1 active mode(s)
```

### Tester les endpoints

```bash
# Vérifier les modes actifs
curl http://localhost:8080/api/v1/payments/capabilities | jq

# Vérifier les permissions admin
curl -H "Authorization: Bearer $JWT_ADMIN" \
  http://localhost:8080/api/v1/admin/payments/capabilities | jq

# Tester le webhook Stripe (en development)
curl -X POST http://localhost:8080/api/v1/payments/webhooks/stripe \
  -H "Content-Type: application/json" \
  -H "Stripe-Signature: t=...,v1=..." \
  -d '{"type": "charge.succeeded", "data": {"object": {...}}}'
```

### Lint et validation

```bash
# Valider les templates `.env`
shellcheck -S warning .env.*.template

# Vérifier l'absence de secrets en clair
git-secrets scan
grep -r "sk_live_\|sk_test_\|whsec_" . --include="*.go" --include="*.md" | grep -v "examples\|docs" || echo "OK"

# Valider les configurations YAML Stripe
striped validate-stripe-config.yaml
```

---

## Annex : Schéma de base de données

Les migrations incluent les tables nécessaires pour chaque mode :

### Tables

- `payment_events` : historique des événements webhook
- `payment_iban` : IBAN chiffré du propriétaire (clé unique par tenant)
- `payment_byo_provider_accounts` : comptes connectés BYO (Stripe, Mollie, PayPal)
- `payment_secrets` : secrets chiffrés (IBAN, clés API, tokens OAuth)
- `documents.payment_status` : statut de paiement (unpaid, pending, paid)

### Requêtes utiles

```sql
-- Voir tous les paiements en attente (tous les modes)
SELECT id, payment_status, created_at FROM documents 
WHERE payment_status = 'pending' 
ORDER BY created_at DESC;

-- Voir l'historique de paiement d'un document
SELECT provider, external_event_id, new_status, occurred_at FROM payment_events 
WHERE document_id = '550e8400-e29b-41d4-a716-446655440000'
ORDER BY occurred_at DESC;

-- Nombre de paiements par mode
SELECT payment_mode, payment_status, COUNT(*) FROM documents 
GROUP BY payment_mode, payment_status;
```

---

## Support et escalade

### Contacter le support

- **Stripe** : https://support.stripe.com
- **Mollie** : https://help.mollie.com
- **PayPal** : https://developer.paypal.com/support

### Problèmes connus

| Mode | Problème | Workaround |
|------|----------|-----------|
| RIB | IBAN non accepté après changement | Vérifier la somme de contrôle (ISO 13616) |
| BYO/Stripe | Webhook non reçu en local | Utiliser ngrok pour exposer localhost en HTTPS |
| Stripe Connect | Compte Express bloqué (KYC) | Contacter Stripe Support avec l'Account ID |

---

**Version** : 1.0
**Date** : 13 novembre 2024
**Statut** : Documentation liée et validée après T1-T6
