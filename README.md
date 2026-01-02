# navidown

`navidown` is a reusable navigable markdown component:

- renders markdown to **ANSI** (for terminal output)
- supports **scrolling** and **pager-style navigation**
- finds **links** and allows **Tab / Shift-Tab** traversal
- on activation (Enter), loads linked markdown via a pluggable loader and replaces current content

This repo contains:

- core package `github.com/boolean-maybe/navidown/navidown` (UI-agnostic state machine + renderers)
- optional adapter `github.com/boolean-maybe/navidown/navidown/tview` (TView primitive)