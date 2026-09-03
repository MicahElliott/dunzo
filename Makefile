.PHONY: run build package clean vet release dunzo

build: dunzo

# dunzo is deliberately unconditional (.PHONY, no file-based
# prerequisites) rather than depending on `$(shell find . -name
# '*.go')` -- that pattern is vulnerable to a real gotcha: Make's
# staleness check is strictly mtime(prerequisite) > mtime(target), so
# if a .go edit and the resulting `go build` both land within the same
# filesystem-mtime tick, the next `make build`/`make run` can see "no
# .go file newer than dunzo" and silently skip rebuilding even though
# the source changed -- exactly what happened during the 2026-09-02
# session (compounded there by also running a stale packaged
# Dunzo.app instead of this binary at all). `go build` itself is
# already near-instant when nothing changed (its own content-hash
# based cache), so always invoking it here costs essentially nothing
# and removes the whole class of tie/staleness bugs.
dunzo:
	go build -o dunzo .

run: build
	./dunzo

# Requires the `fyne` CLI (go install fyne.io/tools/cmd/fyne@latest).
# Produces Dunzo.app with the custom icon (macOS only for now).
package:
	fyne package -os darwin -icon Icon.png

vet:
	go vet ./...

clean:
	rm -f dunzo
	rm -rf Dunzo.app
	rm -f Dunzo-*-macos.zip

# Cuts a local macOS-only release: packages Dunzo.app, zips it, tags
# the current commit, pushes the tag, and creates a GitHub Release
# with the zip attached (via `gh`, using auto-generated notes from
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
	rm -f Dunzo-$(VERSION)-macos.zip
	zip -r -X Dunzo-$(VERSION)-macos.zip Dunzo.app
	@# `fyne package` bumps FyneApp.toml's internal Build counter as a
	@# side effect -- discard that so tagging happens on a clean tree
	@# matching what was already committed.
	git checkout -- FyneApp.toml
	git tag $(VERSION)
	env -u GH_TOKEN git push origin $(VERSION)
	env -u GH_TOKEN gh release create $(VERSION) Dunzo-$(VERSION)-macos.zip --title "$(VERSION)" --generate-notes
