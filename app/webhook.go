package repeater4gcsr

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"go.uber.org/zap"
	"golang.org/x/oauth2/google"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"gopkg.in/go-playground/webhooks.v5/bitbucket"
)

const (
	GCSR_HOSTNAME string = "source.developers.google.com:2022"
)

var (
	logger *zap.Logger
	sugar  *zap.SugaredLogger

	projectName string

	// TODO: gcsrの認証をgcloudでやりたい
	gcsrSshUser = os.Getenv("GCSR_SSH_KEY_USER")
)

func init() {
	logger, _ = zap.NewDevelopment()
	defer logger.Sync()
	sugar = logger.Sugar()

	cred, err := google.FindDefaultCredentials(context.Background())
	if err != nil {
		sugar.Fatalf("default credentials not found")
	}
	if cred == nil {
		sugar.Fatalf("default credentials not found")
	}
	projectName = cred.ProjectID
	sugar.Debugf("GCP_PROJECT : %s", projectName)


	if gcsrSshUser == "" {
		sugar.Fatalf("not set env GCSR_SSH_KEY_USER")
	}
	sugar.Debugf("GCSR_SSH_KEY_USER=%s", gcsrSshUser)
}

func Webhook(w http.ResponseWriter, r *http.Request) {
	ip, err := getMyIp()
	if err != nil {
		_, _ = fmt.Fprintf(w, "%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, "500 Internal Error", http.StatusInternalServerError)
		return
	}
	sugar.Debugf("ip:%s", ip)

	if r.Method != http.MethodPost {
		http.Error(w, "405 - Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "400 - Unhandled Content-Type", http.StatusBadRequest)
		return
	}

	// https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/
	bb, _ := bitbucket.New()
	payload, err := bb.Parse(r, bitbucket.RepoPushEvent)
	if err != nil {
		_, _ = fmt.Fprintf(w, "bb.Parse %s\n", err)
		http.Error(w, "400 - Unhandled Json body", http.StatusBadRequest)
		return
	}

	pushEvent := payload.(bitbucket.RepoPushPayload)
	sugar.Debugf("repository name:%v", pushEvent.Repository.Links.HTML.Href)
	err = cloneAndPush(&pushEvent)
	if err != nil {
		_, _ = fmt.Fprintf(w, "cloneFromOrigin %s\n", err)
		http.Error(w, "500 Internal Error", http.StatusInternalServerError)
		return
	}

	sugar.Infof("webhook success!")
	w.WriteHeader(http.StatusOK)
}

func getMyIp() ([]byte, error) {
	pingUrl := "https://api.ipify.org?format=text"
	resp, err := http.Get(pingUrl)
	if err != nil {
		sugar.Errorf("http.Get failed: %s, %s", pingUrl, err)
		return nil, err
	}
	defer resp.Body.Close()
	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		sugar.Errorf("read body failed: %s", err)
		return nil, err
	}
	return ip, nil
}

func cloneAndPush(payload *bitbucket.RepoPushPayload) error {
	for i, change := range payload.Push.Changes {
		if change.Closed {
			// deleted branch
			sugar.Infof("%s %s is deleted", change.Old.Type, change.Old.Name)
			err := deleteBranch(change.Old.Name)
			if err != nil {
				sugar.Errorf("deleteBranch: %s, %s\n", change.Old.Name, err)
				return err
			}
			continue
		}

		sugar.Debugf("changes[%d] type:%v", i, change.New.Type)
		sugar.Debugf("changes[%d] name:%v", i, change.New.Name)

		switch change.New.Type {
		case "branch":
			repo, err := cloneFromOrigin(payload.Repository.Links.HTML.Href, change.New.Name)
			if err != nil {
				sugar.Errorf("git.Clone %s\n", err)
				return err
			}
			sugar.Debugf("clone repo: %s", payload.Repository.Links.HTML.Href)
			err = push(repo)
			if err != nil {
				sugar.Errorf("git.push %s\n", err)
				return err
			}
		default:
			sugar.Debugf("Nop %s", change.New.Type)
		}
	}
	return nil
}

func cloneFromOrigin(url string, branch string) (*git.Repository, error) {
	auth, err := GetBitBcuketAuth(projectName)
	if err != nil {
		sugar.Errorf("GetBitBcuketAuth repeater4gcsr-bitbucket-key %s", err)
		return nil, err
	}

	sshUrl, _ := transformProtocol(url)
	sugar.Debugf("start clone repo: %s, git url: %s", url, sshUrl)
	// https://dev.classmethod.jp/articles/in-memory-git-commit-and-push/
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:           sshUrl,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch)),
		Auth:          auth,
	})
	if err != nil {
		sugar.Errorf("git.Clone %s\n", err)
		return nil, err
	}

	sugar.Debugf("clone repo: %s", url)
	head, _ := repo.Head()
	sugar.Debugf("branch name:%s, hash:%s", head.Name(), head.Hash())
	return repo, nil
}

func push(repo *git.Repository) error {
	remote, err := repo.CreateRemote(makeGCSRRemoteConfig())
	if err != nil {
		sugar.Errorf("repo.CreateRemote %s", err)
		return err
	}
	sugar.Debugf("add remote %s", remote.Config().Fetch[0])

	auth, err := GetGCSRAuth(projectName, gcsrSshUser)
	if err != nil {
		sugar.Errorf("GetGCSRAuth repeater4gcsr-gcsr-key %s", err)
		return err
	}

	err = remote.Push(&git.PushOptions{
		RemoteName: "gcsr",
		Force:      true,
		Auth:       auth,
	})
	if err != nil {
		message := fmt.Sprintf("%s", err)
		if message == "already up-to-date" {
			sugar.Debugf(message)
			return nil
		}
		sugar.Errorf("remote.push %s", err)
		return err
	}
	return nil
}

func deleteBranch(branchShortName string) error {
	auth, err := GetGCSRAuth(projectName, gcsrSshUser)
	if err != nil {
		sugar.Errorf("GetGCSRAuth repeater4gcsr-gcsr-key %s", err)
		return err
	}
	gcsr := git.NewRemote(memory.NewStorage(), makeGCSRRemoteConfig())
	pushConfig := &git.PushOptions{
		RemoteName: "gcsr",
		RefSpecs:   []config.RefSpec{config.RefSpec(":refs/heads/" + branchShortName)},
		Auth:       auth,
	}
	err = gcsr.Push(pushConfig)

	return err
}
