# Serveur MCP — Devis Generator

Serveur stdio (`backend/cmd/devis-mcp`) exposant 5 tools métier au-dessus de l'API publique.

## Installation / configuration

Variables d'environnement requises :

- `VIBED_API_URL` — URL de base de l'API publique.
- `VIBED_API_KEY` — clé API du compte émetteur.

L'authentification est propagée depuis le contexte MCP : ne jamais passer de clé dans les arguments d'un tool.

Exemple de configuration client MCP (Claude Desktop / compatible) :

```json
{
  "mcpServers": {
    "devis-generator": {
      "command": "devis-mcp",
      "env": {
        "VIBED_API_URL": "https://api.example.com",
        "VIBED_API_KEY": "<clé API du compte>"
      }
    }
  }
}
```

## Tools V1

1. `create_devis` — devis pour un client à partir de lignes.
2. `create_facture` — facture depuis un devis (`depuis_devis_id`) OU client+lignes.
3. `create_acompte` — acompte rattaché à un devis (pourcentage OU montant_ht).
4. `get_document_link` — lien public stable d'un document.
5. `list_documents` — recherche paginée par curseur.

Montants en chaînes décimales + devise ISO 4217. Totaux calculés serveur. Voir `docs/` contrat T1 et `mcp-tools-v1.md` pour les signatures complètes.

Le schéma `inputSchema` renvoyé par `tools/list` décrit précisément les arguments (lignes, `idempotency_key`) et reflète la validation réellement appliquée.

### Format des lignes de création

`create_devis` et la variante client+lignes de `create_facture` reçoivent les
lignes dans le format MCP public suivant :

```json
{
  "description": "Conseil",
  "quantite": "12.345",
  "prix_unitaire_ht": "0.1",
  "tax_rate_bps": 2000
}
```

- `quantite` est une chaîne décimale exprimée en unités ; le serveur MCP la
  convertit en milli-unités pour l'API interne (`"12.345"` → `12345`).
- `prix_unitaire_ht` est une chaîne décimale exprimée en euros HT ; le serveur
  MCP la convertit en centimes pour l'API interne (`"0.1"` → `10`).
- Si une valeur a plus de précision que l'unité interne (trois décimales pour
  la quantité, deux pour le prix), la conversion applique explicitement un
  arrondi commercial au plus proche, les demis étant arrondis en s'éloignant
  de zéro. Par exemple, `"1.2345"` devient 1 235 milli-unités et `"1.005"`
  devient 101 centimes.

Les champs internes `quantity_milli` et `unit_price_cents` ne font pas partie
du contrat MCP public.

Schéma d'une ligne (`additionalProperties: false`, toute autre propriété est
refusée, y compris `quantity_milli` et `unit_price_cents`) :

| Champ | Type | Obligatoire | Contrainte |
|---|---|---|---|
| `description` | chaîne | oui | 1 à 2000 caractères, non vide |
| `quantite` | chaîne décimale | oui | **strictement positive**. Motif du schéma non signé : `^\+?([0-9]+(\.[0-9]*)?\|\.[0-9]+)$` (le `\|` est une alternation) (« + » optionnel, pas de `-`, pas de virgule). `0`/`0.0` satisfont le motif mais sont refusés à l'exécution, comme toute valeur arrondie à zéro au millième |
| `prix_unitaire_ht` | chaîne décimale | oui | prix négatif autorisé (ligne de crédit) ; motif signé `^[+-]?…$` |
| `tax_rate_bps` | entier | non (défaut 0) | `0`, `210`, `550`, `1000` ou `2000` ; forcé à 0 en régime `franchise_293b` (la valeur fournie y est ignorée, y compris pour l'idempotence) |
| `discount_bps` | entier | non (défaut 0) | de 0 à 10000 |

`lignes` contient au plus 500 lignes. Une ligne invalide est rejetée par le
serveur MCP (`status: "error"`) avant tout appel à l'API.

## Idempotence des créations

`idempotency_key` est **obligatoire** pour `create_devis`, `create_facture` et
`create_acompte` (1 à 255 caractères parmi `A-Z a-z 0-9 . _ : / -`, par exemple
un UUID). Sans clé, le serveur renvoie
`{"status":"clarification","missing":["idempotency_key"]}` et rien n'est créé ;
une clé au format invalide donne `status: "error"`.

- Le serveur MCP transmet la clé à l'API dans l'en-tête `Idempotency-Key` de la
  requête de création (`POST /documents`, `POST /documents/{id}/convert-to-invoice`
  ou `POST /documents/{id}/deposit-invoice`).
- L'API la rend durable en PostgreSQL (table `document_idempotency_keys`,
  migration `000022`), pour la portée **(compte propriétaire, opération, clé)**.
  La clé est insérée dans la même transaction que la création du document ; la
  clé primaire sérialise les requêtes concurrentes.
- **Même clé + même requête** : aucun nouveau document ; l'API renvoie le
  document déjà créé (HTTP 200 + `Idempotent-Replayed: true`, au lieu de 201).
  Le document renvoyé est son état courant.
- **Même clé + requête différente** : HTTP 409 (le tool renvoie `status: "error"`).
  La requête comparée est le SHA-256 d'une **représentation canonique** construite
  APRÈS application des défauts métier : devise omise ou vide = `EUR` (même hash,
  le rejeu réussit) ; seuls les champs d'entrée métier comptent (type, titre,
  dates, client, régime TVA, notes, conditions, document lié et, par ligne :
  description, quantité, prix unitaire, remise, taux de TVA, dans l'ordre des
  lignes). Le taux de TVA haché est le taux **effectif** (règle de
  `billing.Compute`) : en `franchise_293b`, taux omis, `0` et `2000` donnent le
  même hash (rejeu réussi) ; hors franchise, `2000` et `0` sont deux requêtes
  différentes (HTTP 409). Les champs dérivés ou sans effet (`id`, `position`,
  `tax_cents`, `total_cents`) et l'ordre des clés JSON sont ignorés. Une vraie
  différence (quantité, prix, description…) donne un hash différent, donc HTTP
  409. Pour `create_acompte`, la requête comparée est (document d'origine, lignes
  fournies) ; le régime TVA vient de l'origine et n'est pas lu pour calculer le
  hash : les taux de ces lignes sont donc hachés tels quels (un acompte sur
  origine en franchise rejoué avec un `tax_rate_bps` différent donne un 409).
- **Ordre de résolution** : une clé **existante** (même compte, opération, clé,
  non expirée) est résolue AVANT les prérequis mutables (existence du client ou
  du document lié, chargement du document d'origine d'une conversion ou d'un
  acompte) : même hash => rejeu du document créé, même si le client ou
  l'origine ont été supprimés depuis ; hash différent => 409 ; document créé
  supprimé => 409 (tombstone). Seule une clé **nouvelle** vérifie ces
  prérequis (client ou origine inconnus => erreur normale, clé non consommée) ;
  sa réservation reste atomique dans la transaction de création.
- Une création en échec (rollback) ne consomme pas la clé : elle peut être retentée.
- **Suppression du document = tombstone** : supprimer un brouillon créé avec une
  clé ne supprime pas la clé (`document_id` n'a aucune clé étrangère). Rejouer la
  même clé avant expiration renvoie **HTTP 409** (« Le document créé avec cette
  Idempotency-Key a été supprimé : utilisez une nouvelle Idempotency-Key… ») et
  ne recrée jamais le document ; une clé neuve en crée un nouveau.
- **Expiration : 24 h** après la création (tombstones compris) ; passé ce délai,
  la clé peut servir à une nouvelle création.
- **Purge** : (1) opportuniste, par lots de 100, des clés expirées du compte qui
  crée ; (2) **tâche périodique globale** démarrée par `cmd/server` : une purge
  au démarrage puis toutes les heures, tous comptes confondus (y compris
  inactifs), par lots de 1000 et au plus 50 lots par passage (le reliquat est
  repris au passage suivant). Elle s'arrête avec l'arrêt du serveur, journalise
  `clés d'idempotence expirées purgées` avec le nombre de lignes (`purged`) et
  journalise sans s'arrêter les erreurs (`purge des clés d'idempotence`).
- Hors MCP, l'en-tête `Idempotency-Key` est facultatif sur ces endpoints (sans
  lui, chaque appel crée un document, comportement historique) ; s'il est
  présent, il doit respecter le même format, sinon HTTP 400.
- Les outils de lecture (`get_document_link`, `list_documents`) n'ont pas de clé.

## Retour standardisé

Chaque création renvoie un identifiant opaque et un lien direct stable (HTTPS). Pour les devis remplacés, `get_document_link` renvoie la version courante et peut signaler le remplacement. Liens de facture/acompte permanents et non révocables.

## Clarifications / erreurs

En cas d'ambiguïté, le serveur renvoie un JSON structuré `{status, missing, alternatives}` plutôt qu'une création partielle :

- création sans `idempotency_key` : `missing: ["idempotency_key"]` (vérifié après les ambiguïtés ci-dessous)
- devis : `valide_jusqu_au` XOR `delai_validite_jours`
- facture : `depuis_devis_id` XOR (`client`+`lignes`) ; `date_echeance` XOR `delai_paiement_jours`
- acompte : `devis_id` + exactement un de `pourcentage`/`montant_ht`

## Exemples de prompts

1. "Crée un devis pour Acme (acme@example.com) : 3 jours de conseil à 500 € HT/jour, TVA 20 %, valide 30 jours."
2. "Transforme le devis DV-2026-001 en facture avec échéance au 2026-12-15."
3. "Émets un acompte de 30 % sur le devis DV-2026-001."
4. "Donne-moi le lien du devis DV-2026-001."
5. "Liste les factures impayées du client Acme."

## Vérification

```bash
cd backend && go test ./internal/mcpserver/ ./... && go vet ./...
```
