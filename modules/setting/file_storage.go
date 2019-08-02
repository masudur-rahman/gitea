package setting

import (
	"os"

	"github.com/davecgh/go-spew/spew"
)

// FileStorage represents where to save avatars
var FileStorage struct {
	Bucket string
}

func newFileStorage() {
	sec := Cfg.Section("storage")
	cwd, _ := os.Getwd()
	FileStorage.Bucket = sec.Key("BUCKET").MustString("file://" + cwd) // Preferred: "gs://<bucket-name>?required_key1=required_value1&rq_k2=rq_v2"
	// Default Credential path for GoogleStorage => $HOME/.config/gcloud/application_default_credentials.json
	spew.Dump(FileStorage.Bucket)
}
