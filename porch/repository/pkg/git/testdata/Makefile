REPOSITORIES = \
  empty-repository.tar \
  trivial-repository.tar \
  simple-repository.tar \
  drafts-repository.tar \
  nested-repository.tar


.PHONY: expand
expand:
	@for f in $(REPOSITORIES); do \
	  dir=$${f%.*}; \
	  rm -rf $${dir}; mkdir $${dir}; \
	  tar xf $${f} -C $${dir}; \
	  echo "$${f} --> $${dir}/"; \
	done

.PHONY: checkout
checkout: expand
	@for f in $(REPOSITORIES); do \
	  dir=$${f%.*}; \
	  (cd $${dir}; git reset --hard refs/heads/main -- 2>/dev/null ); \
	done

.PHONY: tars
tars:
	@for f in $(REPOSITORIES); do \
	  dir=$${f%.*}; \
	  ../../../../hack/tar-test-repo.sh $${dir}/ $${f}; \
	  rm -rf $${dir}; \
	  echo "$${dir}/ --> $${f}"; \
	done

.PHONY: clean
clean:
	@for f in $(REPOSITORIES); do \
	  dir=$${f%.*}; \
	  rm -rf $${dir}; \
	done
