# Changelog

Tous les changements notables apportés à ce projet seront documentés dans ce fichier.

Le format est basé sur [Keep a Changelog](https://keepachangelog.com/fr/1.0.0/),
et ce projet adhère au [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2025-03-29

### Added
- Support des chemins Windows
- Package dédié à la gestion des chemins (internal/pathutil)
- Détection automatique de la plateforme (Windows/Unix)
- Gestion des chemins longs Windows (>260 caractères)
- Gestion des caractères invalides dans les noms de fichiers Windows
- Gestion des noms de fichiers réservés sous Windows

### Changed
- Amélioration de la structure du projet avec séparation des responsabilités
- Utilisation de chemins sanitisés pour toutes les opérations de fichiers
- Meilleure gestion des erreurs pour les opérations de fichiers

## [0.1.0] - 2025-03-29

### Added
- Migration initiale du projet Python vers Go
- Implémentation de la fonctionnalité de crawling d'Artifactory
- Support pour le téléchargement de fichiers via HTTP et wget
- Structure de projet organisée (cmd/internal)
- Configuration via des flags de ligne de commande
- Workflows CI/CD avec GitHub Actions
- Support pour les conventional commits
