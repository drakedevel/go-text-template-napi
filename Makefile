.PHONY: default
default:
	go build -buildmode=c-shared -o binding.node .


.PHONY: fmt
fmt:
	go fmt .
