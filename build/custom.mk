## Runs errcheck against all packages.
.PHONY: errcheck
errcheck:
ifneq ($(HAS_SERVER),)
	@echo Running errcheck
	@# Workaroung because you can't install binaries without adding them to go.mod
	env GO111MODULE=off $(GO) get github.com/kisielk/errcheck
	errcheck ./server/...
endif
