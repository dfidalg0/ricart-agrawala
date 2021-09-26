all: process shared

process: bin/process

shared: bin/shared

bin/process: process/main.go process/go.mod process/**/*.go | bin
	cd ./process && go build -o ../bin/ && cd ..

bin/shared: shared/main.go shared/go.mod | bin
	cd ./shared && go build -o ../bin/ && cd ..

bin:
	mkdir bin

clean:
	rm -rf bin
