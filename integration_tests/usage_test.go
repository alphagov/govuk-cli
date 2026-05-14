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

	Context("when executed with the version flag", func() {
		It("exits successfully", func() {
			cmd, err := cliCmd(context.Background())
			Expect(err).NotTo(HaveOccurred())

			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred())
		})

		It("prints out the version", func() {
			cmd, err := cliCmd(context.Background(), "--version")
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("govuk version dev"))
		})
	})

	Context("when executed with an invalid argument", func() {
		It("exits unsuccessfully", func() {
			cmd, err := cliCmd(context.Background(), "--wibble")
			Expect(err).NotTo(HaveOccurred())

			err = cmd.Run()
			Expect(err).To(HaveOccurred())

			exitErr, ok := errors.AsType[*exec.ExitError](err)
			Expect(ok).To(BeTrueBecause("The reason the command failed was that the govuk cli binary exited non-zero"))

			Expect(exitErr.ExitCode()).NotTo(BeZero())
		})
	})
})
