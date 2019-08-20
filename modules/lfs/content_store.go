package lfs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	"gocloud.dev/blob"
)

var (
	errHashMismatch = errors.New("Content hash does not match OID")
	errSizeMismatch = errors.New("Content size does not match")
)

// ContentStore provides a simple file system based storage.
type ContentStore struct {
	BasePath string
}

// Get takes a Meta object and retrieves the content from the store, returning
// it as an io.Reader. If fromByte > 0, the reader starts from that byte
func (s *ContentStore) Get(meta *models.LFSMetaObject, fromByte int64) (io.ReadCloser, error) {
	var bucket *blob.Bucket
	var err error
	ctx := context.Background()
	if filepath.IsAbs(s.BasePath) {
		if err := os.MkdirAll(s.BasePath, 0700); err != nil {
			log.Fatal("Failed to create '%s': %v", s.BasePath, err)
		}
		bucket, err = blob.OpenBucket(ctx, "file://"+s.BasePath)
		if err != nil {
			return nil, fmt.Errorf("could not open bucket: %v", err)
		}
	} else {
		bucket, err = blob.OpenBucket(ctx, setting.FileStorage.BucketURL)
		if err != nil {
			return nil, fmt.Errorf("could not open bucket: %v", err)
		}
		bucket = blob.PrefixedBucket(bucket, s.BasePath)
	}
	defer bucket.Close()

	reader, err := bucket.NewRangeReader(ctx, transformKey(meta.Oid), fromByte, -1, nil)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

// Put takes a Meta object and an io.Reader and writes the content to the store.
func (s *ContentStore) Put(meta *models.LFSMetaObject, r io.Reader) error {
	var bucket *blob.Bucket
	var err error
	ctx := context.Background()
	if filepath.IsAbs(s.BasePath) {
		if err := os.MkdirAll(s.BasePath, 0700); err != nil {
			log.Fatal("Failed to create '%s': %v", s.BasePath, err)
		}
		bucket, err = blob.OpenBucket(ctx, "file://"+s.BasePath)
		if err != nil {
			return fmt.Errorf("could not open bucket: %v", err)
		}
	} else {
		bucket, err = blob.OpenBucket(ctx, setting.FileStorage.BucketURL)
		if err != nil {
			return fmt.Errorf("could not open bucket: %v", err)
		}
		bucket = blob.PrefixedBucket(bucket, s.BasePath)
	}
	defer bucket.Close()

	bw, err := bucket.NewWriter(ctx, transformKey(meta.Oid), nil)
	if err != nil {
		return err
	}

	hash := sha256.New()
	hw := io.MultiWriter(hash, bw)

	written, err := io.Copy(hw, r)
	if err != nil {
		return err
	}

	if written != meta.Size {
		return errSizeMismatch
	}

	shaStr := hex.EncodeToString(hash.Sum(nil))
	if shaStr != meta.Oid {
		return errHashMismatch
	}

	return bw.Close()
}

// Exists returns true if the object exists in the content store.
func (s *ContentStore) Exists(meta *models.LFSMetaObject) bool {
	var bucket *blob.Bucket
	var err error
	ctx := context.Background()
	if filepath.IsAbs(s.BasePath) {
		if err := os.MkdirAll(s.BasePath, 0700); err != nil {
			log.Fatal("Failed to create '%s': %v", s.BasePath, err)
		}
		bucket, err = blob.OpenBucket(ctx, "file://"+s.BasePath)
		if err != nil {
			return false
		}
	} else {
		bucket, err = blob.OpenBucket(ctx, setting.FileStorage.BucketURL)
		if err != nil {
			return false
		}
		bucket = blob.PrefixedBucket(bucket, s.BasePath)
	}
	defer bucket.Close()

	exist, _ := bucket.Exists(ctx, transformKey(meta.Oid))
	return exist
}

// Verify returns true if the object exists in the content store and size is correct.
func (s *ContentStore) Verify(meta *models.LFSMetaObject) (bool, error) {
	var bucket *blob.Bucket
	var err error
	ctx := context.Background()
	if filepath.IsAbs(s.BasePath) {
		if err := os.MkdirAll(s.BasePath, 0700); err != nil {
			log.Fatal("Failed to create '%s': %v", s.BasePath, err)
		}
		bucket, err = blob.OpenBucket(ctx, "file://"+s.BasePath)
		if err != nil {
			return false, fmt.Errorf("could not open bucket: %v", err)
		}
	} else {
		bucket, err = blob.OpenBucket(ctx, setting.FileStorage.BucketURL)
		if err != nil {
			return false, fmt.Errorf("could not open bucket: %v", err)
		}
		bucket = blob.PrefixedBucket(bucket, s.BasePath)
	}
	defer bucket.Close()

	reader, err := bucket.NewReader(ctx, transformKey(meta.Oid), nil)
	if err != nil {
		return false, err
	}
	defer reader.Close()

	if reader.Size() != meta.Size {
		return false, nil
	}

	return true, nil
}

func transformKey(key string) string {
	if len(key) < 5 {
		return key
	}

	return filepath.Join(key[0:2], key[2:4], key[4:])
}
