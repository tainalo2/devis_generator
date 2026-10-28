# Factur-X — profil EN 16931

Le paquet `backend/internal/facturx` prépare le XML CII d'une facture au profil EN 16931. `Build` renvoie ce XML. Il ne modifie pas le PDF téléchargé : gofpdf ne produit pas un PDF/A-3, donc ce fichier n'est pas encore une facture Factur-X.

Les textes du site qui indiquent que Factur-X n'est pas inclus restent exacts.

## Ce que le XML contient

Guideline : `urn:cen.eu:en16931:2017#compliant#urn:factur-x.eu:1p0:en16931`.

| Donnée | Règle |
|---|---|
| Type | Facture `380`, facture d'acompte `386`, avoir `381`. Un devis est refusé. |
| Numéro, date, devise | Numéro et date d'émission obligatoires. Devise du document, sinon `EUR`. Date au format `102` (`AAAAMMJJ`). |
| Parties | Nom, adresse, SIRET (`GlobalID`, schéma `0009`), SIREN (schéma `0002`, déduit des 9 premiers chiffres du SIRET s'il manque), n° de TVA (schéma `VA`). |
| Pays | Texte libre normalisé (`France`, vide, `Belgique`…). Défaut `FR`. |
| Lignes | Désignation, quantité (milli-unités), prix unitaire HT, remise unitaire, taux. Unité absente du modèle : `C62`. |
| Totaux | HT, TVA et TTC repris des centimes déjà calculés. Ils doivent coïncider avec `billing.Compute`. |
| TVA | Régime standard : catégorie `S`, ou `Z` si le taux est 0. Franchise 293 B : catégorie `E`, taux 0, motif `legal.DefaultMentions().Franchise293B`. |
| Paiement | Conditions et échéance du document. IBAN de l'émetteur, s'il est renseigné, en virement (`30`). |

Un taux normal (`S`) exige le n° de TVA du vendeur. La franchise et le taux 0 (`Z`) non.

## Hors de ce socle

- Adresse électronique de facturation (réforme 2027).
- Références de commande, de contrat ou d'acheteur.
- N° de la facture d'origine sur un avoir : le document ne porte qu'un identifiant interne (`RelatedDocumentID`), pas le numéro affiché.
- CIUS français XP Z12-012.
- Validation Schematron du schéma Factur-X officiel (les tests vérifient la structure et les montants, sans appel réseau).
- Ville et code postal : émis quand ils sont saisis, sans rejet s'ils manquent.
- Embarquement dans un PDF/A-3 (`factur-x.xml` + métadonnées XMP). C'est l'étape suivante ; le PDF actuel reste un PDF simple.
