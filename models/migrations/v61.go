// Copyright 2018 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	"github.com/go-xorm/xorm"
	"gocloud.dev/blob"
)

func addSizeToAttachment(x *xorm.Engine) error {
	type Attachment struct {
		ID   int64  `xorm:"pk autoincr"`
		UUID string `xorm:"uuid UNIQUE"`
		Size int64  `xorm:"DEFAULT 0"`
	}
	if err := x.Sync2(new(Attachment)); err != nil {
		return fmt.Errorf("Sync2: %v", err)
	}

	attachments := make([]Attachment, 0, 100)
	if err := x.Find(&attachments); err != nil {
		return fmt.Errorf("query attachments: %v", err)
	}

	var bucket *blob.Bucket
	var err error
	ctx := context.Background()
	if filepath.IsAbs(setting.AttachmentPath) {
		if err := os.MkdirAll(setting.AttachmentPath, 0700); err != nil {
			log.Fatal("Failed to create '%s': %v", setting.AttachmentPath, err)
		}
		bucket, err = blob.OpenBucket(ctx, "file://"+setting.AttachmentPath)
		if err != nil {
			return fmt.Errorf("could not open bucket: %v", err)
		}
	} else {
		bucket, err = blob.OpenBucket(ctx, setting.FileStorage.BucketURL)
		if err != nil {
			return fmt.Errorf("could not open bucket: %v", err)
		}
		bucket = blob.PrefixedBucket(bucket, setting.AttachmentPath)
	}
	defer bucket.Close()

	for _, attach := range attachments {
		basePath := path.Join(attach.UUID[0:1], attach.UUID[1:2], attach.UUID)
		attrs, err := bucket.Attributes(ctx, basePath)
		if err != nil {
			log.Error("calculate file size of attachment[UUID: %s]: %v", attach.UUID, err)
			continue
		}
		attach.Size = attrs.Size
		if _, err := x.ID(attach.ID).Cols("size").Update(attach); err != nil {
			return fmt.Errorf("update size column: %v", err)
		}
	}
	return nil
}
