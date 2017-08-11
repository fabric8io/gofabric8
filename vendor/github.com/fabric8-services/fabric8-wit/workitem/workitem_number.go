package workitem

import (
	uuid "github.com/satori/go.uuid"
)

// WorkItemNumberSequence the sequence for work item numbers in a space
type WorkItemNumberSequence struct {
	SpaceID    uuid.UUID `sql:"type:uuid" gorm:"primary_key"`
	CurrentVal int
}

const (
	workitemNumberTableName = "work_item_number_sequences"
)

// TableName implements gorm.tabler
func (w WorkItemNumberSequence) TableName() string {
	return workitemNumberTableName
}
