package repeater4gcsr

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"gopkg.in/go-playground/webhooks.v5/bitbucket"
)

func mirroring(projectName string, payload *bitbucket.RepoPushPayload) error {
	var err error = nil
	tempDir, err := cloneFromGCSR(payload.Repository.Name)
	if err != nil {
		sugar.Errorf("cloneFromGCSR error %s", err)
		return err
	}
	sugar.Debugf("cloneFromGCSR tempDir:%s", tempDir)

	url := payload.Repository.Links.HTML.Href
	sshUrl, _ := transformProtocol(url)

	// get credentials
	sshFile, err := storePrivateKeyFile(projectName)
	if err != nil {
		return err
	}
	sugar.Debugf("storePrivateKeyFile %s", sshFile)

	err = fetchBitbucket(tempDir, sshFile, sshUrl)
	if err != nil {
		sugar.Errorf("fetchBitbucket error %s", err)
		return err
	}
	err = pushAll(tempDir, sshFile)
	if err != nil {
		sugar.Errorf("pushAll error %s", err)
		return err
	}
	sugar.Debug("success")

	return nil
}

func cloneFromGCSR(repoName string) (string, error) {
	tempDir, _ := ioutil.TempDir(os.TempDir(), repoName)
	cmd := exec.Command("gcloud", "source", "repos", "clone", repoName, tempDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	sugar.Info(string(out))
	return tempDir, nil
}

func fetchBitbucket(localDir string, sshFile string, sshUrl string) error {
	// git remote add --mirror=fetch from_repo {repo_url}
	cmd := exec.Command(
		"git",
		"remote",
		"add",
		"--mirror=fetch",
		"from_repo",
		sshUrl)
	cmd.Dir = localDir
	sugar.Debug(cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	sugar.Info(string(out))

	gitCmd := fmt.Sprintf("git -c %s fetch --prune --prune-tags --update-head-ok from_repo", sshCommand(sshFile))
	cmd = exec.Command("sh", "-c", gitCmd)
	cmd.Dir = localDir
	sugar.Debug(cmd.Args)
	out, err = cmd.CombinedOutput()
	if err != nil {
		sugar.Errorf("prune failed %s", out)
		return err
	}
	sugar.Info(string(out))

	return nil
}

func pushAll(localDir string, sshFile string) error {
	gitCmd := fmt.Sprintf(
		"git -c %s push --all -f origin", sshCommand(sshFile))
	cmd := exec.Command("sh", "-c", gitCmd)
	cmd.Dir = localDir
	sugar.Debug(cmd.Args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		sugar.Errorf("push branches failed %s", out)
		return err
	}
	sugar.Info(string(out))

	gitCmd = fmt.Sprintf(
		"git -c %s push --tags -f origin", sshCommand(sshFile))
	cmd = exec.Command("sh", "-c", gitCmd)
	cmd.Dir = localDir
	sugar.Debug(cmd.Args)
	out, err = cmd.CombinedOutput()
	if err != nil {
		sugar.Errorf("push tags failed %s", out)
		return err
	}
	sugar.Info(string(out))
	return nil
}

func storePrivateKeyFile(projectName string) (string, error) {
	privateKey, err := getPrivateKey(projectName, "repeater4gcsr-bitbucket-key")
	if err != nil {
		return "", err
	}
	tempFile, _ := ioutil.TempFile(os.TempDir(), "repeater4gcsr-bitbucket-key")
	defer tempFile.Close()
	b := []byte(privateKey)
	_, err = tempFile.Write(b)
	if err != nil {
		return "", err
	}
	return tempFile.Name(), nil
}
