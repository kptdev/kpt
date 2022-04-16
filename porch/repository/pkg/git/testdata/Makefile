.PHONY: expand
expand:
	rm -rf drafts-repository; mkdir drafts-repository
	cd drafts-repository; tar xfv ../drafts-repository.tar; git reset --hard main

.PHONY: tars
tars:
	../../../../hack/tar-test-repo.sh drafts-repository/ drafts-repository.tar
