package jobrequest_test

import (
	"github.com/alphagov/govuk-cli/internal/jobrequest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseAssumedRoleArn", func() {
	It("can parse a valid assumed-role ARN", func() {
		validArn := "arn:aws:sts::123456789:assumed-role/some.one-readonly/staging-readonly"

		userIdentity, err := jobrequest.ParseAssumedRoleArn(validArn)

		Expect(err).To(BeNil())

		Expect(userIdentity.UserName).To(Equal("some.one"))
		Expect(userIdentity.RoleName).To(Equal("readonly"))
		Expect(userIdentity.AccountId).To(Equal("123456789"))
	})

	It("errors when the ARN isn't of the correct length", func() {
		tooLongArn := "arn:aws:sts::123456789:assumed-role/some.one-readonly/staging-readonly:something:else"
		tooShortArn := "arn:aws:sts::123456789"

		_, err := jobrequest.ParseAssumedRoleArn(tooLongArn)
		Expect(err).To(MatchError("malformed ARN: does not contain 6 parts"))

		_, err = jobrequest.ParseAssumedRoleArn(tooShortArn)
		Expect(err).To(MatchError("malformed ARN: does not contain 6 parts"))
	})

	It("errors when the input isn't an ARN", func() {
		notAnArn := "nra:swa:stb::123456789:assumed-role/some.one-readonly/staging-readonly"
		_, err := jobrequest.ParseAssumedRoleArn(notAnArn)
		Expect(err).To(MatchError("invalid ARN 'nra:swa:stb::123456789:assumed-role/some.one-readonly/staging-readonly': expected to start with 'arn:aws:sts', got 'nra:swa:stb'"))

		randomString := "awdjiodas82fg"
		_, err = jobrequest.ParseAssumedRoleArn(randomString)
		Expect(err).To(MatchError("malformed ARN: does not contain 6 parts"))
	})

	It("errors when the resource part isn't of the right length", func() {
		badResourcePart := "arn:aws:sts::123456789:assumed-role/some.one-readonly/staging-readonly/something-else"
		_, err := jobrequest.ParseAssumedRoleArn(badResourcePart)
		Expect(err).To(MatchError("malformed resource part 'assumed-role/some.one-readonly/staging-readonly/something-else' in ARN 'arn:aws:sts::123456789:assumed-role/some.one-readonly/staging-readonly/something-else': resource part not of correct length; expected 3, got 4"))
	})

	It("errors when the resource is not an assumed-role", func() {
		notAssumedRole := "arn:aws:sts::123456789:not-the-right-thing/some.one-readonly/staging-readonly"
		_, err := jobrequest.ParseAssumedRoleArn(notAssumedRole)
		Expect(err).To(MatchError("invalid ARN 'arn:aws:sts::123456789:not-the-right-thing/some.one-readonly/staging-readonly': expected assume-role, got 'not-the-right-thing'"))
	})

	It("errors when the user part is malformed", func() {
		badUserPart := "arn:aws:sts::123456789:assumed-role/some.one-readonly-else/staging-readonly"
		_, err := jobrequest.ParseAssumedRoleArn(badUserPart)
		Expect(err).To(MatchError("malformed user part in ARN 'arn:aws:sts::123456789:assumed-role/some.one-readonly-else/staging-readonly': some.one-readonly-else"))
	})
})
