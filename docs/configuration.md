---
title: CLI arguments · C3LSP
description: Arguments accepted by command line
---

- help: Shows available arguments.
- send-crash-reports: Automatically reports crashes to server.
- log-path: Enables logs and sets its filepath.
- debug: Enables debug mode.
- lang-version: Specify C3 language version.
- c3c-path: Path where c3c is located.
- diagnostics-delay: Delay calculation of code diagnostics after modifications in source. In milliseconds, default 2000 ms.

# c3lsp.json
You can place a `c3lsp.json` file in your C3 project and configure most of the LSP settings from there. This allows to customize behaviour on per project basis.

Schema:
- C3
    - **version**: String, Optional. C3 compiler version your project uses. Serves to select the correct stdlib symbols table. If omitted, it will use last version lsp knows. 
    - **path**: String, Optional. Path to the C3 compiler you want to use. If omitted, c3c path must be defined in your OS PATH.
    - **stdlib-path**: String, Optional. Path to the sources of the stdlib. Allows to use `Go to Definition/Declaration` on stdlib symbols
- Diagnostics
    - enabled: Boolean. Enables Diagnostics feature. c3c path should be either in OS Path or properly configured in `C3.path` configuration.
    - delay: Integer, Optional. Number of milliseconds of delay to recalculate diagnostics. By default 2000.
   
**Note**
There's no current way to configure `send-crash-reports`, `log-path` or `debug` settings in `c3lsp.json`.

Example:
```
{
    "C3": {
        "version": "0.6.1",
        "path": "c3c",       
        "stdlib-path": "/Volumes/Development/c3c/lib/std"
    },
    "Diagnostics": {
        "enabled": true,
        "delay": 2000
    }
}
```