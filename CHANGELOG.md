# Journal des versions

Ce qui a changé d'une version à l'autre d'Aria Scope. Les exécutables sont dans les [Releases](../../releases).

## En cours : v0.7 « Export et historique »

Déjà sur la branche principale, pas encore publié :

- **Export CSV** de chaque vue (arbre avec la profondeur de ton choix, gros fichiers, types, âge, recherche, doublons), prêt pour Excel en français : UTF-8 avec BOM, « ; », virgule décimale, dates reconnues. Un nom de fichier n'est jamais exécuté comme une formule.
- **Rapport HTML** : un seul fichier autonome (résumé, carte, plus gros dossiers, types, âge, plus gros fichiers, doublons), avec une mise en page d'impression pour en faire un PDF.
- **Classeur Excel (.xlsx)** écrit sans aucune dépendance : un onglet par vue, colonnes typées, en-tête figé et filtre automatique.

Reste à venir : l'historique des analyses et leur comparaison.

## v0.6.0 « Recherche » · 1er novembre 2026

- **Recherche multicritère** : nom (jokers, texte ou regex), taille, période de modification, famille de types, longueur du chemin ; fichiers, dossiers ou les deux ; dans toute l'analyse ou un dossier. Jusqu'à 10 000 résultats, triables, et « tout sélectionner ».
- **Chemins trop longs** : un clic trouve ce qui atteint 260 caractères, avec une explication et de quoi raccourcir.
- **Doublons vérifiés** : regroupés par taille, empreinte rapide, puis SHA-256 complet ; liens physiques écartés. Présélections « le plus ancien », « le plus récent », « les copies de ce dossier ».
- **Garde-fou des doublons** : la dernière copie d'un groupe ne peut pas partir à la corbeille.
- Corrigé : à la fin d'un archivage, une nouvelle opération lancée aussitôt pouvait être refusée à tort.

## v0.5.0 « Ménage » · 23 octobre 2026

- **Sélection multiple** (Ctrl+clic, Maj+clic) et barre de sélection avec un total sans double compte.
- **Corbeille**, **déplacer**, **copier**, avec un aperçu avant d'agir : place nécessaire, noms déjà pris, éléments écartés et pourquoi.
- **Archive .zip** relue en entier avant d'être validée, et en option les originaux à la corbeille.
- **Garde-fous** : dossiers système, programmes, profil et AppData jamais touchés ; jamais de suppression définitive ; double confirmation sur un disque amovible.
- **Journal et annulation** : remettre un déplacement en place, recycler une copie, restaurer les originaux d'une archive, même longtemps après.
- L'analyse suit chaque opération sans rescan.

## v0.4.0 « Tout le PC » · 21 octobre 2026

- Tous les disques internes et externes analysés en même temps, réunis sous « Ce PC ».
- Analyse lancée automatiquement à l'ouverture (désactivable), réglage gardé dans `%APPDATA%\AriaScope`.

## v0.3.0 « Enquête » · 21 octobre 2026

- Les **1 000 plus gros fichiers** de l'analyse.
- Répartition par **type** (familles et extensions) et par **âge**.
- **Filtres** appliqués pendant le scan : exclure, inclure, regex, taille et ancienneté minimales.

## v0.2.0 « Voir » · 20 octobre 2026

- **Treemap néon** et **graphique en anneau**, synchronisés avec l'arborescence.
- Tri sur toutes les colonnes, colonne « Modifié ».

## v0.1.1 · 20 octobre 2026

- Nouveau logo aux jumelles, icône de l'exe, animation de balayage pendant une analyse.
- Corrigé : « C: » reconnu comme la racine du lecteur.

## v0.1.0 · 20 octobre 2026

- Premier noyau : choix d'un disque ou d'un dossier, scan parallèle, arborescence triée, taille réelle et sur disque, ouverture dans l'Explorateur.
