#!/usr/bin/env node

const { spawnSync } = require('child_process');

const baseURL = (process.env.DEPLOY_PRIME_URL && String(process.env.DEPLOY_PRIME_URL).trim()) || '/';

const npmExecutable = process.platform === 'win32' ? 'npm.cmd' : 'npm';
const result = spawnSync(
  npmExecutable,
  ['run', '_hugo:dev', '--', '--minify', '--baseURL', baseURL],
  {
    stdio: 'inherit',
    shell: process.platform === 'win32',
  }
);

process.exit(result.status ?? 1);
