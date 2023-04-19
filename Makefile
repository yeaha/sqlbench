.PHONY: benchmark
benchmark:
	@go test -cpu=4,8 -bench=.
