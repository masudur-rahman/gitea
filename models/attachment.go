// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/timeutil"

	"github.com/go-xorm/xorm"
	gouuid "github.com/satori/go.uuid"
	"gocloud.dev/blob"
)

// Attachment represent a attachment of issue/comment/release.
type Attachment struct {
	ID            int64  `xorm:"pk autoincr"`
	UUID          string `xorm:"uuid UNIQUE"`
	IssueID       int64  `xorm:"INDEX"`
	ReleaseID     int64  `xorm:"INDEX"`
	UploaderID    int64  `xorm:"INDEX DEFAULT 0"` // Notice: will be zero before this column added
	CommentID     int64
	Name          string
	DownloadCount int64              `xorm:"DEFAULT 0"`
	Size          int64              `xorm:"DEFAULT 0"`
	CreatedUnix   timeutil.TimeStamp `xorm:"created"`
}

// IncreaseDownloadCount is update download count + 1
func (a *Attachment) IncreaseDownloadCount() error {
	// Update download count.
	if _, err := x.Exec("UPDATE `attachment` SET download_count=download_count+1 WHERE id=?", a.ID); err != nil {
		return fmt.Errorf("increase attachment count: %v", err)
	}

	return nil
}

// APIFormat converts models.Attachment to api.Attachment
func (a *Attachment) APIFormat() *api.Attachment {
	return &api.Attachment{
		ID:            a.ID,
		Name:          a.Name,
		Created:       a.CreatedUnix.AsTime(),
		DownloadCount: a.DownloadCount,
		Size:          a.Size,
		UUID:          a.UUID,
		DownloadURL:   a.DownloadURL(),
	}
}

// AttachmentLocalPath returns where attachment is stored in local file
// system based on given UUID.
func AttachmentLocalPath(uuid string) string {
	return path.Join(setting.AttachmentPath, uuid[0:1], uuid[1:2], uuid)
}

// LocalPath returns where attachment is stored in local file system.
func (a *Attachment) LocalPath() string {
	return AttachmentLocalPath(a.UUID)
}

// AttachmentBasePath returns the file name of attachment
func (a *Attachment) AttachmentBasePath() string {
	return path.Join(a.UUID[0:1], a.UUID[1:2], a.UUID)
}

// DownloadURL returns the download url of the attached file
func (a *Attachment) DownloadURL() string {
	return fmt.Sprintf("%sattachments/%s", setting.AppURL, a.UUID)
}

// UploadToBucket uploads attachments to bucket
func (a *Attachment) UploadToBucket(buf []byte, file io.Reader) (*Attachment, error) {
	var bucket *blob.Bucket
	var err error
	ctx := context.Background()
	if filepath.IsAbs(setting.AttachmentPath) {
		if err := os.MkdirAll(setting.AttachmentPath, 0700); err != nil {
			log.Fatal("Failed to create '%s': %v", setting.AttachmentPath, err)
		}
		bucket, err = blob.OpenBucket(ctx, "file://"+setting.AttachmentPath)
		if err != nil {
			return nil, fmt.Errorf("could not open bucket: %v", err)
		}
	} else {
		bucket, err = blob.OpenBucket(ctx, setting.FileStorage.BucketURL)
		if err != nil {
			return nil, fmt.Errorf("could not open bucket: %v", err)
		}
		bucket = blob.PrefixedBucket(bucket, setting.AttachmentPath)
	}
	defer bucket.Close()

	bw, err := bucket.NewWriter(ctx, a.AttachmentBasePath(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain writer: %v", err)
	}

	if _, err = bw.Write(buf); err != nil {
		return nil, fmt.Errorf("error occurred while writing: %v", err)
	} else if _, err = io.Copy(bw, file); err != nil {
		return nil, fmt.Errorf("error occurred while copying: %v", err)
	}
	if err = bw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close: %v", err)
	}

	attrs, err := bucket.Attributes(ctx, a.AttachmentBasePath())
	if err != nil {
		return nil, fmt.Errorf("failed to read attributes: %v", err)
	}
	a.Size = attrs.Size

	return a, nil
}

// NewAttachment creates a new attachment object.
func NewAttachment(attach *Attachment, buf []byte, file io.Reader) (_ *Attachment, err error) {
	attach.UUID = gouuid.NewV4().String()

	attach, err = attach.UploadToBucket(buf, file)
	if err != nil {
		return nil, err
	}

	if _, err := x.Insert(attach); err != nil {
		return nil, err
	}

	return attach, nil
}

// GetAttachmentByID returns attachment by given id
func GetAttachmentByID(id int64) (*Attachment, error) {
	return getAttachmentByID(x, id)
}

func getAttachmentByID(e Engine, id int64) (*Attachment, error) {
	attach := &Attachment{ID: id}

	if has, err := e.Get(attach); err != nil {
		return nil, err
	} else if !has {
		return nil, ErrAttachmentNotExist{ID: id, UUID: ""}
	}
	return attach, nil
}

func getAttachmentByUUID(e Engine, uuid string) (*Attachment, error) {
	attach := &Attachment{UUID: uuid}
	has, err := e.Get(attach)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrAttachmentNotExist{0, uuid}
	}
	return attach, nil
}

func getAttachmentsByUUIDs(e Engine, uuids []string) ([]*Attachment, error) {
	if len(uuids) == 0 {
		return []*Attachment{}, nil
	}

	// Silently drop invalid uuids.
	attachments := make([]*Attachment, 0, len(uuids))
	return attachments, e.In("uuid", uuids).Find(&attachments)
}

// GetAttachmentByUUID returns attachment by given UUID.
func GetAttachmentByUUID(uuid string) (*Attachment, error) {
	return getAttachmentByUUID(x, uuid)
}

// GetAttachmentByReleaseIDFileName returns attachment by given releaseId and fileName.
func GetAttachmentByReleaseIDFileName(releaseID int64, fileName string) (*Attachment, error) {
	return getAttachmentByReleaseIDFileName(x, releaseID, fileName)
}

func getAttachmentsByIssueID(e Engine, issueID int64) ([]*Attachment, error) {
	attachments := make([]*Attachment, 0, 10)
	return attachments, e.Where("issue_id = ? AND comment_id = 0", issueID).Find(&attachments)
}

// GetAttachmentsByIssueID returns all attachments of an issue.
func GetAttachmentsByIssueID(issueID int64) ([]*Attachment, error) {
	return getAttachmentsByIssueID(x, issueID)
}

// GetAttachmentsByCommentID returns all attachments if comment by given ID.
func GetAttachmentsByCommentID(commentID int64) ([]*Attachment, error) {
	return getAttachmentsByCommentID(x, commentID)
}

func getAttachmentsByCommentID(e Engine, commentID int64) ([]*Attachment, error) {
	attachments := make([]*Attachment, 0, 10)
	return attachments, x.Where("comment_id=?", commentID).Find(&attachments)
}

// getAttachmentByReleaseIDFileName return a file based on the the following infos:
func getAttachmentByReleaseIDFileName(e Engine, releaseID int64, fileName string) (*Attachment, error) {
	attach := &Attachment{ReleaseID: releaseID, Name: fileName}
	has, err := e.Get(attach)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, err
	}
	return attach, nil
}

// Open provides attachment reader from bucket
func (a *Attachment) Open() (io.ReadCloser, error) {
	var bucket *blob.Bucket
	var err error
	ctx := context.Background()
	if filepath.IsAbs(setting.AttachmentPath) {
		if err := os.MkdirAll(setting.AttachmentPath, 0700); err != nil {
			log.Fatal("Failed to create '%s': %v", setting.AttachmentPath, err)
		}
		bucket, err = blob.OpenBucket(ctx, "file://"+setting.AttachmentPath)
		if err != nil {
			return nil, fmt.Errorf("could not open bucket: %v", err)
		}
	} else {
		bucket, err = blob.OpenBucket(ctx, setting.FileStorage.BucketURL)
		if err != nil {
			return nil, fmt.Errorf("could not open bucket: %v", err)
		}
		bucket = blob.PrefixedBucket(bucket, setting.AttachmentPath)
	}
	defer bucket.Close()

	exist, err := bucket.Exists(ctx, a.AttachmentBasePath())
	if err != nil {
		return nil, err
	} else if !exist {
		return nil, os.ErrNotExist
	}

	return bucket.NewReader(ctx, a.AttachmentBasePath(), nil)
}

// deleteFromBucket deletes attachments from bucket
func (a *Attachment) deleteFromBucket() error {
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

	exist, err := bucket.Exists(ctx, a.AttachmentBasePath())
	if err != nil {
		return err
	} else if !exist {
		return os.ErrNotExist
	}

	return bucket.Delete(ctx, a.AttachmentBasePath())
}

// DeleteAttachment deletes the given attachment and optionally the associated file.
func DeleteAttachment(a *Attachment, remove bool) error {
	_, err := DeleteAttachments([]*Attachment{a}, remove)
	return err
}

// DeleteAttachments deletes the given attachments and optionally the associated files.
func DeleteAttachments(attachments []*Attachment, remove bool) (int, error) {
	if len(attachments) == 0 {
		return 0, nil
	}

	var ids = make([]int64, 0, len(attachments))
	for _, a := range attachments {
		ids = append(ids, a.ID)
	}

	cnt, err := x.In("id", ids).NoAutoCondition().Delete(attachments[0])
	if err != nil {
		return 0, err
	}

	if remove {
		for i, a := range attachments {
			if err := a.deleteFromBucket(); err != nil {
				return i, err
			}
		}
	}
	return int(cnt), nil
}

// DeleteAttachmentsByIssue deletes all attachments associated with the given issue.
func DeleteAttachmentsByIssue(issueID int64, remove bool) (int, error) {
	attachments, err := GetAttachmentsByIssueID(issueID)

	if err != nil {
		return 0, err
	}

	return DeleteAttachments(attachments, remove)
}

// DeleteAttachmentsByComment deletes all attachments associated with the given comment.
func DeleteAttachmentsByComment(commentID int64, remove bool) (int, error) {
	attachments, err := GetAttachmentsByCommentID(commentID)

	if err != nil {
		return 0, err
	}

	return DeleteAttachments(attachments, remove)
}

// UpdateAttachment updates the given attachment in database
func UpdateAttachment(atta *Attachment) error {
	return updateAttachment(x, atta)
}

func updateAttachment(e Engine, atta *Attachment) error {
	var sess *xorm.Session
	if atta.ID != 0 && atta.UUID == "" {
		sess = e.ID(atta.ID)
	} else {
		// Use uuid only if id is not set and uuid is set
		sess = e.Where("uuid = ?", atta.UUID)
	}
	_, err := sess.Cols("name", "issue_id", "release_id", "comment_id", "download_count").Update(atta)
	return err
}
