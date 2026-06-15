package build

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
	"github.com/sirupsen/logrus"

	l "github.com/alphagov/govuk-cli/internal/logger"
)

func BuildContainerImage(appDir string, appName string, registryHost string, imageTag string) error {
	l.Log().WithFields(logrus.Fields{
		"dir":      appDir,
		"name":     appName,
		"registry": registryHost,
	}).Debug("building container image")

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return err
	}
	defer cli.Close()

	buildCtx, err := archive.TarWithOptions(
		appDir,
		&archive.TarOptions{},
	)
	if err != nil {
		return err
	}
	defer buildCtx.Close()

	tag := fmt.Sprintf("%s/alphagov/govuk/%s:%s", registryHost, appName, imageTag)

	resp, err := cli.ImageBuild(
		context.Background(),
		buildCtx,
		types.ImageBuildOptions{
			Tags:       []string{tag},
			Dockerfile: "Dockerfile",
			Remove:     true,
		},
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fd, isTerm := term.GetFdInfo(os.Stderr)
	err = jsonmessage.DisplayJSONMessagesStream(
		resp.Body,
		os.Stderr,
		fd,
		isTerm,
		nil,
	)
	if err != nil {
		return err
	}

	pushResp, err := cli.ImagePush(context.Background(), tag, image.PushOptions{})
	if err != nil {
		return err
	}
	defer pushResp.Close()

	fd, isTerm = term.GetFdInfo(os.Stderr)
	err = jsonmessage.DisplayJSONMessagesStream(
		pushResp,
		os.Stderr,
		fd,
		isTerm,
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}
