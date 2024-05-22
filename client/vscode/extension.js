const { Trace } = require('vscode-jsonrpc');
const { window, workspace, commands, ExtensionContext, Uri } = require('vscode');
const { LanguageClient } = require('vscode-languageclient');

let client = null;
const config = workspace.getConfiguration('c3lspclient.lsp');

module.exports = {
  activate: function (context) {
    const enabled = config.get('enable');
    if (!enabled) {
        return;
    }

    const executable = config.get('path');
    let args = [];
    if (config.get('sendCrashReports')) {
      args.push('--sendCrashReports');
    }

    const serverOptions = {
      run: {
        command: executable,
        options: { execArgv: args }
      },
      debug: {
        command:executable,
        options: { execArgv: ['--nolazy', '--inspect=6009', ...args] }
      }
    }

    const clientOptions = {
      documentSelector: [{ scheme: 'file', language:'c3'}],
      synchronize: {
        fileEvents: workspace.createFileSystemWatcher('**/*.c3')
      }
    }

    client = new LanguageClient(
      'C3LSP',
      serverOptions,
      clientOptions
    );
    client.setTrace(Trace.Verbose);
    client.start();
  },

  deactivate: function () {
    return client.stop();
  }
}
