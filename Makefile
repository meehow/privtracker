build:
	CGO_ENABLED=0 go build -ldflags="-s -w -buildid=" -trimpath

deploy: build
	rsync -avzL --exclude '*.gz' docs privtracker privtracker:web/

test:
	go test -bench . -benchmem
