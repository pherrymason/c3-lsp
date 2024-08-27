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
      args.push('--send-crash-reports');
    }

    if (config.get('log.path').length > 0) {
        args.push('--log-path='+config.get('log.path'));
    }

    if (config.get('c3.version')) {
        args.push('--lang-version='+config.get('c3.version'));
    }

    if (config.get('c3.path')) {
        args.push('--c3c-path='+config.get('c3.path'));
    }

    if (config.get('diagnosticsDelay')) {
        args.push('--diagnostics-delay='+config.get('diagnosticsDelay'));
    }

    const serverOptions = {
      run: {
        command: executable,
        args: args,
      },
      debug: {
        command:executable,
        args: args,
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
