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

Make sure not to change the title of the alert type. It should always be either Note, Warning or Critical.

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
   $ kpt fn eval my-package --image ghcr.io/kptdev/krm-functions-catalog/search-replace
   ```

   - Incorrect:

   ```shell
   $ kpt fn eval --image ghcr.io/kptdev/krm-functions-catalog/search-replace my-package
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


## License

Licensed under the [Creative Commons Attribution 4.0 International license](../LICENSE-documentation)
