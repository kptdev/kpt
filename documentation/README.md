# kpt Documentation Site

[![Netlify Status](https://api.netlify.com/api/v1/badges/57cfe7e6-fce7-4a0e-b00b-f6cc68b3f506/deploy-status)](https://app.netlify.com/projects/kptdocs/deploys)

This directory contains a [Hugo](https://gohugo.io) web site published via [Netlify](https://www.netlify.com/) to
<https://kptdocs.netlify.app> what is redirected to <https://kpt.dev/>.

When the `main` branch of this repo is updated a fresh build and deploy of the website is executed. Recent Netlify
builds and deployments are listed at <https://app.netlify.com/sites/kptdocs>.

Add content by adding Markdown files to directories in [./content](./content).

Update layouts for each content type in [./layouts](./layouts/).

Configuration is set in [config.toml](./config.toml).

## Setting up a local dev instance

To set up a local dev environment make sure you have [npm](https://www.npmjs.com/) installed, then run the following
from this folder:

```sh
npm install
```

Then run the site using `make serve`. 

### Windows note (PowerShell/CMD)

The site pulls some dependencies via Git submodules. If `npm install` succeeds but the site fails to build (for example, missing theme assets), initialize submodules and try again:

```powershell
git submodule update --init --recursive
```

## License

Licensed under the [Creative Commons Attribution 4.0 International license](LICENSE-documentation)
