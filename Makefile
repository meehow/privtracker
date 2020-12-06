build:
	go build -ldflags="-s -w" -trimpath

deploy: build
	rsync -avzL web privtracker privtracker:
