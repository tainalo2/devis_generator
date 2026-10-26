Outil gratuit de devis et factures pour micro-entrepreneurs, indépendants et petites sociétés. L'interface est calme et précise, comme une feuille bien tenue : une encre ardoise, un accent bleu-vert, des titres en Fraunces et le texte en Geist. Desktop d'abord, mais pleinement utilisable sur mobile.

## Contenu et ton

- **Français uniquement**, y compris les libellés techniques visibles (« Brouillon », « À encaisser », « Mentions légales »).
- **Vouvoiement** et phrases directes : « Créez un document en mode invité, ou ouvrez un compte pour les conserver. »
- **Le nom exact** est `Devis Generator : VIBED`, avec l'espace avant les deux-points et VIBED en capitales. Ne pas l'abréger dans les titres.
- **Promesse** : « Des devis et factures clairs, conformes, sans friction. » On parle de conformité, de TVA et de sources officielles avec des mots concrets, sans jargon marketing.
- **Boutons** : un verbe et son objet (« Créer un devis sans compte », « Enregistrer », « Supprimer »).
- **Pas d'emoji.** Les statuts sont des mots (« Payée », « Brouillon ») portés par un `Badge`.
- **Casse de phrase** partout. Seuls les libellés de statistiques passent en capitales, avec le style `overline`.
- **Vocabulaire métier** : devis, facture, facture d'acompte, avoir, émis, brouillon. Un document émis est immuable et se corrige par un avoir : ne jamais proposer « Modifier » sur un document émis.

## Couleur

- **Deux thèmes**, `light` (Clair) et `dark` (Sombre). L'app suit le thème système par défaut, avec un bouton de bascule. Chaque token a une valeur par thème : toujours passer par le token, jamais par une couleur en dur.
- **Surfaces** : `background` pour la page, `card` pour les panneaux, `popover` pour les menus et toasts, `muted` pour les zones discrètes. La barre latérale a sa propre série `sidebar-*`.
- **Texte** : `foreground` pour le texte courant, `muted-foreground` pour le secondaire (descriptions, aides, dates). Les deux dépassent 5.5:1 sur toutes les surfaces, dans les deux thèmes.
- **Action** : `primary` porte l'action principale. C'est l'encre ardoise en clair, qui devient un bleu-vert pâle en sombre (le texte `primary-foreground` s'inverse avec lui).
- **Marque** : `brand` (bleu-vert) sert aux succès, au spinner, aux halos décoratifs et au `Badge` `success`. On l'emploie surtout en teinte (15–20 % d'opacité) ; un aplat `brand` prend `brand-foreground`.
- **Ambiance** : `accent` est le bleu-vert très pâle des halos de fond et des surlignages.
- **Erreur** : `destructive` s'utilise en texte sur un fond `destructive` à 10 % (20 % en sombre), jamais en aplat plein.
- **Graphiques** : `chart-1` à `chart-5`, dans cet ordre.
- **Bordures** : `border` pour les filets et cartes, `input` pour les champs. En clair, `input` reste à 1.29:1, sous le seuil de 3:1. C'est la valeur du dépôt, conservée : ne pas s'en servir seule pour signaler un état.
- **À éviter** : violets, crème/terracotta, dégradés bleu-violet. Le fond de page porte deux halos radiaux très légers (`brand` à 12 %, `accent` à 55 %) : c'est le seul dégradé de l'interface.

## Typographie

- **Titres** (h1–h3) en Fraunces (`--font-heading`), avec l'axe d'optique automatique et un interlettrage serré (−0.025em) :
  - `display` : nom du produit sur la page d'accueil
  - `page-title` : h1 de chaque page
  - `section-title` : h2
  - `card-title` : titres de carte et d'état vide
  - `wordmark` : nom dans la barre latérale
- **Texte** en Geist (`--font-sans`). `body-sm` (14/20) est le texte par défaut de l'application. `body` (16/24) est réservé à la page d'accueil, et `prose-sm` (interligne 1.625) aux pages légales.
- **Libellés** : `label` (14 px, 500) pour les champs et boutons, `caption` pour les mentions, `badge` pour les statuts.
- **Code** en monospace système (`--font-mono`), style `code`, pour les variables d'environnement et clés techniques, sur `muted` à 50 %.
- Largeur de lecture : `max-w-xl` à `max-w-lg` pour les accroches, `max-w-md` pour les descriptions d'état vide.

## Espacement, rayons, ombres

- **Grille Tailwind de 4 px.**
  - Gouttière de page : `space-4` sur mobile, `space-6` à partir de md, 32 px en lg sur la page d'accueil.
  - Contenu principal : `space-8` de padding vertical.
  - Sections de la page d'accueil : `space-16`.
- **Largeurs** : contenu de l'app plafonné à 64rem (`max-w-5xl`), page d'accueil à 72rem (`max-w-6xl`).
- **Rayons** dérivés de `radius` (0.5rem) :
  - `radius-md` : champs et badges
  - `radius-lg` : boutons et toasts
  - `radius-xl` : cartes et états vides (bordure en pointillés)
  - `radius-2xl` : grands panneaux de l'accueil
- **Ombres** rares : les cartes se distinguent par leur bordure `border` et un fond `card` translucide, pas par l'ombre. `shadow-lg` est réservé aux toasts, `shadow-float` à la carte flottante de l'accueil.

## États et mouvement

- **Focus** : anneau `ring` de 2 px décalé de 2 px sur `background` (3 px à 50 % sur les boutons). `ring` atteint 4.49:1 en clair et 6:1 en sombre.
- **Désactivé** : opacité 50 %, sans interaction.
- **Pression** : le bouton descend de 1 px.
- **Animations** :
  - `fade-up` : 0.55 s ease-out, 12 px vers le haut, pour le héros et les toasts
  - `fade-in` : 0.45 s
  - `soft-pulse` : 3.5 s, pour un halo décoratif
  - transitions de 150 ms
- **Thème** : rien n'est animé au changement de thème (`disableTransitionOnChange`).

## Iconographie

- **Lucide** (`lucide-react`), trait de 2 px, 16 px dans les boutons (12–14 px dans les petites tailles), couleur héritée du texte.
- Le logo (`assets/Logos/devis-generator-mark.svg`) représente une feuille de devis avec une coche dans une pastille bleu-vert. Il sert de favicon et d'icône d'application ; ne pas le redessiner.
- Pas d'emoji ni d'illustrations figuratives. La page d'accueil illustre le produit par une carte abstraite : barres de texte arrondies et bouton `primary`.

## Composants

`Button`, `Badge`, `Input`, `Label`, `Separator`, `Spinner` : les composants de `frontend/src/components/ui/`. Le global est `window.DevisUI`, avec les mêmes variantes et tailles que le code. Une ligne de formulaire, c'est un `Label` qui enveloppe un texte et un `Input`. Une zone ne porte qu'un seul `Button` `default`.

## Non synchronisé

- **Composants** : réécrits à la main en CSS et React à partir des classes exactes du dépôt, sans exécuter son build. Dans l'app, `Button` et `Separator` reposent sur Base UI ; ici, ce sont un `<button>` et un `<div role="separator">`.
- **Composants d'application** non repris : `EmptyState`, `AppShell`, `DocumentForm`, `PdfPreview`, `VatValidator`, `CompanyAutocomplete`, toasts.
- **Espacement** : le dépôt ne déclare pas d'échelle. Les tokens d'espacement sont les pas Tailwind que les composants utilisent.
- **Polices** : seul le sous-ensemble latin des polices variables (fontsource 5.3.0) est inclus, sans latin-ext ni vietnamien.
- **Icônes** : le sprite `frontend/public/icons.svg` (logos Bluesky, Discord, GitHub, X…) n'est pas repris, car ce sont des marques tierces. Les images `src/image/` appartiennent au site historique.
- **Hors tokens** : l'ambre du `Badge` `warning` vient de la palette Tailwind (`amber-500` / `800` / `200`).
