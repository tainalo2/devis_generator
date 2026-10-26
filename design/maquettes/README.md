# Maquettes Devis Generator : VIBED

Maquettes de l'interface, réalisées dans un canvas Design à partir du design system
synchronisé depuis cette branche (`ds/devisui/`). Chaque fichier `.dc.html` est un
artboard ; `canvas.json` décrit leur disposition et les pages du canvas.

## Pages du canvas

| Page | Fichiers principaux |
|---|---|
| Nav | `Main`, `Invite`, `Sombre`, `Mobile`, `NavMobileInvite`, variantes `Alt*` (B retenue) |
| Landing | `Artboard-pjyc` (page assemblée), `Landing*`, `SiteFooter`, `LandingMobile` |
| Editor | zonage (`Artboard-frvg`, `EditorMobile`), maquettes `EditorBureau` / `EditorFacture` / `EditorAcompte` / `EditorMobileMaquette`, composants `Editor*`, modales `EditorModalClient` et `EditorModalEmetteur` |
| Dashboard | zonage `Artboard-h81v`, `MWfDashboard` ; maquette `ScrDashboard` (+ `M` pour mobile) |
| Autres pages | zonage `Wf*` (bureau) et `MWf*` (mobile) ; maquettes `Scr*` (bureau) et `Scr*M` (mobile) |

## Conventions

- Les écrans `Scr*` s'adaptent à la largeur : la version mobile `Scr*M` importe le même
  fichier à 390 px avec `mobile` activé.
- Les composants de l'éditeur acceptent `kind` (`devis`, `facture`, `acompte`) et
  `compact` (mobile).
- Les données d'exemple (clients, montants) sont fictives ; les identités restent
  entre crochets, par exemple `[Votre raison sociale]`.

Ces fichiers sont des maquettes, pas du code de l'application.
