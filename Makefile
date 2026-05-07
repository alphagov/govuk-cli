go_files := $(shell git ls-files '*.go') go.mod go.sum

govuk: ${go_files}
	go build -o govuk main.go
