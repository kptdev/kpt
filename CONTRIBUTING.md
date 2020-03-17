# How to Contribute

We'd love to accept your patches and contributions to this project. There are
just a few small guidelines you need to follow.

## Contributor License Agreement

Contributions to this project must be accompanied by a Contributor License
Agreement. You (or your employer) retain the copyright to your contribution;
this simply gives us permission to use and redistribute your contributions as
part of the project. Head over to <https://cla.developers.google.com/> to see
your current agreements on file or to sign a new one.

You generally only need to submit a CLA once, so if you've already submitted one
(even if it was for a different project), you probably don't need to do it
again.

## Code reviews

All submissions, including submissions by project members, require review. We
use GitHub pull requests for this purpose. Consult
[GitHub Help](https://help.github.com/articles/about-pull-requests/) for more
information on using pull requests.

## Community Guidelines

This project follows
[Google's Open Source Community Guidelines](https://opensource.google.com/conduct/).

## Building the Source

1. Clone the project

    ```sh
   git clone https://github.com/GoogleContainerTools/kpt
   cd kpt
   ```

2. Build `kpt` to `$(go env GOPATH)/bin/kpt`

    ```sh
    make
    ```

3. Run test

    ```sh
    make all
    ```

## Contributing to docs

### Run the docs locally

- install hugo
- `make servedocs`

### Update docs

- `make docs`
- `git add .`
- `git commit`

### Adding or updating docs

Docs are under `docsy/` and use the
[docsy](https://github.com/google/docsy) theme for hugo.  Learn more
about docsy [here](https://www.docsy.dev/docs/).

### Adding or updating diagrams

- Diagrams are created using Omnigraffle
- Open docsy/diagrams/diagrams.graffle in omnigraffle
- Change the diagram you want (or add a new canvas)
- **Convert text to shapes!!!** -> Edit -> Objects -> Convert Text to Shapes
- Export the canvas as an svg to `docsy/static/images` 
- **Undo convert text to shapes!!!** with command-z
  - This is important
- Reference the image using the `svg` shortcode

## Adding or updating asciinema

- asciinema casts are under `docsy/static/casts`
- add or modify a `*.sh*` script which will run the commands that will
  be recorded
- run `make-gif.sh` with the script name (without extension) as the argument
- add the updated cast to git
- Reference the cast using the `asciinema` shortcode
