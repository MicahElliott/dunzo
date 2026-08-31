.PHONY: run package clean vet release

build: dunzo

dunzo: $(shell find . -name '*.go')
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
	ditto -c -k --sequesterRsrc --keepParent Dunzo.app Dunzo-$(VERSION)-macos.zip
	@# `fyne package` bumps FyneApp.toml's internal Build counter as a
	@# side effect -- discard that so tagging happens on a clean tree
	@# matching what was already committed.
	git checkout -- FyneApp.toml
	git tag $(VERSION)
	env -u GH_TOKEN git push origin $(VERSION)
	env -u GH_TOKEN gh release create $(VERSION) Dunzo-$(VERSION)-macos.zip --title "$(VERSION)" --generate-notes
