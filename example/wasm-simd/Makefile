GO = ../../bin/go

simd.o: simd.s $(GO) Makefile
	GOOS=js GOARCH=wasm $(GO) tool asm -debug -S -o $@ $<

.PHONY: bench
bench:
	GOOS=js GOARCH=wasm $(GO) test \
		-exec=./go_js_wasm_exec \
		-bench=. -test.benchtime=0.1s

.PHONY: vet
vet:
	GOOS=js GOARCH=wasm $(GO) vet .

