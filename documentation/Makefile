EXTERNAL_REPO = https://github.com/kptdev/krm-functions-catalog.git
CLONE_DIR = tmp-krm-clone
CONTENT_SUBDIR = documentation/content/en/function-catalog/
TARGET_HUGO_DIR = content/en/function-catalog/

LAYOUT_SUBDIR = documentation/layouts/
TARGET_HUGO_LAYOUT_DIR = layouts/

EXAMPLES_SUBDIR = examples/
TARGET_HUGO_EXAMPLES_DIR = content/en/examples/

.PHONY: serve production-build preview-build fetch-external-docs

fetch-external-docs:
	@echo "--- Starting content synchronization from $(EXTERNAL_REPO) ---"
	rm -rf $(CLONE_DIR)
	git clone --depth 1 $(EXTERNAL_REPO) $(CLONE_DIR)
	mkdir -p $(TARGET_HUGO_DIR)
	mkdir -p $(TARGET_HUGO_EXAMPLES_DIR)
	cp -r $(CLONE_DIR)/$(CONTENT_SUBDIR)* $(TARGET_HUGO_DIR)
	cp -r -n $(CLONE_DIR)/$(LAYOUT_SUBDIR)* $(TARGET_HUGO_LAYOUT_DIR)
	cp -r $(CLONE_DIR)/$(EXAMPLES_SUBDIR)* $(TARGET_HUGO_EXAMPLES_DIR)
	rm -rf $(CLONE_DIR)
	@echo "--- External documentation ready in $(TARGET_HUGO_DIR) ---"

serve:
	hugo server \
		--disableFastRender \
		--buildDrafts \
		--buildFuture \
		--ignoreCache
		--printI18nWarnings \
		--printMemoryUsage \
		--printPathWarnings \
		--printUnusedTemplates \
		--templateMetrics \
		--templateMetricsHints \
		--gc


production-build: fetch-external-docs
	git submodule update --init --recursive
	hugo version
	hugo \
		--minify
	npx -y pagefind --site public


preview-build: fetch-external-docs
	git submodule update --init --recursive
	hugo \
		--baseURL $(DEPLOY_PRIME_URL) \
		--buildDrafts \
		--buildFuture \
		--minify
	npx -y pagefind --site public
