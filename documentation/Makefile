.PHONY: verify
verify:
	go run github.com/golangci/misspell/cmd/misspell@v0.8.0 -error content/

# Build the full versioned docs site (all versions from versions.json), the
# same build Netlify runs. Output goes to documentation/public by default;
# override with OUTPUT_DIR=<dir>.
#
# The build runs from the canonically resolved repo path (realpath) so it works
# when the checkout is reached through a symlink. Node's filesystem permission
# model (Node >= 23) otherwise denies PostCSS from following the symlink during
# CSS processing, which breaks the Hugo build.
OUTPUT_DIR ?= $(CURDIR)/public

.PHONY: build-versioned
build-versioned:
	@command -v hugo >/dev/null 2>&1 || { echo "Error: hugo not found. See https://gohugo.io/installation/"; exit 1; }
	@command -v npm >/dev/null 2>&1 || { echo "Error: npm not found. See https://nodejs.org/"; exit 1; }
	@real_root="$$(realpath $(CURDIR)/..)"; \
	real_out="$$(realpath -m $(OUTPUT_DIR))"; \
	HUGO_ENV=production "$$real_root/scripts/build-versioned-docs.sh" "$$real_out"

# Build the full versioned site and serve it locally. This is the one command
# to preview the docs exactly as deployed, including the versioned subsites
# (/, /v1.0/). A plain static file server is used because `hugo server` only
# renders the current working tree and would omit the tagged versions.
# Override the port with SERVE_PORT=<port>.
SERVE_PORT ?= 1313

.PHONY: serve
serve: build-versioned
	@command -v python3 >/dev/null 2>&1 || { echo "Error: python3 not found."; exit 1; }
	@out="$$(realpath -m $(OUTPUT_DIR))"; \
	echo "==> Serving $$out at http://localhost:$(SERVE_PORT)/ (Ctrl+C to stop)"; \
	echo "    latest -> /   v1.0 -> /v1.0/"; \
	python3 -m http.server $(SERVE_PORT) --directory "$$out"

.PHONY: check-links-external
check-links-external: production-build
	@command -v lychee >/dev/null 2>&1 || { echo "Error: lychee not found. Install it from https://github.com/lycheeverse/lychee#installation"; exit 1; }
	lychee -vv --config lychee.toml --root-dir public --extensions html public

# production-build builds only the current working tree (single, unversioned
# site). Used by check-links-external and the weekly docs link-check workflow
# (.github/workflows/docs-weekly.yml). Netlify production uses build-versioned.
production-build:
	git fetch --tags
	hugo version
	HUGO_KPT_VERSION=$$(git describe --tags --abbrev=0 --match='v[0-9]*' 2>/dev/null || true) hugo \
		--minify
	npx -y pagefind --site public

# preview-build builds only the current working tree (with drafts/future posts)
# against DEPLOY_PRIME_URL. Used by Netlify deploy-preview and branch-deploy
# contexts; PR previews don't rebuild the tagged version snapshots.
preview-build:
	git fetch --tags
	HUGO_KPT_VERSION=$$(git describe --tags --abbrev=0 --match='v[0-9]*' 2>/dev/null || true) hugo \
		--baseURL $(DEPLOY_PRIME_URL) \
		--buildDrafts \
		--buildFuture \
		--minify
	npx -y pagefind --site public
