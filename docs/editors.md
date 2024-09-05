---
title: Editors integration · C3LSP
description: How to integrate C3LSP with different editors
---

C3-LSP can be used with any editor that supports integration with a Langauge Server.

Usually your editor/IDE of choice will allow you to configure a Language Server (either natively or with the help of a plugin), and it will take care of starting it for you automatically.

The steps to configure your editor/IDE will go along the lines of:

- Download and place on your desired path the c3-lsp binary. https://github.com/pherrymason/c3-lsp/releases
- Configure your editor/IDE to tell it you want to use a Language server. Somehow you will need to tell him where the c3-lsp binary is located.

Following are instructions to configure typical editors/IDE's.

## Visual Studio Code
Use the [C3 language extension](https://marketplace.visualstudio.com/items?itemName=tonios2.c3-vscode) located in the Vscode marketplace.  
This extension will connect to the language server when editing c3 files. 
It already comes with c3-lsp binaries for Linux, Mac and Windows, so you don't need to install c3-lsp manually. Just be aware that it might not come with the last version. 

If the builtin c3-lsp version is not the last one, you can always download the last server manually and tell the extension to use the server located in the installed path.  
At the moment of writing this, this extension also includes syntax-highlighting.


### Alternative
For debugging purposes, this repo includes a simple vscode extension: https://github.com/pherrymason/c3-lsp/releases.  
This extension only includes connection with the language server, and is not available in the vscode marketplace.  
It does not include c3-lsp binaries, so it requires to specify the installation path.  
However, it is granted it supports all options featured by the language server.

A way to use the last version of C3-lsp while keeping syntax highlighting could be installing both extensions, disabling c3-lsp in **C3 extension** and enabling it in **C3 LSP Client extension**.  
This way you use can use the specific lsp version you want 


## Sublime Text

The [LSP Package](https://packagecontrol.io/packages/LSP) can be configured to
work with `c3-lsp`. Add the [syntax highlighting package](https://github.com/c3lang/editor-plugins/tree/main/sublime-text) and configure a LSP client in `Settings` -> `Package Settings` -> `LSP` -> `Settings` like the following:

```yaml
"clients": {
    "c3-lsp": {
      "enabled": true,
      // The command line required to run the server.
      "command": [
        "c3-lsp",
      ],

      "selector": "source.c3",
      "schemes": [
        "file"
      ],
      "diagnostics_mode": "open_files",
    }
  }
```

The first element in the `command` array is the path to the `c3-lsp` executable.

## nvim

```
local lspconfig = require('lspconfig')
  local util = require('lspconfig/util')
  local configs = require('lspconfig.configs')
  if not configs.c3_lsp then
	  configs.c3_lsp = {
	  	default_config = {
	  		cmd = { "/path/to/c3-lsp" },
	  		filetypes = { "c3", "c3i" },
	  		root_dir = function(fname)
	      		return util.find_git_ancestor(fname)
	    	end,
	    	settings = {},
	    	name = "c3_lsp"
	  	}
	  }
  end
  lspconfig.c3_lsp.setup{}
```