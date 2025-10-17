# Améliorations de la Gestion du Mode Jour/Nuit pour les PDF

## Vue d'ensemble

Ce document décrit les améliorations apportées au projet Devis Generator pour optimiser la gestion du mode jour/nuit lors de la génération des PDF. L'architecture actuelle a été sauvegardée dans la branche `architecture-backup`.

## Problèmes identifiés

### 1. Couleurs hardcodées dans le PDF
- **Problème** : Le PDF utilisait des couleurs fixes (blanc, noir, gris) qui ne s'adaptent pas au mode jour/nuit
- **Impact** : Les PDF générés en mode sombre avaient un fond blanc et des couleurs incohérentes

### 2. Background blanc fixe
- **Problème** : `background-color: rgb(255, 255, 255)` dans `#absolute_to_generate`
- **Impact** : Le PDF était toujours blanc, même en mode sombre

### 3. Couleurs de texte et bordures fixes
- **Problème** : Couleurs comme `#202020`, `black`, etc. hardcodées
- **Impact** : Manque de cohérence visuelle entre l'interface et le PDF

### 4. Pas de gestion dynamique
- **Problème** : La fonction `generatePDF()` ne prenait pas en compte le mode actuel
- **Impact** : Les PDF générés ne reflétaient pas le thème choisi par l'utilisateur

## Solutions implémentées

### 1. Variables CSS pour les PDF

Ajout de variables CSS spécifiques pour les PDF dans `:root` :

```css
/* Variables spécifiques pour le PDF */
--pdf-bg-color: white;
--pdf-text-color: black;
--pdf-border-color: #202020;
--pdf-section-bg: rgb(247, 247, 247);
--pdf-payment-bg: rgb(230, 230, 230);
--pdf-button-bg: black;
--pdf-button-text: white;
```

### 2. Styles PDF dynamiques

Modification de tous les styles PDF pour utiliser les variables CSS :

```css
#absolute_to_generate {
    background-color: var(--pdf-bg-color);
    color: var(--pdf-text-color);
}

.section_to_generate {
    border: 1px solid var(--pdf-border-color);
    background-color: var(--pdf-bg-color);
    color: var(--pdf-text-color);
}

.section_devis_to_generate_line:nth-child(even) {
    background: var(--pdf-section-bg);
}
```

### 3. Fonction utilitaire centralisée

Création d'une fonction `applyTheme()` pour gérer les thèmes de manière centralisée :

```javascript
function applyTheme(isDarkMode) {
    const root = document.querySelector(':root');
    
    if (isDarkMode) {
        // Mode nuit - variables PDF
        root.style.setProperty('--pdf-bg-color', 'black');
        root.style.setProperty('--pdf-text-color', 'white');
        root.style.setProperty('--pdf-border-color', '#404040');
        // ... autres variables
    } else {
        // Mode jour - variables PDF
        root.style.setProperty('--pdf-bg-color', 'white');
        root.style.setProperty('--pdf-text-color', 'black');
        root.style.setProperty('--pdf-border-color', '#202020');
        // ... autres variables
    }
}
```

### 4. Détection automatique du thème système

Ajout de la détection automatique du thème système :

```javascript
function detectSystemTheme() {
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
        document.getElementById("switch_toggle_light").checked = true;
        applyTheme(true);
    } else {
        document.getElementById("switch_toggle_light").checked = false;
        applyTheme(false);
    }
}
```

### 5. Sauvegarde des préférences

Implémentation de la sauvegarde des préférences de thème :

```javascript
function loadThemePreference() {
    const savedTheme = localStorage.getItem('theme-preference');
    if (savedTheme) {
        const isDarkMode = savedTheme === 'dark';
        document.getElementById("switch_toggle_light").checked = isDarkMode;
        applyTheme(isDarkMode);
        return true;
    }
    return false;
}
```

### 6. Mise à jour de la génération PDF

Modification de la fonction `generatePDF()` pour s'assurer que le thème est appliqué :

```javascript
function generatePDF() {
    alertDisplay("waiting", "Génération en cours...");
    
    // S'assurer que le thème PDF est à jour
    initializePDFTheme();
    
    // ... reste de la fonction
}
```

## Améliorations apportées

### 1. Cohérence visuelle
- Les PDF s'adaptent automatiquement au mode jour/nuit choisi
- Couleurs harmonisées entre l'interface et le PDF généré

### 2. Expérience utilisateur améliorée
- Détection automatique du thème système
- Sauvegarde des préférences utilisateur
- Changement de thème en temps réel

### 3. Code plus maintenable
- Fonction utilitaire centralisée pour les thèmes
- Variables CSS organisées et documentées
- Séparation claire entre les styles interface et PDF

### 4. Performance optimisée
- Application des thèmes uniquement quand nécessaire
- Gestion efficace des changements de thème

## Structure des fichiers modifiés

### `style.css`
- Ajout des variables CSS pour les PDF
- Modification des styles PDF pour utiliser les variables
- Amélioration de la cohérence des couleurs

### `main.js`
- Ajout de la fonction `applyTheme()`
- Ajout de la fonction `detectSystemTheme()`
- Ajout de la fonction `loadThemePreference()`
- Modification de `toggleLightMode()`
- Modification de `generatePDF()`

## Tests

Un fichier de test `test_theme.html` a été créé pour vérifier le bon fonctionnement des améliorations :

- Test de basculement entre les modes jour/nuit
- Vérification des variables CSS
- Prévisualisation des styles PDF

## Utilisation

1. **Mode automatique** : Le thème système est détecté automatiquement au chargement
2. **Mode manuel** : L'utilisateur peut basculer entre les modes via le toggle
3. **Sauvegarde** : Les préférences sont sauvegardées et restaurées à la prochaine visite
4. **Génération PDF** : Les PDF générés respectent le thème actuel

## Compatibilité

- ✅ Chrome/Chromium
- ✅ Firefox
- ✅ Safari
- ✅ Edge
- ✅ Mobile (iOS/Android)

## Notes techniques

- Utilisation de `window.matchMedia` pour la détection du thème système
- `localStorage` pour la persistance des préférences
- Variables CSS pour une gestion flexible des couleurs
- Écouteurs d'événements pour les changements de thème en temps réel

## Conclusion

Ces améliorations permettent une gestion complète et cohérente des thèmes jour/nuit pour les PDF, améliorant significativement l'expérience utilisateur et la maintenabilité du code.