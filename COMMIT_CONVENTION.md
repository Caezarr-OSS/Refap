# Commit Convention

Ce document décrit le format des messages de commit utilisés dans le projet Refap.

## Types de Commit

- `feat`: Nouvelles fonctionnalités
- `fix`: Corrections de bugs
- `docs`: Modifications de la documentation
- `style`: Changements de style (formatting, semi-colons, etc.)
- `refactor`: Refactorisation de code sans modification du comportement
- `perf`: Améliorations de performance
- `test`: Ajout ou modification de tests
- `chore`: Tâches de maintenance, mise à jour des dépendances
- `ci`: Changements liés à l'intégration continue
- `build`: Changements du système de build
- `revert`: Annulation d'un commit précédent

## Scopes

- `core`: Logique d'application principale
- `config`: Système de configuration
- `crawler`: Système de crawling d'Artifactory
- `download`: Système de téléchargement
- `logging`: Système de journalisation
- `auth`: Authentification et autorisation
- `cmd`: Commandes CLI

## Format

```
type(scope): description

[optional body]

[optional footer]
```

Exemple:
```
feat(crawler): ajouter le support du filtrage par extension de fichier

- Implémente le mode whitelist pour filtrer les extensions incluses
- Implémente le mode blacklist pour filtrer les extensions exclues
```

## Branche Git Flow

Le projet suit le workflow GitFlow:

- `main`: Code prêt pour la production
- `develop`: Branche d'intégration pour les développements
- Branches de fonctionnalités: Créées à partir de `develop` (format: `feature/nom-fonctionnalité`)
- Branches de version: Créées à partir de `develop` pour préparer une release (format: `release/x.y.z`)
- Branches de correctif: Créées à partir de `main` pour les correctifs urgents (format: `hotfix/x.y.z`)
