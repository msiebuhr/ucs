
.PHONY: git-install-hook
git-install-hook: .git/hooks/pre-commit

.git/hooks/pre-commit:
	curl https://golang.org/misc/git/pre-commit > $@
	chmod +x $@
