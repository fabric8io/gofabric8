package comment

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// RevisionType defines the type of revision for a comment
type RevisionType int

const (
	_ RevisionType = iota // ignore first value by assigning to blank identifier
	// RevisionTypeCreate a comment creation
	RevisionTypeCreate // 1
	// RevisionTypeDelete a comment deletion
	RevisionTypeDelete // 2
	_                  // ignore 3rd value
	// RevisionTypeUpdate a comment update
	RevisionTypeUpdate // 4
)

// Revision represents a version of a comment
type Revision struct {
	ID uuid.UUID `gorm:"primary_key"`
	// the timestamp of the modification
	Time time.Time `gorm:"column:revision_time"`
	// the type of modification
	Type RevisionType `gorm:"column:revision_type"`
	// the identity of author of the comment modification
	ModifierIdentity uuid.UUID `sql:"type:uuid" gorm:"column:modifier_id"`
	// the id of the comment that changed
	CommentID uuid.UUID `gorm:"column:comment_id"`
	// the id of the parent of the comment that changed
	CommentParentID uuid.UUID `gorm:"column:comment_parent_id"`
	// the body of the comment (nil when comment was deleted)
	CommentBody *string `gorm:"column:comment_body"`
	// the markup used to input the comment body (nil when comment was deleted)
	CommentMarkup *string `gorm:"column:comment_markup"`
}

const (
	revisionTableName = "comment_revisions"
)

// TableName implements gorm.tabler
func (w Revision) TableName() string {
	return revisionTableName
}
