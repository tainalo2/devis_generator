# Stripe Connect (mode plateforme) — conformité opérationnelle minimale

Ce mode est **désactivé par défaut** (`PAYMENT_STRIPE_CONNECT_ENABLED=false`). L'opérateur self-hoster doit explicitement fournir : clés plateforme Stripe (test ou live, jamais mélangées), webhook secret, URL publique HTTPS, et acknowledgement de conformité (`STRIPE_CONNECT_COMPLIANCE_ACK=true`).

## Rôle de facilitateur
L'opérateur agit comme facilitateur de paiement via Stripe Connect (direct charges). La commission est calculée côté serveur (basis points + fixe, plafonnée) et transmise via `application_fee_amount`. Elle n'est jamais configurable par l'utilisateur final.

## Onboarding
Comptes Express uniquement, via Account Links. Reprise possible. Aucun paiement tant que `charges_enabled` et `payouts_enabled` et capabilities requises ne sont pas actives.

## Secrets
Les identifiants compte connecté sont persistés via le package `paymentsecrets` (AES-256-GCM). Jamais de tokens en clair. Diagnostics admin sans secrets.

## Webhooks
Signatures vérifiées. Événements plateforme et comptes connectés routés dans le ledger idempotent. Pas de rétrogradation silencieuse d'un statut `paid`.

## KYC/AML/Fiscalité
Le code n'automatise pas KYC/AML/fiscalité au-delà des capacités Stripe. L'opérateur reste responsable.