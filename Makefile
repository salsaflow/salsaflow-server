install:
	go install

run:
	@salsaflow-server

deps.fetch:
	go get -d -u github.com/codegangsta/negroni
