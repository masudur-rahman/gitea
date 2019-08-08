package setting

import (
	"os"

	"code.gitea.io/gitea/modules/log"
)

// FileStorage represents where to save avatars, attachments
var FileStorage struct {
	BucketURL string
}

func newFileStorage() {
	sec := Cfg.Section("storage")
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("fails to detect current working dir, err:%v", err)
	}
	// Preferred: "gs://<bucket-name>
	// Default Credential path for GoogleStorage => $HOME/.config/gcloud/application_default_credentials.json
	FileStorage.BucketURL = sec.Key("BUCKET_URL").MustString("file://" + cwd)
}
