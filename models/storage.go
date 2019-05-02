package models

import (
	"context"
	"path/filepath"

	"github.com/pkg/errors"

	"gopkg.in/ini.v1"

	"bytes"
	"fmt"
	"image"
	"image/png"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

// bucketStorage
var (
	Bucket    string
	BucketURL string
)

func init() {
	LoadStorageConfigs()
}

func LoadStorageConfigs() {
	Cfg := ini.Empty()
	sec := Cfg.Section("storage")
	Bucket = sec.Key("BUCKET").MustString("gs://gitea-appscode")
	BucketURL = sec.Key("BUCKET_URL").MustString("https://storage.googleapis.com/gitea-appscode")

	// Default Credential path for GoogleStorage => $HOME/.config/gcloud/application_default_credentials.json
}

func (u *User) GetAvatarLinkFromBucket() (string, error) {
	ctx := context.Background()

	bucket, err := blob.OpenBucket(ctx, Bucket)
	if err != nil {
		return "", fmt.Errorf("Failed to setup bucket: %v", err)
	}
	exist, err := bucket.Exists(ctx, u.CustomAvatarPath())
	if exist {
		return filepath.Join(BucketURL, u.CustomAvatarPath()), nil
	}
	return "", errors.Errorf("file doesn't exist, error %v", err)

}

func (u *User) UploadAvatarToBucket(img image.Image) error {
	ctx := context.Background()
	bucket, err := blob.OpenBucket(ctx, Bucket)
	if err != nil {
		return fmt.Errorf("failed to setup bucket: %v", err)
	}

	buf := new(bytes.Buffer)
	if err = png.Encode(buf, img); err != nil {
		return fmt.Errorf("failed to encode: %v", err)
	}
	imgData := buf.Bytes()

	bucketWriter, err := bucket.NewWriter(ctx, u.CustomAvatarPath(), nil)
	if err != nil {
		return fmt.Errorf("failed to obtain writer: %v", err)
	}

	if _, err = bucketWriter.Write(imgData); err != nil {
		return fmt.Errorf("error occured: %v", err)
	}
	if err = bucketWriter.Close(); err != nil {
		return fmt.Errorf("Failed to close: %v", err)
	}

	return nil
}

func (u *User) DeleteAvatarFromBucket() error {
	ctx := context.Background()
	bucket, err := blob.OpenBucket(ctx, Bucket)
	if err != nil {
		return fmt.Errorf("failed to setup bucket: %v", err)
	}
	exist, err := bucket.Exists(ctx, u.CustomAvatarPath())
	if err != nil {
		return err
	} else if !exist {
		return errors.New("avatar not found")
	}

	return bucket.Delete(ctx, u.CustomAvatarPath())
}
