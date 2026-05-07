go_files := $(shell find . -type f -name '*.go') go.mod go.sum

govuk: ${go_files}
	go build -o govuk main.go
