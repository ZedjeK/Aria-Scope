<p align="center">
  <img src="docs/logo.png" width="160" alt="Logo d'Aria Scope : un A néon glitché avec des jumelles orange">
</p>

<h1 align="center">Aria Scope</h1>

<p align="center">
  L'œil qui fouille ton disque : Aria Scope te montre, dossier par dossier,<br>
  qui prend toute la place. Scan parallèle, arborescence triée, un seul exe Windows.<br>
  Rien à installer, aucun droit administrateur nécessaire.
</p>

<p align="center">
  <a href="../../releases/latest"><b>Télécharger la dernière version</b></a> · v0.4 · Windows 10/11 x64 · licence MIT
</p>

## Ce qu'il fait

| Fonction | Ce que ça fait |
| --- | --- |
| **Tout le PC, dès l'ouverture** | À son lancement, Aria Scope analyse tous tes disques internes et externes en même temps (pas les lecteurs réseau ni optiques) et les réunit sous « Ce PC ». Désactivable d'un clic sur la page des disques |
| **Disques** | Tous tes lecteurs avec leur remplissage, pour analyser un seul disque. Ou un dossier précis, si tu sais déjà où chercher |
| **Scan parallèle** | Jusqu'à 32 dossiers lus en même temps, une seule requête par dossier pour obtenir toutes les tailles. Progression en direct, et un bouton pour arrêter quand tu veux |
| **Arborescence triée** | Chaque dossier, du plus lourd au plus léger : taille, part de son parent, nombre de fichiers et de sous-dossiers, date du fichier le plus récent. Un clic sur un en-tête trie par n'importe quelle colonne |
| **Treemap néon** | Chaque rectangle a la surface de ce qu'il pèse, une couleur par grande branche. Survol pour le détail, double-clic pour plonger dans un dossier, fil d'Ariane pour remonter |
| **Anneau** | La répartition du dossier sélectionné entre ses 8 plus gros éléments ; un clic sur une part plonge dedans |
| **Tout est relié** | Ce que tu sélectionnes dans l'arbre s'allume sur la carte et l'anneau, et l'inverse : l'arbre se déplie tout seul jusqu'à la bonne ligne |
| **Plus gros fichiers** | Les 1 000 plus gros fichiers de l'analyse, relevés pendant le scan, triables, avec leur dossier ; un clic les retrouve dans l'arbre et sur la carte |
| **Types et âge** | La répartition par famille (vidéos, archives, jeux, impression 3D…) et par extension, puis par date de modification, pour voir ce qui dort depuis des années. Pour toute l'analyse ou un seul dossier |
| **Filtres** | Exclure des dossiers ou des fichiers (`node_modules`, `*.tmp`), ne garder que certains types (`*.mkv`), une regex, une taille ou une ancienneté minimales. Appliqués pendant le scan, donc tout reste cohérent ; un bandeau rappelle qu'une analyse est filtrée |
| **Deux tailles** | La taille réelle des données, ou la place occupée sur le disque (clusters, compression, fichiers OneDrive restés en ligne) |
| **Fichiers à la demande** | Les fichiers d'un dossier sont relus au moment où tu les déplies : la liste reflète le disque, et la mémoire reste légère même sur un disque entier |
| **Explorateur** | Un clic sur ↗ ouvre le dossier, ou sélectionne le fichier, dans l'Explorateur Windows |

## Installation

1. Télécharge `Aria-Scope-<version>.zip` dans les [Releases](../../releases) et décompresse-le où tu veux.
2. Lance `Aria-Scope.exe`. L'exe n'est pas signé : si Windows SmartScreen affiche « Windows a protégé votre ordinateur », clique sur « Informations complémentaires » puis « Exécuter quand même ».

## Bon à savoir

- **Dossiers illisibles** : sans droits administrateur, certains dossiers système sont refusés. Aria les compte et les signale d'un ⚠ dans l'arborescence. Lancé en administrateur (clic droit → « Exécuter en tant qu'administrateur »), il en lira la plupart.
- **Liens et jonctions** : ils ne sont pas suivis, pour ne rien compter deux fois. Windows en sème un peu partout (« Application Data », « Mes documents »…), ils apparaissent dans le compteur « liens ignorés ».
- **Réglages** : ils sont gardés dans `%APPDATA%\AriaScope\reglages.json`. L'exe, lui, reste portable.
- **Gros dossiers** : au-delà de 300 éléments, les plus légers sont regroupés sur une ligne « N autres éléments », pour que l'interface reste fluide même dans `WinSxS`.

## Compiler soi-même

La façon simple : **Actions → Compiler Aria Scope → Run workflow** sur GitHub. Le runner Windows teste, compile, et publie la release si tu coches la case.

Sur ton PC, avec Go 1.26 ou plus et Node.js LTS :

```powershell
powershell -ExecutionPolicy Bypass -File .\build.ps1
```

## Feuille de route

- **Ménage** : corbeille, déplacer, copier, archiver, renommage en masse, menu contextuel, annulation
- **Recherche** : recherche multicritère, chemins trop longs, doublons vérifiés par SHA-256
- **Export et historique** : CSV, HTML, Excel, PDF, SQLite ; analyses sauvegardées et comparées
- **Plus tard** : détails NTFS (dont le propriétaire des fichiers), ligne de commande et analyses planifiées, scan éclair via la MFT, version Linux

## Licence

Code d'Aria Scope sous licence MIT. Les composants tiers et leurs licences sont listés dans [LICENCES-TIERS.txt](LICENCES-TIERS.txt).
