build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath

deploy: build
	rsync -avzL docs privtracker privtracker:web/
