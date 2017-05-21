package tenant

import (
	"database/sql/driver"
	"errors"
	"time"

	uuid "github.com/satori/go.uuid"
)

// NamespaceType describes which type of namespace this is
type NamespaceType string

// Value - Implementation of valuer for database/sql
func (ns NamespaceType) Value() (driver.Value, error) {
	return string(ns), nil
}

// Scan - Implement the database/sql scanner interface
func (ns *NamespaceType) Scan(value interface{}) error {
	if value == nil {
		*ns = NamespaceType("")
		return nil
	}
	if bv, err := driver.String.ConvertValue(value); err == nil {
		// if this is a bool type
		if v, ok := bv.(string); ok {
			// set the value of the pointer yne to YesNoEnum(v)
			*ns = NamespaceType(v)
			return nil
		}
	}
	// otherwise, return an error
	return errors.New("failed to scan NamespaceType")
}

// Represents the namespace type
const (
	TypeChe     NamespaceType = "che"
	TypeJenkins NamespaceType = "jenkins"
	TypeTest    NamespaceType = "test"
	TypeStage   NamespaceType = "stage"
	TypeRun     NamespaceType = "run"
	TypeUser    NamespaceType = "user"
)

// Tenant is the owning OpenShift account
type Tenant struct {
	ID        uuid.UUID `sql:"type:uuid" gorm:"primary_key"` // This is the ID PK field
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	Email     string
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m Tenant) TableName() string {
	return "tenants"
}

// Namespace represent a single namespace owned by an Tenant
type Namespace struct {
	ID        uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	TenantID  uuid.UUID `sql:"type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	Name      string
	MasterURL string
	Type      NamespaceType
	Version   string
	State     string
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m Namespace) TableName() string {
	return "namespaces"
}
