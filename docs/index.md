---
title: C3LSP
description: Language server for C3 language
---

- [Configuration](/config.html)
- [Editor integration](/editors.html)

# Project Goals

Writing a Language Server Protocol (LSP) can be a complex and challenging task. To manage this complexity, our initial focus is on covering the basic yet essential needs of a typical LSP implementation.

## Current target

The main current target, is to cover the most essential feature which is to scan precise information about symbols used within the source code of a project.
This information can then be used by an IDE to enable the following features:

- Go to Definition: Navigate to the exact location where a symbol is defined.
- Hover Information: Display detailed information about symbols when hovering over them.
- Autocomplete: Suggest relevant symbols and code completions as you type.

These features will significantly improve the developer experience by providing accurate and efficient code navigation and assistance.
Future plans
Once these initial objectives are completed, we will explore additional functionalities that can be added to the project, further enhancing its capabilities and usefulness.