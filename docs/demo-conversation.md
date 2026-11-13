# Script de conversation démo — MCP Devis

Contenu validé produit (Démo-T1). À utiliser tel quel pour l'implémentation UI (Démo-T2).
Le lien final est le vrai lien généré par Démo-T3 (génération réelle via MCP sur tenant sandbox), jamais un lien mort ni une capture d'écran.

## Dialogue (FR)

1. **Utilisateur** : « Fais-moi un devis pour M. Dubois, rénovation complète d'une salle de bain, main d'œuvre + matériaux, environ 4500 € HT. »

2. **Agent** : « Avec plaisir ! Pour finaliser le devis, j'ai besoin de quelques précisions : l'email de M. Dubois pour l'envoi du devis, et comment souhaitez-vous décomposer les 4500 € HT — par exemple Main d'œuvre 2500 € / Matériaux 2000 € ? »

3. **Utilisateur** : « Oui exactement, son email c'est m.dubois@exemple.fr. Délai de validité : 30 jours. »

4. **Agent** : « Devis créé ! Voici le lien à lui envoyer : [LIEN RÉEL /d/{token}] »

## Points clés

- Le dialogue illustre la clarification intelligente du MCP : demande des infos manquantes, pas d'échec sec (cohérent avec Epic3-T4 déjà livré).
- Nom/email client fictifs mais réalistes (pas « Client Test 1 »).
- Le lien final doit être le vrai lien généré par Démo-T3 (génération réelle via MCP), jamais une capture d'écran statique ni un lien mort.
- Peut être décliné en anglais si la landing cible aussi un public anglophone (à trancher avec le reste de la landing existante).

## Données structurées pour l'UI (chat simulé)

- Client : M. Dubois — email m.dubois@exemple.fr
- Objet : rénovation complète d'une salle de bain
- Lignes : Main d'œuvre 2500 € HT / Matériaux 2000 € HT (total 4500 € HT)
- Validité : 30 jours
- Lien final : `/d/{token}` (token réel issu de Démo-T3)
