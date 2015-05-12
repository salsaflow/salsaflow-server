install:
	go install

run:
	@salsaflow-server

deps.fetch:
	go get -d -u github.com/codegangsta/negroni
	go get -d -u gopkg.in/tylerb/graceful.v1
