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

Then run the site using `make serve`. This builds the full versioned site (the
current working tree at `/` plus a snapshot of each released version listed in
[versions.json](./versions.json)) and serves it locally, matching what Netlify
deploys. It requires *python3* to serve the built output.

## Versioned documentation

The published site is versioned: the current `main` working tree is served at
the site root as *latest*, and each released version in
[versions.json](./versions.json) is built from its git tag and served under a
subdirectory (for example, `/v1.0/`). A version selector in the navbar switches
between them.

Each version entry tracks a *minor* release line (`v1.0`, `v1.1`, ...), which
is the unit at which the documentation changes meaningfully; patch releases
(`v1.0.1`, `v1.0.2`, ...) roll up into their minor's entry automatically. The
`tagPattern` glob resolves to the newest stable patch on that line, so a new
patch is picked up on the next build with no manifest change.

To publish a new minor version, add an entry to
[versions.json](./versions.json) with the version label (for example, `v1.1`),
a `tagPattern` glob (`v1.1.*`), and a `path` (`/v1.1/`), then move the
`latestGA` flag to the new entry. Moving `latestGA` marks the previous line as
archived (it starts showing the "no longer actively maintained" banner) and
keeps the new line banner-free as the current release. The build script
([scripts/build-versioned-docs.sh](../scripts/build-versioned-docs.sh))
resolves the pattern to the newest matching stable release tag (pre-releases
such as `-beta` are ignored) and builds that version's content with the current
theme and layouts.

## Checking external links

To validate external links in the documentation, use:

```sh
make check-links-external
```

This builds the site with Hugo and runs [lychee](https://github.com/lycheeverse/lychee) against the rendered HTML.

### Prerequisites

- **Hugo** — installed via `npm install` (from `devDependencies`) or [standalone](https://gohugo.io/installation/)
- **lychee** — install via one of:
  - macOS: `brew install lychee`
  - Linux/macOS (Cargo): `cargo install lychee`
  - Binary download: see [lychee releases](https://github.com/lycheeverse/lychee/releases)

### Using a GitHub token

To avoid GitHub rate limiting, pass a token:

```sh
GITHUB_TOKEN=$(gh auth token) make check-links-external
```

The token is only sent to github.com domains.

## Style guide for documentation

1. Use US English in the documentation

2. Do not manually add a table of contents to the documents. Hugo and Docsy take care of this.

3. Do not use H1 (#) headers in the documents. Docsy generates an H1 header for every document
   consistent with the title of the document. Start the headings with H2 (##)

4. There are three alert types available based on the importance of the information:

   | Alert type | Code              | Alert color |
   |------------|-------------------|-------------|
   | Note       | `color="primary"` | Blue        |
   | Warning    | `color="warning"` | Yellow      |
   | Critical   | `color="danger"`  | Red         |

   Make sure not to change the alert title. It should always be either `Note`, `Warning`, or `Critical`.

   ```markdown
   {{%/* alert title="Note" color="primary" */%}}
   Important information here.
   {{%/* /alert */%}}
   ```

5. If you add any commands to the content inline, surround the command with backticks (\` \`), like `ls -la`

6. Do not surround IP addresses, domain names, or any other identifiers with backticks. Use italics
   (for example, `*example.com*`) to mark any inline IP address, domain name, file name, file location, or similar.

7. Whenever possible, define the type of code for your code blocks
   - <code>```shell</code> for all shell blocks
   - <code>```golang</code> for all Go blocks
   - <code>```yaml</code> for all YAML blocks
   - <code>```yang</code> for all YANG blocks
   - a full list of language identifiers is available [here](https://gohugo.io/content-management/syntax-highlighting/#list-of-chroma-highlighting-languages)


8. Links to other kpt doc pages should be absolute:
   - Correct: `[pkg]: /reference/cli/pkg/get/`
   - Incorrect: `[pkg]: ../../../reference/cli/pkg/get`

9. Flags must appear after positional args:

   - Correct:

   ```shell
   kpt fn eval my-package --image ghcr.io/kptdev/krm-functions-catalog/search-replace
   ```

   - Incorrect:

   ```shell
   kpt fn eval --image ghcr.io/kptdev/krm-functions-catalog/search-replace my-package
   ```

10. The name of the tool should always appear as small caps (even at start of
   sentences) and not in block quotes:
   - Correct: kpt
   - Incorrect: `kpt`
   - Incorrect: Kpt
   - Incorrect: KPT

11. References to a particular KRM group, version, kind, field should appear with
   inline quotes:
   - Correct: `ConfigMap`
   - Incorrect: ConfigMap

12. Do not add any TBDs to the documentation. If something is missing, create an [issue](https://github.com/kptdev/kpt/issues) for it.

13. Do not prefix shell commands with `$` in code blocks that contain only commands.
   Use `$` only when a block shows both the command and its output, to distinguish
   the command from the output:

   - Correct (command only):

   ```shell
   kpt fn render my-package
   ```

   - Correct (command + output):

   ```shell
   $ kpt fn render my-package
   Package "my-package":
   [PASS] "ghcr.io/kptdev/krm-functions-catalog/set-labels:latest"
   ```

   - Incorrect (command only with $):

   ```shell
   $ kpt fn render my-package
   ```


## License

Licensed under the [Creative Commons Attribution 4.0 International license](../LICENSE-documentation)
