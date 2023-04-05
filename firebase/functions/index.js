/**
 * Copyright 2022 The kpt Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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

exports.resourcegroup = vanityGoEndpoint({
  importPrefix: "kpt.dev/resourcegroup",
  vcs: "git",
  repoRoot: "https://github.com/GoogleContainerTools/kpt-resource-group.git",
});
