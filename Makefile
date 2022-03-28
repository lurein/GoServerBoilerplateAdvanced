TAG ?= latest

.DEFAULT_GOAL = build

gen:
	go generate ./...

build: gen
	go vet
	go build -o whimsy

test: gen
	go test ./...

swagger:
	swagger generate spec | jq '.definitions.Decimal.type="string"' > static/swagger.json
	swagger validate static/swagger.json

image: build
	docker build . -t givecard-platform/whimsy:$(TAG)