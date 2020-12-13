package repeater4gcsr

import (
	"fmt"
	"net/url"

	"github.com/go-git/go-git/v5/config"
)

func transformProtocol(http string) (string, error) {
	u, err := url.Parse(http)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("git@%s:%s.git", u.Host, u.Path), nil
}

func makeGCSRRemoteConfig() *config.RemoteConfig {
	remoteUrl := fmt.Sprintf("%s/p/%s/r/%s", GCSR_HOSTNAME, projectName, "repeater4gcsr")
	sugar.Infof("gcsr remote url: %s", remoteUrl)
	return &config.RemoteConfig{
		Name: "gcsr",
		URLs: []string{remoteUrl},
	}
}
