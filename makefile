TIMESTAMP:=$(shell date -u +%Y%m%d.%H%M%S)
COVER_PROFILE_FILENAME:=$(TEMP)/logfeller-testcov.$(TIMESTAMP).out

# Default tests to run as all
# can be used like the following `make test T=./<PACKAGENAME>` command to run
# tests for a specific package.
# can also be run like the following `make test T=./<PACKAGENAME> -run <TESTNAME>/<SUBTESTNAME>`
# to run a subtest in a test in a specific package
T:=./...

.PHONY: test
test:
	go generate ./...
# Normal test with percentages, no report
	go test -race -cover $(T)

.PHONY: lint
lint:
	go generate ./...
	golangci-lint --version
	golangci-lint run

.PHONY: testcov
testcov:
	go generate ./...
# Runs test and generates a report to see coverage
	go test $(T) -race -covermode atomic -coverprofile $(COVER_PROFILE_FILENAME)

.PHONY: showtestcov
showtestcov: testcov
	go tool cover -html=$(COVER_PROFILE_FILENAME)
