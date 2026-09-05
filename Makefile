.PHONY: run build package clean vet release dunnit

TARGET_OS ?= $(shell go env GOOS)

build: dunnit

# dunnit is deliberately unconditional (.PHONY, no file-based
# prerequisites) rather than depending on `$(shell find . -name
# '*.go')` -- that pattern is vulnerable to a real gotcha: Make's
# staleness check is strictly mtime(prerequisite) > mtime(target), so
# if a .go edit and the resulting `go build` both land within the same
# filesystem-mtime tick, the next `make build`/`make run` can see "no
# .go file newer than dunnit" and silently skip rebuilding even though
# the source changed -- exactly what happened during the 2026-09-02
# session (compounded there by also running a stale packaged
# Dunnit.app instead of this binary at all). `go build` itself is
# already near-instant when nothing changed (its own content-hash
# based cache), so always invoking it here costs essentially nothing
# and removes the whole class of tie/staleness bugs.
dunnit:
	go build -o dunnit .

run: build
	./dunnit

# Requires the `fyne` CLI (go install fyne.io/tools/cmd/fyne@latest).
# Produces a native desktop package with the custom icon.
package:
	@if [ "$(TARGET_OS)" != "darwin" ] && [ "$(TARGET_OS)" != "linux" ]; then \
		echo "Unsupported desktop target: $(TARGET_OS) (supported: darwin, linux)"; \
		exit 1; \
	fi
	fyne package -os $(TARGET_OS) -icon Icon.png

vet:
	go vet ./...

clean:
	rm -f dunnit
	rm -rf Dunnit.app
	rm -f Dunnit-*-macos.zip Dunnit-*-linux.tar.xz Dunnit.tar.xz

# Cuts a local native release: packages for the host OS, tags
# the current commit, pushes the tag, and creates a GitHub Release
# with the native artifact attached (via `gh`, using auto-generated notes from
# commits since the last tag). No CI/GoReleaser involved -- this is
# purely a local, manual-trigger convenience wrapper.
#
# Usage: make release VERSION=v0.1.0
release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make release VERSION=vX.Y.Z"; \
		exit 1; \
	fi
	@if ! git diff --quiet || ! git diff --cached --quiet; then \
		echo "Working tree has uncommitted changes -- commit or stash first."; \
		exit 1; \
	fi
	$(MAKE) package
	@if [ "$(TARGET_OS)" = "darwin" ]; then \
		rm -f Dunnit-$(VERSION)-macos.zip; \
		zip -r -X Dunnit-$(VERSION)-macos.zip Dunnit.app; \
	else \
		mv Dunnit.tar.xz Dunnit-$(VERSION)-linux.tar.xz; \
	fi
	@# `fyne package` bumps FyneApp.toml's internal Build counter as a
	@# side effect -- discard that so tagging happens on a clean tree
	@# matching what was already committed.
	git checkout -- FyneApp.toml
	git tag $(VERSION)
	env -u GH_TOKEN git push origin $(VERSION)
	@if [ "$(TARGET_OS)" = "darwin" ]; then \
		artifact=Dunnit-$(VERSION)-macos.zip; \
	else \
		artifact=Dunnit-$(VERSION)-linux.tar.xz; \
	fi; \
	env -u GH_TOKEN gh release create $(VERSION) "$$artifact" --title "$(VERSION)" --generate-notes
