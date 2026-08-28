.PHONY: run package clean vet

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
