> Je cherche du travail ! Freelance ou CDI en dev front/back -> www.alexandre-rongier.fr
> [Devis generator le site officiel ici](https://www.purpletwit.com/devisgenerator)
# Devis Generator ![Static Badge](https://img.shields.io/badge/Licencse-GPL_V3-red) ![Static Badge](https://img.shields.io/badge/__-vanilla-yellow?style=flat&logo=javascript) ![Static Badge](https://img.shields.io/badge/__-Node--Red-red?style=flat&logo=nodered&logoColor=%23ff0505)

Devis Generator est un générateur de facture/devis pour les micros-entrepreneurs français, en format web.


<p align="center">
  <img width="350" height="335" src="https://raw.githubusercontent.com/tainalo2/devis_generator/main/image_for_readme_1.png">
  <img width="350" height="335" src="https://raw.githubusercontent.com/tainalo2/devis_generator/main/image_for_readme_2.png">
  <img width="350" height="335" src="https://raw.githubusercontent.com/tainalo2/devis_generator/main/image_for_readme_3.png">
</p>

## Problématique
Il n'existe pas d'outils simple et concis permettant d'éditer une facture ou un devis en PDF sans oublier toutes les obligations légales. Le statut de micro-entrepreneur permettant notamment d'encadré des activités additionnelles au salariat, censé être administrativement simplifié pour ne pas être un frein à son encadrement légale, je trouvais ça lourd d'éditer ma facture à chaque fois dans un vieux modèle google sheet.

Au Pays-bas il est courant de monétiser son hobby et d'en faire un complément de revenu. En France le statut de micro-entrepreneur le permet également mais ne donne pas les outils nécessaires.

## Solution
Une plateforme gratuite, accessible à tous, et qui va droit au but pour économiser le temps de l'utilisateur.

Un développement open-source pour assurer la gratuité, un format web pour être compatible à tous les supports, une UI/UX minimaliste pour ne pas décourager l'utilisateur.

La solution est donc un site web, l'officiel est hébergé ici : https://www.purpletwit.com/devisgenerator

Le code est en HTML/CSS/JS vanilla pour simplifier le déploiement et l'auto-hébergement.

## Fonctionalités

Devis generator permet de :
* Générer un PDF de facture/devis
* Respecter les lois en vigueurs pour les micro-entrepreneurs
* Enregistrer des templates de prestataire/client/devis (bientôt) (seulement avec une base SQL)
* Ajouter une signature "manuscrite"
* Pré-calculer la TVA
* Ajout d'un bouton cliquable pour le paiement en ligne

## Roadmap
* Ajout de l'hébergement des templates
* Système de double chiffrement de la base SQL
* Support des ventes commerciale (TVA et déclarations différentes)

## Dépendances
Devis generator base certaines fonctionalités sur des librairies JS :
* Système de signature : [Signature Pad](https://github.com/szimek/signature_pad?tab=readme-ov-file)
* Génération du PDF : [jsPDF](https://github.com/parallax/jsPDF)
* Transformation d'éléments en image pour le PDF + jsPDF : [html2canvas](https://github.com/niklasvh/html2canvas)

## Participer au projet

Les contributions sont les bienvenues, de même pour les feature requests et issues.

Ce projet n'est pas financé, je ne m'engage pas sur la rapidité des réponses.

## License
Ce projet est sous license [GNU-GPL-V3](https://www.gnu.org/licenses/gpl-3.0), son exploitation est autorisé uniquement dans un cadre non-commercial !
