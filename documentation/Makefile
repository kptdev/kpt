serve:
	hugo server \
		--disableFastRender \
		--buildDrafts \
		--buildFuture \
		--ignoreCache \
		--printI18nWarnings \
		--printMemoryUsage \
		--printPathWarnings \
		--printUnusedTemplates \
		--templateMetrics \
		--templateMetricsHints \
		--gc

.PHONY: verify
verify:
	go run github.com/golangci/misspell/cmd/misspell@v0.8.0 -error content/

.PHONY: check-links-external
check-links-external: production-build
	@command -v lychee >/dev/null 2>&1 || { echo "Error: lychee not found. Install it from https://github.com/lycheeverse/lychee#installation"; exit 1; }
	lychee 'public/**/*.html'

production-build:
	git submodule update --init --recursive
	git fetch --tags
	hugo version
	HUGO_KPT_VERSION=$$(git describe --tags --abbrev=0 --match='v[0-9]*' 2>/dev/null || true) hugo \
		--minify
	npx -y pagefind --site public

preview-build:
	git submodule update --init --recursive
	git fetch --tags
	HUGO_KPT_VERSION=$$(git describe --tags --abbrev=0 --match='v[0-9]*' 2>/dev/null || true) hugo \
		--baseURL $(DEPLOY_PRIME_URL) \
		--buildDrafts \
		--buildFuture \
		--minify
	npx -y pagefind --site public
