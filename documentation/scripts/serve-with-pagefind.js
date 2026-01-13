#!/usr/bin/env node

const { spawnSync } = require('child_process');

function run(command, args, extraEnv) {
  const result = spawnSync(command, args, {
    stdio: 'inherit',
    shell: process.platform === 'win32',
    env: {
      ...process.env,
      ...(extraEnv || {}),
    },
  });

  if (result.error) {
    console.error(result.error.message);
    process.exit(1);
  }

  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

const hugo = 'hugo';
const npm = process.platform === 'win32' ? 'npm.cmd' : 'npm';
const npx = process.platform === 'win32' ? 'npx.cmd' : 'npx';

// Force Hugo to use the local PostCSS binary. This avoids Windows path-with-spaces issues
// when Hugo resolves postcss via PATH / shell.
const postcssBin = process.platform === 'win32'
  ? `${process.cwd()}\\node_modules\\.bin\\postcss.cmd`
  : `${process.cwd()}/node_modules/.bin/postcss`;

// Prepend node_modules/.bin to PATH so Hugo can find postcss and friends.
// Use the correct separator for the current platform.
const pathKey = Object.keys(process.env).find((k) => k.toLowerCase() === 'path') || 'PATH';
const pathSep = process.platform === 'win32' ? ';' : ':';
const binDir = process.platform === 'win32'
  ? `${process.cwd()}\\node_modules\\.bin`
  : `${process.cwd()}/node_modules/.bin`;
const patchedPath = `${binDir}${pathSep}${process.env[pathKey] || ''}`;

// Theme is configured via Hugo Modules in config.toml (theme = ["github.com/google/docsy"]).
// Do not pass --theme=docsy here; it expects a local themes/docsy directory.
run(hugo, ['--baseURL=/', '--minify'], {
  HUGO_POSTCSS: postcssBin,
  [pathKey]: patchedPath,
});

run(npx, [
  'pagefind',
  '--site',
  'public',
  '--output-subdir',
  '../static/pagefind',
]);

run(npm, ['run', 'serve']);
