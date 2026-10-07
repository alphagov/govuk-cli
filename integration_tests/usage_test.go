package integration_tests

import (
	"errors"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Usage", func() {
	Context("when executed with no arguments", func() {
		It("exits successfully", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when executed with the version flag", func() {
		It("exits successfully", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred())
		})

		It("prints out the version", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "--version")
			Expect(err).NotTo(HaveOccurred())

			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			Expect(string(output)).To(ContainSubstring("govuk-cli version dev"))
		})
	})

	Context("when executed with an invalid argument", func() {
		It("exits unsuccessfully", func(ctx SpecContext) {
			cmd, err := cliCmd(ctx, "--wibble")
			Expect(err).NotTo(HaveOccurred())

			err = cmd.Run()
			Expect(err).To(HaveOccurred())

			exitErr, ok := errors.AsType[*exec.ExitError](err)
			Expect(ok).To(BeTrueBecause("The reason the command failed was that the govuk cli binary exited non-zero"))

			Expect(exitErr.ExitCode()).NotTo(BeZero())
		})
	})
})
