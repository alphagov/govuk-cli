package jobrequest

import (
	"errors"
	"fmt"
	"strings"
)

type UserIdentity struct {
	AccountId string
	UserName  string
	RoleName  string
}

// Pull a username out of an assumed-role ARN
func ParseAssumedRoleArn(arn string) (UserIdentity, error) {
	identity := UserIdentity{}

	split := strings.Split(arn, ":")
	if len(split) != 6 {
		return identity, errors.New("malformed ARN: does not contain 6 parts")
	}
	if split[0] != "arn" || split[1] != "aws" || split[2] != "sts" {
		return identity, fmt.Errorf("invalid ARN '%s': expected to start with 'arn:aws:sts', got '%s:%s:%s'", arn, split[0], split[1], split[2])
	}

	accountId := split[4]
	identity.AccountId = accountId

	resourcePart := split[5]
	resourcePartSplit := strings.Split(resourcePart, "/")

	if len(resourcePartSplit) != 3 {
		return identity, fmt.Errorf("malformed resource part '%s' in ARN '%s': resource part not of correct length; expected 3, got %d", resourcePart, arn, len(resourcePartSplit))
	}

	if resourcePartSplit[0] != "assumed-role" {
		return identity, fmt.Errorf("invalid ARN '%s': expected assume-role, got '%s'", arn, resourcePartSplit[0])
	}

	userPart := resourcePartSplit[1]
	userPartSplit := strings.Split(userPart, "-")
	if len(userPartSplit) != 2 {
		return identity, fmt.Errorf("malformed user part in ARN '%s': %s", arn, userPart)
	}

	userName := userPartSplit[0]
	userPartRoleName := userPartSplit[1]

	identity.UserName = userName
	identity.RoleName = userPartRoleName

	return identity, nil
}
