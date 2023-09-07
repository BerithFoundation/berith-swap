build:
	go build -o run_test/berith-swap

runtest:
	cd run_test && ./berith-swap

.PHONY: build runtest