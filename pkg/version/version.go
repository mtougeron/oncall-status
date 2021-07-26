package version

import (
	"context"

	"github.com/google/go-github/v37/github"
)

func CheckForNewVersion(currentVersion string) bool {
	latestVersion, err := getLatestVersion()
	if err != nil {
		return false
	}
	return latestVersion != currentVersion
}

func getLatestVersion() (string, error) {
	client := github.NewClient(nil)

	options := &github.ListOptions{}
	releases, _, err := client.Repositories.ListReleases(context.Background(), "mtougeron", "oncall-status", options)
	if err != nil {
		return "", err
	}

	for _, release := range releases {
		if *release.Prerelease {
			continue
		}
		// Return the first release found
		return *release.TagName, nil
	}

	return "", nil
}
