## Visual Code Studio LSP Client
Simple extension to use c3-lsp server

### Configuration
- *Enable*: `c3lspclient.lsp.enable` Enables or disables the connection with the Language server.
- *Binary path*: `c3lspclient.lsp.path` The path to the **c3-lsp** binary. Mandatory or extension will fail to start.
- *Send crash reports*: `c3lspclient.lsp.sendCrashReports` Enables sending crash reports to a server. Will help debug possible bugs. Disabled by default.