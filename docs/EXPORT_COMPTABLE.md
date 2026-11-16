# Guide export comptable — données pour logiciels de comptabilité

Documentation complète pour déployer et utiliser l'export comptable : colonnes disponibles, limitations légales, intégration avec vos outils comptables.

## Table des matières

1. [Vue d'ensemble](#1-vue-densemble)
2. [Schéma des colonnes](#2-schéma-des-colonnes)
3. [Limitations légales](#3-limitations-légales)
4. [Formats et encodage](#4-formats-et-encodage)
5. [Activation et filtrage](#5-activation-et-filtrage)
6. [Intégration comptable](#6-intégration-comptable)
7. [Fichier exemple](#7-fichier-exemple)
8. [Évolutions futures](#8-évolutions-futures)

---

## 1. Vue d'ensemble

L'export comptable vous permet de télécharger vos documents (devis, factures, acomptes, avoirs) au format **CSV** pour les importer dans votre logiciel de comptabilité, Excel, ou les transmettre à votre expert-comptable.

### À quoi sert-il ?

- Alimenter votre logiciel de comptabilité (Ciel, Sage, Pennylane, etc.)
- Analyser vos revenus par période dans Excel ou Google Sheets
- Archiver vos documents et leurs statuts de paiement
- Préparer votre déclaration fiscale ou vos bilans annuels

### Ce que ce n'est PAS

- **Pas un FEC (Fichier des Écritures Comptables)** : ce n'est pas le format réglementaire officiel français, qui s'adresse aux entreprises en comptabilité double-entrée (journaux, comptes, lettrage). Le FEC est **optionnel** pour les TPE/PME cibles.
- **Pas une transmission d'écritures** : vous restez responsable de l'intégration manuelle ou via connecteur dans votre logiciel de compta.
- **Pas obligatoire légalement** : seules les micro-entreprises ont l'obligation de facturation électronique (Factur-X en émission) à partir du 1ᵉʳ septembre 2027. L'export CSV est un **service additionnel** pour votre confort.

---

## 2. Schéma des colonnes

Le CSV contient une en-tête et une ligne par document. Chaque colonne capture une dimension métier clé.

### Format standard (V1)

| Colonne | Type | Exemple | Description |
|---------|------|---------|-------------|
| `numero_document` | texte | `F-2026-0001` | Numéro métier unique du document (voir [Numérotation](DECISIONS.md#numérotation)). Format : `[D\|F\|FA\|A]-AAAA-NNNN` (D = devis, F = facture, FA = facture d'acompte, A = avoir). |
| `type_document` | texte | `facture` | Type du document : `devis` \| `facture` \| `facture_acompte` \| `avoir`. |
| `date_emission` | date ISO | `2026-01-15` | Date d'émission (YYYY-MM-DD). Pour un brouillon non émis, ce champ est vide. |
| `client_nom` | texte | `Dupont SARL` | Raison sociale ou nom du client au moment de l'émission (gelé dans `legal_snapshot`). |
| `client_siret` | texte | `12345678900012` | Numéro SIRET du client (14 chiffres) si renseigné. Vide sinon. |
| `montant_ht` | nombre | `1000.00` | Montant hors taxe en euros, 2 décimales. Séparateur décimal : `.` (point). |
| `taux_tva` | nombre | `20.0` | Taux de TVA en pourcentage (points de base) : `0` (franchise art. 293 B) \| `2.1` \| `5.5` \| `10` \| `20`. Une seule valeur par document (voir ci-dessous). |
| `montant_tva` | nombre | `200.00` | Montant TVA en euros, 2 décimales. Peut être négatif pour un avoir. |
| `montant_ttc` | nombre | `1200.00` | Montant TTC en euros, 2 décimales. HT + TVA. |
| `statut_paiement` | texte | `paye` | Statut de paiement : `en_attente` \| `paye` \| `en_retard` \| `annule`. |
| `date_paiement` | date ISO | `2026-01-20` | Date du paiement enregistré (YYYY-MM-DD). Vide si non payé. |
| `mode_paiement` | texte | `virement` | Mode de paiement connu (si renseigné à l'émission). Peut être vide. |
| `document_origine` | texte | `D-2026-0003` | Numéro du document source pour acomptes (facture d'acompte liée à un devis) ou avoirs (avoir lié à une facture). Vide sinon. |

---

## 3. Limitations légales

### 3.1 Ce n'est pas un FEC réglementaire

Le FEC (Fichier des Écritures Comptables) est un **format imposé par l'Administration fiscale française** pour les entreprises soumises à un certain régime d'imposition. Il répond à une grammaire stricte (XML ou texte fixe-largeur) et s'adresse à la **comptabilité en partie double** (journaux, comptes, lettrage, rapprochements).

**Devis Generator exporte un CSV simple** car :
- **Les micro-entreprises et indépendants cibles n'ont pas toujours besoin d'un FEC** : c'est optionnel jusqu'à certains seuils de chiffre d'affaires ou régimes spéciaux (micro-entreprise, auto-entrepreneur).
- **Un CSV incomplet vaut mieux qu'un FEC mal formé** : un FEC rejeté par l'administration crée des complications ; un CSV importable dans votre logiciel compta ou Excel demeure utile.
- **Pas de risque de comptabilité double** : le CSV liste les factures/paiements simples, pas les écritures comptables.

### 3.2 Pour un vrai FEC : services alternatifs

Si vous êtes obligatoire de livrer un FEC :
1. Exportez ce CSV depuis Devis Generator.
2. Importez les données dans votre logiciel de comptabilité habituel (Ciel, Sage, Pennylane, etc.).
3. Votre logiciel générera l'export FEC conformément au format fiscal.

**Ou**, remontrez ces documents à votre **expert-comptable** ou votre **centre de gestion agréé (CGA)** : ils disposent des outils pour générer FEC et déclarations.

### 3.3 Une seule TVA par document

Devis Generator s'adresse à des micro-entreprises et TPE : **chaque document a un **taux de TVA unique** (0, 2,1, 5,5, 10 ou 20 %)**, appliqué à toutes les lignes.

Si un document comporte plusieurs taux (ex. : 20 % et 5,5 %), le taux exporté sera le **plus fréquent** (calculé une fois à l'export). Cette simplification est appropriée pour 95 % des cas ; si elle pose problème, montez vos documents à taux mixtes dans votre logiciel compta et rectifiez manuellement.

### 3.4 Pas d'obligation d'importation

L'export CSV ne vous oblige **pas** à importer les données dans un logiciel tiers : vous pouvez le conserver tel quel pour archive, le diffuser à un tiers de confiance, ou le détruire. C'est **votre responsabilité** de décider comment utiliser les données exportées.

---

## 4. Formats et encodage

### 4.1 Encodage de caractères

Le fichier CSV est encodé en **UTF-8 avec BOM** (Byte Order Mark : `EF BB BF`). Le BOM est présent pour assurer que les applications Microsoft Office (Excel sur Windows, notamment) **reconnaissent l'accents et les caractères spéciaux dès l'ouverture**, sans intervention de l'utilisateur.

**Exemple** : un client nommé « Noé Équipe » s'affichera correctement dans Excel sans délai ou paramétrage supplémentaire.

### 4.2 Séparateur et guillemets

- **Séparateur de colonnes** : point-virgule `;` (convention française pour Excel/LibreOffice).
- **Séparateur décimal** : point `.` (montants : `1000.00`), pour compatibilité logiciels compta et import-friendly.
- **Guillemets échappés** : une valeur contenant un `;`, une `"` ou une sauts de ligne est entourée de guillemets doubles `"…"`, et chaque `"` interne est doublé (`""`).

**Exemple** : un client nommé `Leroy "Maçon" SARL` s'exportera comme :
```
F-2026-0001;facture;2026-01-15;"Leroy ""Maçon"" SARL";…
```

### 4.3 Protection contre les formules de tableur

Certaines cellules commençant par `=`, `+`, `@` ou `-` suivies d'un nombre peuvent être **interprétées comme des formules** par Excel, posant un risque de sécurité (injection de commandes).

**Devis Generator neutralise ce risque** en préfixant les valeurs suspectes par une apostrophe `'` (qui reste invisible à l'affichage mais bloque l'exécution).

---

## 5. Activation et filtrage

### 5.1 Accès et permissions

L'export comptable est **accessible uniquement aux comptes authentifiés** (login/mot de passe ou session valide). Chaque utilisateur ne peut exporter que **ses propres documents**.

### 5.2 Filtrage par défaut

L'export inclut :
- **Tous les types de documents** : devis, factures, factures d'acompte, avoirs.
- **Documents **émis** uniquement** : les brouillons (non numérotés) ne sont pas inclus.
- **Toutes les plages de dates** : par défaut, depuis la création du compte.

### 5.3 Filtrage optionnel (via UI ou API)

Vous pouvez affiner l'export selon :
- **Plage de dates** : du `2026-01-01` au `2026-12-31` (optionnel).
- **Type de document** : inclure/exclure devis, factures, avoirs (optionnel).
- **Statut de paiement** : en attente, payé, en retard, annulé (optionnel).

**Exemple** : « Exporte-moi toutes les factures payées entre le 1ᵉʳ janvier et le 31 mars 2026 ».

### 5.4 Endpoints API (pour développeurs)

L'export comptable est disponible via l'API HTTP :

**Endpoint** : `GET /api/v1/documents/accounting-export`

**Paramètres de requête** (tous optionnels) :
- `start_date` : date de début (ISO 8601 : `2026-01-01`)
- `end_date` : date de fin (ISO 8601 : `2026-12-31`)
- `document_types` : types à inclure (CSV : `facture,avoir`)
- `payment_status` : statuts de paiement (CSV : `paye,en_retard`)

**Réponse** : fichier CSV avec Content-Type `text/csv; charset=utf-8` et en-tête `Content-Disposition: attachment; filename="documents.csv"`.

**Exemple de requête curl** :

```bash
curl -X GET "https://self-hosted.example.com/api/v1/documents/accounting-export?start_date=2026-01-01&end_date=2026-03-31&payment_status=paye" \
  -H "Authorization: Bearer YOUR_API_TOKEN" \
  -o documents.csv
```

---

## 6. Intégration comptable

### 6.1 Logiciels comptables courants

Devis Generator est **générique** : le CSV peut être importé dans la plupart des outils, mais le chemin dépend du logiciel.

#### 6.1.1 Ciel Compta (Windows)

1. Ouvrez Ciel Compta et connectez-vous à votre dossier.
2. Allez à **Fichier > Importer > Données externes** (ou **Outils > Assistant d'importation**).
3. Sélectionnez le fichier CSV téléchargé.
4. Configurez le mapping :
   - **Numéro facture** ← `numero_document`
   - **Date facture** ← `date_emission`
   - **Client** ← `client_nom` (il faut que le client existe dans Ciel)
   - **Montant HT** ← `montant_ht`
   - **TVA** ← `taux_tva`
   - **Montant TTC** ← `montant_ttc`
5. Vérifiez le type de pièce (Facture de vente) et validez l'import.

**Note** : Ciel ne gère pas les avoirs (avoir) de la même façon ; consultez la documentation Ciel pour les avoir ou les avoir de retour.

#### 6.1.2 Sage 50 Cloud (Windows/Mac)

1. Dans Sage, allez à **Accueil > Imports/Exports > Importer des données**.
2. Sélectionnez « Factures clients » ou « Documents de vente ».
3. Téléchargez le fichier CSV.
4. Faites correspondre les colonnes :
   - **Référence** ← `numero_document`
   - **Date** ← `date_emission`
   - **Tiers** ← `client_nom`
   - **Montant HT** ← `montant_ht`
   - **Montant TTC** ← `montant_ttc`
5. Validez et créez les pièces.

#### 6.1.3 Pennylane (Cloud, multi-plateforme)

1. Depuis votre tableau de bord Pennylane, allez à **Comptes > Importation de données** (ou équivalent).
2. Sélectionnez « Factures » ou « Clients ».
3. Chargez le fichier CSV Devis Generator.
4. Pennylane reconnaîtra automatiquement certaines colonnes (`numero_document`, `date_emission`, `montant_ttc`).
5. Vérifiez le mapping et lancez l'import.

**Conseil** : Pennylane propose aussi une API ; contactez-nous si vous souhaitez un connecteur direct.

#### 6.1.4 Excel ou Google Sheets (analyste)

1. Téléchargez le CSV depuis Devis Generator.
2. Ouvrez-le dans Excel (`Fichier > Ouvrir > documents.csv`) ou importez-le dans Google Sheets (`Fichier > Importer > Charger > documents.csv`).
3. Excel/Sheets reconnaîtra les colonnes et les montants (séparateur décimal en `.`).
4. Utilisez les filtres et graphiques pour analyser vos revenus par période, client, type de document.

**Pivot possible** : agrégez par `client_nom`, `date_emission` (par mois), `statut_paiement` pour des tableaux de synthèse.

### 6.2 Intégration API (pour développeurs)

L'export peut être **appelé programmatiquement** dans un flux d'automatisation :

#### 6.2.1 Synchronisation quotidienne

```bash
#!/bin/bash
# Script cron : exporte quotidiennement les documents payés du jour
DATE=$(date +%Y-%m-%d)
curl -X GET "https://self-hosted.example.com/api/v1/documents/accounting-export?start_date=${DATE}&end_date=${DATE}&payment_status=paye" \
  -H "Authorization: Bearer ${API_TOKEN}" \
  -o "/backups/devis-generator/export-${DATE}.csv"
```

#### 6.2.2 Webhook d'intégration (utiliser un service tiers)

Associez un service comme **Zapier** ou **Make** (anciennement Integromat) :
1. Créez un scénario qui déclenche « quotidiennement à 9h ».
2. Appellez l'endpoint `/api/v1/documents/accounting-export`.
3. Parsez le CSV et créez des factures dans votre logiciel compta via son API.

---

## 7. Fichier exemple

Voici un export type, avec 5 documents représentant différents cas :

```csv
numero_document;type_document;date_emission;client_nom;client_siret;montant_ht;taux_tva;montant_tva;montant_ttc;statut_paiement;date_paiement;mode_paiement;document_origine
F-2026-0001;facture;2026-01-15;Dupont SARL;12345678900012;1000.00;20.0;200.00;1200.00;paye;2026-01-20;virement;
FA-2026-0001;facture_acompte;2026-02-01;Martin Consulting;;500.00;0;0.00;500.00;paye;2026-02-03;stripe;D-2026-0003
F-2026-0002;facture;2026-02-10;Dupont SARL;12345678900012;750.00;20.0;150.00;900.00;en_attente;;
A-2026-0001;avoir;2026-02-15;Martin Consulting;;-100.00;20.0;-20.00;-120.00;paye;2026-02-15;virement;F-2026-0002
D-2026-0004;devis;2026-03-01;Leclerc & Co;;2000.00;5.5;110.00;2110.00;en_attente;;
```

**Explications** :

1. **F-2026-0001** : Facture standard, payée par virement le 2026-01-20.
2. **FA-2026-0001** : Facture d'acompte (TVA 0 car non facturée en soi), payée par Stripe, liée au devis D-2026-0003.
3. **F-2026-0002** : Facture non payée (en attente), pas de date de paiement.
4. **A-2026-0001** : Avoir (montants négatifs) émis le 2026-02-15, payé le même jour (cas d'un remboursement instantané).
5. **D-2026-0004** : Devis émis (numéroté), jamais converti, donc en attente — pas de paiement possible.

---

## 8. Évolutions futures

### 8.1 Hors V1

Selon la demande utilisateur et en fonction des retours, les évolutions suivantes sont envisagées :

- **Export FEC (Fichier des Écritures Comptables)** : un bouton distinct pour générer un vrai FEC XML/texte respectant la grammaire administrative française (ex. : modèle Autorité des Normes Comptables).
- **Connecteur Pennylane direct** : synchronisation automatique des factures via API Pennylane, sans passage par CSV.
- **Connecteur d'autres logiciels** (Sage, Ciel) : intégrations par API-push ou polling.
- **Paiements multi-linéaires** : détail des lignes dans l'export (actuellement agrégé par document).
- **Lettrage comptable** : ajout de colonnes pour le rapprochement manuel (ex. : hash unique stable par document).

### 8.2 Marche à suivre pour proposer une évolution

Si vous avez un besoin spécifique :
1. **Ouvrez une discussion** ou un formulaire de feedback sur le site.
2. **Décrivez votre use case** : quel logiciel compta, quelle fréquence, quel format attendu.
3. **Citez votre logiciel et version** pour validation.

Nous prioriserons les demandes les plus fréquentes et techniquement faisables.

---

## Résumé et bonnes pratiques

### Checklist pour votre intégration comptable

- [ ] **Téléchargez votre premier export** depuis le tableau de bord (menu Analyses > Export comptable).
- [ ] **Ouvrez le CSV dans Excel/Sheets** pour vérifier la lisibilité (taux de TVA, noms clients, montants).
- [ ] **Testez l'import dans votre logiciel compta** (Ciel, Sage, Pennylane) sur un dossier de test ou un brouillon.
- [ ] **Validez les montants** : HT, TVA, TTC (vérifiez que les colonnes correspondent).
- [ ] **Archivez les exports mensuels** ou trimestriels (ex. : `/backups/export-Q1-2026.csv`).
- [ ] **En cas d'erreur d'import**, consultez la documentation de votre logiciel ou contactez votre expert-comptable.
- [ ] **Pour une intégration continue (API)**, testez la requête curl ou le scénario Zapier avant la production.

### Limitations à retenir

- ✅ **C'est un export CSV simple** : importable partout, générique.
- ❌ **Ce n'est pas un FEC** : ne pas l'envoyer à l'administration fiscale.
- ❌ **Ce n'est pas une écriture comptable** : pas de journaux, pas de lettrage, pas de rapprochement automatique.
- ⚠️ **Un seul taux de TVA par document** : si vous aviez des lignes à taux mixte, elle sera lissée.
- ⚠️ **Mode de paiement optionnel** : importez les documents, rectifiez les modes manuellement si besoin.

---

## Support et questions

- **Problème d'import dans votre logiciel compta ?** → Consultez la doc du logiciel ou contactez son support.
- **Besoin d'un connecteur direct ou d'un FEC ?** → Ouvrez une issue ou écrivez-nous.
- **Données sensibles ou confidentielles ?** → Rappelez-vous que l'export CSV contient des données de clients et montants ; conservez-le de façon sécurisée.

---

**Dernière mise à jour** : novembre 2026 (V1 CSV simple).
