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

Montants en chaînes décimales + devise ISO 4217. Totaux calculés serveur. `idempotency_key` obligatoire pour toute création. Voir `docs/` contrat T1 et `mcp-tools-v1.md` pour les signatures complètes.

## Retour standardisé

Chaque création renvoie un identifiant opaque et un lien direct stable (HTTPS). Pour les devis remplacés, `get_document_link` renvoie la version courante et peut signaler le remplacement. Liens de facture/acompte permanents et non révocables.

## Clarifications / erreurs

En cas d'ambiguïté, le serveur renvoie un JSON structuré `{status, missing, alternatives}` plutôt qu'une création partielle :

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
