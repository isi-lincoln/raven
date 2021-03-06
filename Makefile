all: \
	build/rvn

build/rvn: rvn-cli/rvn.go rvn/*.go| build
	CGO_LDFLAGS="-L /usr/local/lib" go build -o build/rvn rvn-cli/rvn.go

build:
	mkdir build

clean:
	rm -rf build

install: build/rvn
	sudo cp build/rvn /usr/local/bin/rvn
