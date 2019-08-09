// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"code.gitea.io/gitea/modules/setting"
	"gocloud.dev/blob"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/timeutil"

	"github.com/Unknwon/com"
)

//NoticeType describes the notice type
type NoticeType int

const (
	//NoticeRepository type
	NoticeRepository NoticeType = iota + 1
)

// Notice represents a system notice for admin.
type Notice struct {
	ID          int64 `xorm:"pk autoincr"`
	Type        NoticeType
	Description string             `xorm:"TEXT"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
}

// TrStr returns a translation format string.
func (n *Notice) TrStr() string {
	return "admin.notices.type_" + com.ToStr(n.Type)
}

// CreateNotice creates new system notice.
func CreateNotice(tp NoticeType, desc string) error {
	return createNotice(x, tp, desc)
}

func createNotice(e Engine, tp NoticeType, desc string) error {
	n := &Notice{
		Type:        tp,
		Description: desc,
	}
	_, err := e.Insert(n)
	return err
}

// CreateRepositoryNotice creates new system notice with type NoticeRepository.
func CreateRepositoryNotice(desc string) error {
	return createNotice(x, NoticeRepository, desc)
}

// RemoveAllWithNotice removes all directories in given path and
// creates a system notice when error occurs.
func RemoveAllWithNotice(title, path string) {
	removeAllWithNotice(x, title, path)
}

func removeAllFromBucket(bucketPath, objKey string) error {
	var bucket *blob.Bucket
	var err error
	ctx := context.Background()
	if filepath.IsAbs(bucketPath) {
		if err := os.MkdirAll(bucketPath, 0700); err != nil {
			log.Fatal("Failed to create '%s': %v", bucketPath, err)
		}
		bucket, err = blob.OpenBucket(ctx, "file://"+bucketPath)
		if err != nil {
			return err
		}
	} else {
		bucket, err = blob.OpenBucket(ctx, setting.FileStorage.BucketURL)
		if err != nil {
			return err
		}
		bucket = blob.PrefixedBucket(bucket, setting.AttachmentPath)
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

func removeAllWithNotice(e Engine, title, path string) {
	if err := os.RemoveAll(path); err != nil {
		desc := fmt.Sprintf("%s [%s]: %v", title, path, err)
		log.Warn(title+" [%s]: %v", path, err)
		if err = createNotice(e, NoticeRepository, desc); err != nil {
			log.Error("CreateRepositoryNotice: %v", err)
		}
	}
}

// CountNotices returns number of notices.
func CountNotices() int64 {
	count, _ := x.Count(new(Notice))
	return count
}

// Notices returns notices in given page.
func Notices(page, pageSize int) ([]*Notice, error) {
	notices := make([]*Notice, 0, pageSize)
	return notices, x.
		Limit(pageSize, (page-1)*pageSize).
		Desc("id").
		Find(&notices)
}

// DeleteNotice deletes a system notice by given ID.
func DeleteNotice(id int64) error {
	_, err := x.ID(id).Delete(new(Notice))
	return err
}

// DeleteNotices deletes all notices with ID from start to end (inclusive).
func DeleteNotices(start, end int64) error {
	sess := x.Where("id >= ?", start)
	if end > 0 {
		sess.And("id <= ?", end)
	}
	_, err := sess.Delete(new(Notice))
	return err
}

// DeleteNoticesByIDs deletes notices by given IDs.
func DeleteNoticesByIDs(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := x.
		In("id", ids).
		Delete(new(Notice))
	return err
}
