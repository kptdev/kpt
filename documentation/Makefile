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
