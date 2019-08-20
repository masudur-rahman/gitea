package models

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	"gocloud.dev/blob"
)

// IsAvatarValid checks if the avatarLink is valid
func IsAvatarValid(avatarUploadPath, objKey string) bool {
	var bucket *blob.Bucket
	var err error
	ctx := context.Background()
	if filepath.IsAbs(avatarUploadPath) {
		if err := os.MkdirAll(setting.AvatarUploadPath, 0700); err != nil {
			log.Fatal("Failed to create '%s': %v", setting.AvatarUploadPath, err)
		}
		bucket, err = blob.OpenBucket(ctx, "file://"+avatarUploadPath)
		if err != nil {
			log.Error("could not open bucket: %v", err)
			return false
		}
	} else {
		bucket, err = blob.OpenBucket(ctx, setting.FileStorage.BucketURL)
		if err != nil {
			log.Error("could not open bucket: %v", err)
			return false
		}
		bucket = blob.PrefixedBucket(bucket, avatarUploadPath)
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
	var bucket *blob.Bucket
	var err error
	ctx := context.Background()
	if filepath.IsAbs(avatarUploadPath) {
		if err := os.MkdirAll(avatarUploadPath, 0700); err != nil {
			log.Fatal("Failed to create '%s': %v", avatarUploadPath, err)
		}
		bucket, err = blob.OpenBucket(ctx, "file://"+avatarUploadPath)
		if err != nil {
			return fmt.Errorf("could not open bucket: %v", err)
		}
	} else {
		bucket, err = blob.OpenBucket(ctx, setting.FileStorage.BucketURL)
		if err != nil {
			return fmt.Errorf("could not open bucket: %v", err)
		}
		bucket = blob.PrefixedBucket(bucket, avatarUploadPath)
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
	var bucket *blob.Bucket
	var err error
	ctx := context.Background()
	if filepath.IsAbs(avatarUploadPath) {
		if err := os.MkdirAll(avatarUploadPath, 0700); err != nil {
			log.Fatal("Failed to create '%s': %v", avatarUploadPath, err)
		}
		bucket, err = blob.OpenBucket(ctx, "file://"+avatarUploadPath)
		if err != nil {
			return fmt.Errorf("could not open bucket: %v", err)
		}
	} else {
		bucket, err = blob.OpenBucket(ctx, setting.FileStorage.BucketURL)
		if err != nil {
			return fmt.Errorf("could not open bucket: %v", err)
		}
		bucket = blob.PrefixedBucket(bucket, avatarUploadPath)
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
