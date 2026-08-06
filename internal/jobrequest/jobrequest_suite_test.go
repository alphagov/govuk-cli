package jobrequest_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestJobRequest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "JobRequest Suite")
}
