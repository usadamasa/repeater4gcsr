package repeater4gcsr

import (
	"fmt"
	"net/url"
)

func transformProtocol(http string) (string, error) {
	u, err := url.Parse(http)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("git@%s:%s.git", u.Host, u.Path), nil
}

func sshCommand(sshFile string) string {
	return fmt.Sprintf("core.sshCommand=\"ssh -i '%s' -o StrictHostKeyChecking=no -F /dev/null\"", sshFile)
}
