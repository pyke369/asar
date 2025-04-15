#!/bin/sh

PROGNAME=asar

# build targets
$(PROGNAME): *.go
	@env GOPATH=/tmp/go go get && env GOPATH=/tmp/go CGO_ENABLED=0 go build -trimpath -o $(PROGNAME)
	@-strip $(PROGNAME) 2>/dev/null || true
	@-#upx -9 $(PROGNAME) 2>/dev/null || true
distclean:
	@rm -rf $(PROGNAME) *.upx _in.extract

# run targets
list: $(PROGNAME)
	@./$(PROGNAME) list _in.asar
extract: $(PROGNAME)
	@./$(PROGNAME) extract _in.asar _in.extract
