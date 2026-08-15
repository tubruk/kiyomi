# Rewrite

This document serves as outline for rewriting Kiyomi.

THIS FILE MUST NOT BE COMMITTED TO GIT

High priority:
1. Design (/docs/design) alignment
2. Clean architecture for both backend and frontend
3. Filesystem-based library
4. Provider plugin implementation
   1. plugins treated as built-in in this phase and built in the same binary, no WASM plugin yet.
   2. must already implemented fingerprinting feature to avoid being blocked by the upstream service
5. Exploring providers for mangas
6. Adding mangas from provider to library
7. Browsing and adding library

Deferred complex features:
- Caching
- Library index DB
- Downloading, all async/background queue
- Progress tracking
