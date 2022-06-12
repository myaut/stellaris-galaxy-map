.PHONY: update-apigw
update-apigw:
	./update-yc-function.sh apigw
	
.PHONY: update-upload
update-upload:
	./update-yc-function.sh upload
	
.PHONY: update-render
update-render:
	./update-yc-function.sh render

.PHONY: build
build:
	mkdir -p web/cli
	GOOS=linux go build -o ./web/cli/stellaris-galaxy-map-linux ./cmd/stellaris-galaxy-map-cli
	GOOS=windows go build -o ./web/cli/stellaris-galaxy-map.exe ./cmd/stellaris-galaxy-map-cli
	GOOS=darwin go build -o ./web/cli/stellaris-galaxy-map-macos ./cmd/stellaris-galaxy-map-cli
