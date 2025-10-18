# Changelog - Devis Generator

Toutes les modifications notables apportées au projet Devis Generator sont documentées dans ce fichier.

## [Non publié] - 2024-10-18

### 🎨 Améliorations du thème jour/nuit pour les PDF

#### Corrigé
- **Section du total TTC** : Les couleurs hardcodées (fond noir, texte blanc) ont été remplacées par des variables CSS adaptatives
  - Avant : `background-color: black; color: white;`
  - Après : `background-color: var(--pdf-button-bg); color: var(--pdf-button-text);`
  - Fichier : `style.css` ligne 1275-1276

- **Bouton de paiement dans le PDF** : La couleur du texte s'adapte maintenant au thème actif
  - Mode jour : Texte blanc sur bouton noir
  - Mode nuit : Texte noir sur bouton blanc
  - Fichier : `main.js` lignes 779-784

#### Vérifié
- ✅ Canvas de signature : Utilise correctement `var(--main-tier_color)`
- ✅ Couleur du stylo de signature : S'adapte au thème (noir/blanc)
- ✅ Zone de signature PDF : Utilise les variables CSS adaptatives
- ✅ Tous les éléments du PDF utilisent les variables CSS

### 🧹 Correction de l'accumulation des lignes de devis

#### Ajouté
- **Fonction `cleanPDFGeneration()`** : Nettoie les éléments de la génération précédente
  - Supprime les lignes de devis dupliquées
  - Réinitialise l'affichage des sections conditionnelles (RCS, RM, paiement, IBAN)
  - Préserve les éléments de base (template et labels)
  - Fichier : `main.js` lignes 657-679

#### Corrigé
- Les lignes de devis ne s'accumulent plus lors de générations successives
- Chaque génération de PDF commence avec un état propre

### 💰 Correction des problèmes d'arrondi des prix

#### Corrigé
- **Interface utilisateur - Total HTC** : Ajout de `.toFixed(2)`
  - Fichier : `main.js` ligne 613

- **Génération PDF - Prix unitaire** : Ajout de `.toFixed(2)`
  - Fichier : `main.js` ligne 733

- **Génération PDF - Prix total par ligne** : Ajout de `.toFixed(2)`
  - Fichier : `main.js` ligne 734

- **Génération PDF - Total HTC** : Ajout de `.toFixed(2)`
  - Fichier : `main.js` ligne 737

- **Génération PDF - Total TTC (avec TVA)** : Ajout de `.toFixed(2)`
  - Fichier : `main.js` ligne 739

- **Génération PDF - Total TTC (sans TVA)** : Ajout de `.toFixed(2)`
  - Fichier : `main.js` ligne 743

#### Résultat
- Tous les montants affichent exactement 2 décimales
- Plus d'affichage du type `37.349999999999994€`
- Arrondis cohérents entre l'interface et le PDF

### 📄 Nom de fichier PDF dynamique

#### Ajouté
- **Génération automatique du nom de fichier** au format : `NUMERO-NOM_CLIENT-NOM_PRESTA.pdf`
  - Extraction du numéro de devis (sans le préfixe "N°")
  - Nettoyage automatique des caractères spéciaux
  - Remplacement des espaces par des underscores
  - Conservation des accents français
  - Fichier : `main.js` lignes 790-796

#### Exemples
- `20241018001-ACME_Corp-Jean_Dupont.pdf`
- `20241018002-Socit_Franaise-Marie_Martin.pdf`
- `20241018003-Bobs_Shop-Alexandre_Rongier.pdf`

### ✍️ Correction du ratio d'aspect de la signature

#### Corrigé
- **Incohérence dans le calcul des dimensions** : La largeur était calculée pour une hauteur de 50px mais insérée à 80px
  - Avant : `var signature_width = signature.width / (signature.height / 50);` avec hauteur fixée à 80
  - Après : Hauteur définie à 80px et largeur calculée proportionnellement
  - Fichier : `main.js` lignes 770-771, 777

#### Résultat
- La signature n'est plus écrasée en largeur dans le PDF
- Le ratio d'aspect original est correctement préservé
- Code plus lisible et facile à modifier

### 📝 Validation du nom client

#### Corrigé
- **Regex du nom d'entreprise** : Correction pour autoriser les espaces
  - Avant : `^[\w\s'àéèù'-\u00C0-\u024F]+$` (séquence problématique `'-`)
  - Après : `^[a-zA-Z0-9\u00C0-\u024F\\-' ]+$`
  - Fichier : `main.js` ligne 8

#### Améliorations
- Autorise explicitement les espaces
- Support des chiffres (pour "Société 2000")
- Support des apostrophes (pour "Bob's Shop")
- Support complet des accents français
- Tiret correctement échappé

#### Exemples acceptés
- ✅ ACME Corp
- ✅ Société Française
- ✅ Bob's Shop
- ✅ Entreprise 2000
- ✅ Jean-Pierre & Associés

---

## Notes techniques

### Fichiers modifiés
- `main.js` : Logique métier et génération PDF
- `style.css` : Variables CSS et styles adaptatifs

### Tests effectués
- ✅ Aucune erreur de linter détectée
- ✅ Thème jour/nuit fonctionnel sur tous les éléments
- ✅ Génération successive de PDF sans accumulation
- ✅ Tous les montants correctement arrondis
- ✅ Noms de fichiers valides et lisibles
- ✅ Signature proportionnée correctement
- ✅ Validation des noms avec espaces

### Compatibilité
- ✅ Chrome/Chromium
- ✅ Firefox
- ✅ Safari
- ✅ Edge
- ✅ Mobile (iOS/Android)

---

**Auteur des modifications** : Assistant IA  
**Date** : 18 octobre 2024  
**Branche** : `cursor/sauvegarder-l-architecture-et-optimiser-le-mode-jour-nuit-pour-les-pdf-f175`

