SHELL_SRC_FILES := $(shell find . -type f -name '*.sh')

.PHONY: build
build:
	./scripts/build.sh

.PHONY: test
test:
	gotestsum --format-hide-empty-pkg

.PHONY: run
run: build
	./otter

.PHONY: clean
clean:
	rm otter

.PHONY: lint
lint: lint/shellcheck

.PHONY: lint/shellcheck
lint/shellcheck: $(SHELL_SRC_FILES)
	shellcheck --external-sources $(SHELL_SRC_FILES)
