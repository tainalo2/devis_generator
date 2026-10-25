# Décisions produit — Devis Generator : VIBED

Décisions validées par le propriétaire. Ce fichier est la référence courte ; l’implémentation complète arrivera par itérations.

## Identité

- **Nom officiel** : `Devis Generator : VIBED`
- **Licence (code Vibed)** : Apache-2.0 — voir `LICENSE.vibed`
- **Licence (site vanilla historique à la racine)** : GPL-3.0 — fichier `LICENSE` inchangé
- **Langue d’interface** : français uniquement

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

## Comptes

Optionnels : **mode invité** (sans compte) **et** compte **email / mot de passe**.

## PDF

Génération **côté serveur** (pas uniquement dans le navigateur).

## Hors scope immédiat

Auth complète, CRUD métier, sources `legal` officielles, moteur PDF : prévus, non livrés dans le socle.
