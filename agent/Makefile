# dwm-agent — terminal agent that manages the X session over dwm's IPC socket.

PREFIX ?= /usr/local
# Must match the binary aiscratchpadcmd launches in config.single.h. When these
# drifted apart, every "make install" succeeded while Alt+a kept running a
# months-stale binary that honestly reported it had no shell tool.
BIN    ?= monty

# The module lives inside a git worktree, where Go's VCS stamping fails with
# "error obtaining VCS status: exit status 128". Stamping buys us nothing here.
GOFLAGS := -buildvcs=false

.PHONY: all build vet test install uninstall clean probe

all: build

build:
	go build $(GOFLAGS) -ldflags='-s -w' -o $(BIN) .

vet:
	go vet ./...

test:
	go test $(GOFLAGS) ./...

probe:
	go build $(GOFLAGS) -o probe ./cmd/probe

install: build
	install -Dm755 $(BIN) $(DESTDIR)$(PREFIX)/bin/$(BIN)

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BIN)

clean:
	rm -f $(BIN) probe
