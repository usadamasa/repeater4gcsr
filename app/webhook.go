package repeater4gcsr

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"go.uber.org/zap"
	"golang.org/x/oauth2/google"

	"gopkg.in/go-playground/webhooks.v5/bitbucket"
)

type IndexPayload struct {
	Ip string `json:"ip"`
}

var (
	logger *zap.Logger
	sugar  *zap.SugaredLogger

	projectName string
)

func init() {
	logger, _ = zap.NewDevelopment()
	defer logger.Sync()
	sugar = logger.Sugar()

	cred, err := google.FindDefaultCredentials(context.Background())
	if err != nil {
		sugar.Warnf("default credentials not found err:%s", err)
	}
	if cred == nil {
		sugar.Warn("default credentials not found")
	}
	if cred.ProjectID == "" {
		projectName = os.Getenv("GCP_PROJECT")
	} else {
		projectName = cred.ProjectID
	}
	if projectName == "" {
		sugar.Fatalf("ProjectID not found")
	}
	sugar.Debugf("GCP_PROJECT : %s", projectName)
}

func Index(w http.ResponseWriter, r *http.Request) {
	ip, err := getMyIp()
	if err != nil {
		_, _ = fmt.Fprintf(w, "%v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, "500 Internal Error", http.StatusInternalServerError)
		return
	}
	sugar.Debugf("ip:%s", ip)
	w.Header().Set("Content-Type", "application/json")
	index := IndexPayload{
		Ip: string(ip),
	}
	body, _ := json.Marshal(index)
	_, _ = w.Write(body)
	w.WriteHeader(http.StatusOK)
}

func Driver(w http.ResponseWriter, r *http.Request) {
	tempDir, err := cloneFromGCSR("repeater4gcsr")
	if err != nil {
		sugar.Errorf("cloneFromGCSR error %s", err)
	}
	sugar.Debugf("cloneFromGCSR tempDir:%s", tempDir)
	sshFile, err := storePrivateKeyFile(projectName)
	if err != nil {
		return
	}
	sugar.Debugf("storePrivateKeyFile %s", sshFile)

	err = fetchBitbucket(tempDir, sshFile, "git@bitbucket.org:/usadamasa/repeater4gcsr.git")
	if err != nil {
		sugar.Errorf("fetchBitbucket error %s", err)
	}

	err = pushAll(tempDir, sshFile)
	if err != nil {
		sugar.Errorf("pushAll error %s", err)
	}
}

func Webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sugar.Info("405 - Method Not Allowed")
		http.Error(w, "405 - Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		sugar.Info("400 - Unhandled Content-Type")
		http.Error(w, "400 - Unhandled Content-Type", http.StatusBadRequest)
		return
	}

	// https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/
	bb, _ := bitbucket.New()
	payload, err := bb.Parse(r, bitbucket.RepoPushEvent)
	if err != nil {
		sugar.Info("400 - Unhandled Json body")
		_, _ = fmt.Fprintf(w, "bb.Parse %s\n", err)
		http.Error(w, "400 - Unhandled Json body", http.StatusBadRequest)
		return
	}
	pushEvent := payload.(bitbucket.RepoPushPayload)
	sugar.Infof("Receive webhook event from %s", pushEvent.Repository.FullName)
	err = mirroring(projectName, &pushEvent)
	if err != nil {
		_, _ = fmt.Fprintf(w, "mirroring %s\n", err)
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
