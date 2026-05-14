package integration_tests

import (
	"context"
	"errors"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Usage", func() {
	Context("when executed with no arguments", func() {
		It("exits successfully", func() {
			cmd, err := cliCmd(context.Background())
			Expect(err).NotTo(HaveOccurred())

			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when executed with an invalid argument", func() {
		It("Exist unsuccessfully", func() {
			cmd, err := cliCmd(context.Background())
			Expect(err).NotTo(HaveOccurred())

			cmd.Args = append(cmd.Args, "--wibble")
			err = cmd.Run()
			Expect(err).To(HaveOccurred())

			exitErr, ok := errors.AsType[*exec.ExitError](err)
			Expect(ok).To(BeTrueBecause("The reason the command failed was that the govuk cli binary exited non-zero"))

			Expect(exitErr.ExitCode()).NotTo(BeZero())
		})
	})
})
