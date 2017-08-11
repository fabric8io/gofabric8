package controller

// PermissionDefinition defines the Permissions available
type PermissionDefinition struct {
	CreateWorkItem string
	ReadWorkItem   string
	UpdateWorkItem string
	DeleteWorkItem string
}

// CRUDWorkItem returns all CRUD permissions for a WorkItem
func (p *PermissionDefinition) CRUDWorkItem() []string {
	return []string{p.CreateWorkItem, p.ReadWorkItem, p.UpdateWorkItem, p.DeleteWorkItem}
}

var (
	// Permissions defines the value of each Permission
	Permissions = PermissionDefinition{
		CreateWorkItem: "create.workitem",
		ReadWorkItem:   "read.workitem",
		UpdateWorkItem: "update.workitem",
		DeleteWorkItem: "delete.workitem",
	}
)
