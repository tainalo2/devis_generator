# Devis Generator, pour les humains

Devis Generator est un outil gratuit pour les micro-entrepreneurs, les indépendants et les petites sociétés en France. On y prépare un devis ou une facture, on le relit, on le télécharge en PDF.

L’objectif du projet est plus précis : **une plateforme gratuite pour signer les factures en France.** Signer, ici, veut dire apposer une signature électronique sur la facture, au nom de celui qui l’émet, pour que le fichier lui soit attribuable et qu’on puisse voir s’il a été modifié ensuite.

## Ce que vous pouvez faire aujourd’hui

- Créer un devis, une facture, une facture d’acompte ou un avoir.
- Travailler sans compte : le PDF est généré et n’est pas conservé sur le serveur.
- Ouvrir un compte pour garder les clients, les brouillons et l’historique, puis exporter ou supprimer ces données.
- Choisir la franchise de TVA (article 293 B du CGI) ou un taux par ligne : 0 %, 2,1 %, 5,5 %, 10 %, 20 %.
- Obtenir un PDF lisible, avec les mentions figées au moment de l’émission.

L’application en service est [devisgenerator.fr](https://www.devisgenerator.fr/).

## Ce que « signer une facture » ajoute

Une facture signée est le même document, plus une preuve liée au fichier :

1. La facture contient les données utiles sous une forme lisible par un logiciel (le format Factur-X), en plus du PDF qu’on lit.
2. La signature électronique identifie l’émetteur et protège l’intégrité du fichier.
3. L’envoi vers le client et vers l’administration passe par le circuit prévu par la réforme de la facturation électronique.

Aujourd’hui, l’étape 1 est commencée : le projet sait préparer le fichier de données Factur-X (profil européen EN 16931). Ce fichier n’est pas encore glissé dans le PDF téléchargé. Les étapes 2 et 3 restent à construire. Le site de l’application le dit encore : Factur-X n’est pas inclus dans le téléchargement.

Devenir la plateforme qui transmet elle-même les factures à l’administration est une démarche à part, l’immatriculation comme plateforme agréée auprès de la DGFiP. Tant que cette immatriculation n’existe pas, l’envoi pourra passer par une plateforme déjà immatriculée. Les deux chemins mènent au même service pour l’utilisateur : préparer la facture, la signer, la faire partir.

## Calendrier utile

Le projet retient le calendrier publié par la DGFiP : toutes les entreprises assujetties reçoivent des factures électroniques à compter du 1er septembre 2026 ; les grandes entreprises et les entreprises de taille intermédiaire les émettent à la même date ; les PME, TPE et micro-entrepreneurs les émettent au plus tard le 1er septembre 2027.

## Ce qui ne change pas

Le service reste gratuit. Il n’encaisse pas de paiement pour le compte de l’utilisateur. Un lien de don peut être affiché, sans prestataire de paiement intégré. Les factures déjà émises restent immuables : une erreur se corrige par un avoir.

## Pour aller plus loin

- Le fonctionnement du logiciel : [technique.md](technique.md).
- Le détail du fichier Factur-X déjà produit : [factur-x.md](factur-x.md).
- Les décisions déjà tranchées : [DECISIONS.md](DECISIONS.md).
