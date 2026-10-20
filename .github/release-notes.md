**Aria Scope** {VERSION} : l'analyseur d'espace disque de la famille Aria, dans un seul exe Windows portable.

## Installation

1. Télécharge `Aria-Scope-{VERSION}.zip` ci-dessous et décompresse-le où tu veux.
2. Lance `Aria-Scope.exe`. L'exe n'est pas signé : si Windows SmartScreen affiche « Windows a protégé votre ordinateur », clique sur « Informations complémentaires » puis « Exécuter quand même ».

Rien à installer, aucun droit administrateur nécessaire. Le guide complet est dans le [README]({REPO_URL}#readme).

Le détail (commit d'Aria, toolchains) est dans `BUILDINFO.txt`, à l'intérieur du zip.

## Vérifier ce que tu as téléchargé

Attestation de provenance Sigstore (nécessite le [CLI GitHub](https://cli.github.com)) :

```powershell
gh attestation verify .\Aria-Scope-{VERSION}.zip --owner {OWNER}
gh attestation verify .\Aria-Scope.exe --owner {OWNER}
```
