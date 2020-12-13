package repeater4gcsr

import (
	"testing"
)

func TestWebhook(t *testing.T) {
	_, _ = cloneFromOrigin("https://bitbucket.org/usadamasa/repeater4gcsr", "develop")
}

func TestRemotePush(t *testing.T) {
	repo, err := cloneFromOrigin("https://bitbucket.org/usadamasa/repeater4gcsr", "develop")
	if err != nil {
		t.Fail()
	}
	err = push(repo)
	if err != nil {
		t.Fail()
	}
}
