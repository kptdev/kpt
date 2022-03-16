const functions = require("firebase-functions");


// Return meta tag for remote go import path
// See: https://pkg.go.dev/cmd/go#hdr-Remote_import_paths
const remoteGoImport = ({ importPrefix, vcs, repoRoot }) => {
  return `<meta name="go-import" content="${importPrefix} ${vcs} ${repoRoot}">`;
}

// Creates a firebase endpoint which implements a golang vanity server
const vanityGoEndpoint = ({ importPrefix, vcs, repoRoot }) =>
    functions.https.onRequest((request, response) => {
      if (request.query["go-get"] === "1") {
        return response.send(
            remoteGoImport({ importPrefix, vcs, repoRoot })
        );
      }
      return response.redirect(repoRoot);
    });

exports.configsync = vanityGoEndpoint({
  importPrefix: 'kpt.dev/configsync',
  vcs: 'git',
  repoRoot: 'https://github.com/GoogleContainerTools/kpt-config-sync.git'
});
