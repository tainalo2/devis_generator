# Tableau de bord analytics — métriques disponibles (self-hosters)

Cette page documente les métriques exposées par le tableau de bord analytics de l'instance, leur interprétation et leurs limites connues. Les métriques sont calculées à partir des documents (devis, factures) et des paiements enregistrés pour le compte du propriétaire.

## 1. Métriques disponibles

### 1.1 Taux de conversion (devis → facture)

- **Définition** : proportion des devis émis qui ont été convertis en facture (ou acceptés/signés), sur la période sélectionnée.
- **Formule** : `nombre de devis convertis / nombre de devis émis × 100`.
- **Interprétation** : un taux élevé indique une bonne adéquation entre l'offre et la demande ; un taux bas peut signaler des prix trop élevés, un ciblage inadapté ou des délais de réponse trop longs.
- **Limites** :
  - Un devis converti manuellement puis annulé peut fausser le comptage si le statut n'est pas mis à jour.
  - Les devis expirés sans réponse sont comptés comme non convertis.

### 1.2 Délai de paiement (jours)

- **Définition** : nombre de jours entre l'émission (ou la date d'échéance) d'une facture et la date à laquelle elle est marquée comme payée.
- **Formule** : `date de paiement − date d'émission` (en jours).
- **Interprétation** : permet de suivre la rapidité d'encaissement et d'anticiper les besoins de trésorerie. Une valeur élevée ou croissante indique des retards de paiement récurrents.
- **Limites connues** :
  - **Mode RIB/IBAN** : le délai de paiement n'est **pas calculable automatiquement**. Le virement bancaire n'étant pas tracé par l'instance, la facture doit être marquée manuellement comme « payée » (date de paiement saisie par le propriétaire) pour que la métrique soit renseignée. Sans ce marquage, la facture reste considérée comme impayée et n'entre pas dans le calcul.
  - Les paiements partiels ne sont pas pris en compte comme paiement complet tant que le statut n'est pas « payé ».
  - La date de paiement saisie manuellement dépend de la rigueur de l'utilisateur ; une saisie tardive gonfle artificiellement le délai.

### 1.3 Chiffre d'affaires (CA)

- **Définition** : somme des montants des factures marquées comme payées sur la période sélectionnée (hors taxes ou TTC selon le réglage).
- **Interprétation** : indicateur principal d'activité. À comparer avec les périodes précédentes et avec le CA prévisionnel (devis en cours).
- **Limites** :
  - Le CA ne reflète que les factures dont le statut est « payé » ; les factures émises non payées ne sont pas comptabilisées.
  - En mode RIB/IBAN, le CA dépend du marquage manuel du statut payé (voir 1.2).
  - Les avoirs/annulations ne sont déduits que si le document correspondant est correctement annulé.

## 2. Filtrage par période

Les métriques sont filtrables par période (voir [Analytics-T4]) : jour, semaine, mois, trimestre, année ou plage personnalisée. Le filtrage s'applique à la date d'émission du document (et, pour le délai de paiement, à la date de paiement).

## 3. Export

Les données agrégées peuvent être exportées en CSV (voir [Analytics-T5]) pour exploitation externe (tableur, BI). L'export reprend les mêmes métriques et filtres que l'écran.

## 4. Performance et mise en cache

Les agrégations sont calculées côté backend et mises en cache pour limiter la charge (voir [Analytics-T6]). En cas de forte volumétrie, les valeurs peuvent être légèrement décalées par rapport aux dernières modifications (rafraîchissement périodique).

## 5. Bonnes pratiques

- Marquer les factures comme payées dès réception du paiement, surtout en mode RIB/IBAN, pour garder des métriques fiables.
- Vérifier régulièrement la cohérence entre le CA affiché et la comptabilité.
- Interpréter le délai de paiement en tenant compte du délai contractuel (ex. 30 jours) : un délai moyen inférieur à l'échéance est sain.
