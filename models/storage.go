package models

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
)

// IsAvatarValid checks if the avatarLink is valid
func IsAvatarValid(avatarUploadPath, objKey string) bool {
	ctx := context.Background()
	bucket, err := setting.OpenBucket(ctx, avatarUploadPath)
	if err != nil {
		log.Error("could not open bucket: %v", err)
		return false
	}
	defer bucket.Close()

	exist, err := bucket.Exists(ctx, objKey)
	if err != nil {
		log.Error(err.Error())
		return false
	}
	return exist
}

// uploadImage uploads avatar to bucket
func uploadImage(avatarUploadPath, objKey string, img image.Image) error {
	ctx := context.Background()
	bucket, err := setting.OpenBucket(ctx, avatarUploadPath)
	if err != nil {
		return fmt.Errorf("could not open bucket: %v", err)
	}
	defer bucket.Close()

	buf := new(bytes.Buffer)
	if err = png.Encode(buf, img); err != nil {
		return fmt.Errorf("failed to encode: %v", err)
	}
	imgData := buf.Bytes()

	bw, err := bucket.NewWriter(ctx, objKey, nil)
	if err != nil {
		return fmt.Errorf("failed to obtain writer: %v", err)
	}

	if _, err = bw.Write(imgData); err != nil {
		return fmt.Errorf("error occurred: %v", err)
	}
	if err = bw.Close(); err != nil {
		return fmt.Errorf("failed to close: %v", err)
	}
	return nil
}

// deleteAvatarFromBucket deletes user or repo avatar from bucket
func deleteAvatarFromBucket(avatarUploadPath, objKey string) error {
	ctx := context.Background()
	bucket, err := setting.OpenBucket(ctx, avatarUploadPath)
	if err != nil {
		return fmt.Errorf("could not open bucket: %v", err)
	}
	defer bucket.Close()

	exist, err := bucket.Exists(ctx, objKey)
	if err != nil {
		return err
	} else if exist {
		return bucket.Delete(ctx, objKey)
	}
	return nil
}
