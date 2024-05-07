const { Trace } = require('vscode-jsonrpc');
const { window, workspace, commands, ExtensionContext, Uri } = require('vscode');
const { LanguageClient } = require('vscode-languageclient');

let client = null;

module.exports = {
  activate: function (context) {
    const executable = '/Volumes/Development/raul/c3/go-lsp/server/c3-lsp';
    const serverOptions = {
      run: {
        command: executable,
      },
      debug: {
        command:executable,
        options: { execArgv: ['--nolazy', '--inspect=6009'] }
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
