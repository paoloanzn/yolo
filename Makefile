BINARY := yolo
SRC := $(wildcard *.go)

# Discover user-local bin: prefer ~/.local/bin (XDG), fall back to ~/bin
ifeq ($(shell uname),Darwin)
  INSTALL_DIR := $(HOME)/.local/bin
else ifeq ($(shell uname),Linux)
  INSTALL_DIR := $(HOME)/.local/bin
else
  INSTALL_DIR := $(HOME)/bin
endif

.PHONY: build install uninstall clean init

build: $(BINARY)

$(BINARY): $(SRC) go.mod go.sum
	go build -o $(BINARY) .

install: $(BINARY)
	@mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"
	@echo $(PATH) | tr ':' '\n' | grep -qx "$(INSTALL_DIR)" \
		|| echo "Warning: $(INSTALL_DIR) is not in your PATH — add it to your shell rc"

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY)
	@echo "Removed $(INSTALL_DIR)/$(BINARY)"

clean:
	rm -f $(BINARY)

init: build
	./$(BINARY) init
