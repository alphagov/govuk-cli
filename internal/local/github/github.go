package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

type CacheEntry struct {
	Tag       string `json:"tag"`
	UpdatedAt int64  `json:"updatedAt"`
}

type GitHubVersionsMetadataContainer struct {
	Tags []string `json:"tags"`
}

type GitHubVersionsMetadata struct {
	Container GitHubVersionsMetadataContainer `json:"container"`
}

type GitHubVersionsResponse struct {
	Metadata GitHubVersionsMetadata `json:"metadata"`
}

var cache map[string]CacheEntry

func initCache() error {
	if cache != nil {
		return nil
	}

	cache = make(map[string]CacheEntry)

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	localCacheDir := path.Join(cacheDir, "govuk-local")
	cacheFilePath := path.Join(localCacheDir, "tags.json")

	err = os.Mkdir(localCacheDir, 0700)
	if err != nil && !os.IsExist(err) {
		return err
	}

	_, err = os.Stat(cacheFilePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	cacheFile, err := os.ReadFile(cacheFilePath)

	return json.Unmarshal(cacheFile, &cache)
}

func writeCache() error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	cacheFilePath := path.Join(cacheDir, "govuk-local", "tags.json")

	cacheFileData, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFilePath, cacheFileData, 0600)
}

func githubFindLatestTag(githubToken string, orgName string, imageName string) (string, error) {
	client := &http.Client{}

	url := fmt.Sprintf(
		"https://api.github.com/orgs/%s/packages/container/%s/versions",
		orgName,
		url.QueryEscape(imageName),
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", githubToken))
	req.Header.Add("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Add("Accept", "application/vnd.github+json")

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		return "", fmt.Errorf("github API error: %v", body)
	}

	versions := []GitHubVersionsResponse{}
	err = json.Unmarshal(body, &versions)
	if err != nil {
		return "", err
	}

	for _, version := range versions {
		for _, tag := range version.Metadata.Container.Tags {
			if strings.HasPrefix(tag, "v") {
				return tag, nil
			}
		}
	}

	return "", fmt.Errorf("version tag not found for package %s/%s", orgName, imageName)
}

func GetImageTag(githubToken string, orgName string, imageName string) (string, error) {
	err := initCache()
	if err != nil {
		return "", err
	}

	cacheKey := fmt.Sprintf("%s/%s", orgName, imageName)

	cacheValue, ok := cache[cacheKey]
	if ok {
		return cacheValue.Tag, nil
	}

	imageTag, err := githubFindLatestTag(githubToken, orgName, imageName)
	if err != nil {
		return "", err
	}

	cache[cacheKey] = CacheEntry{
		Tag:       imageTag,
		UpdatedAt: time.Now().Unix(),
	}
	writeCache()

	return imageTag, nil
}
