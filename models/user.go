package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserRole represents the role of a user
type UserRole string

const (
	RoleCompanyAdmin UserRole = "company_admin"
	RoleBotManager   UserRole = "bot_manager"
	RoleHumanAgent   UserRole = "human_agent"
	RoleAnalyst      UserRole = "analyst"
	RoleViewer       UserRole = "viewer"
)

// User represents a user in the system
type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username string             `bson:"username" json:"username"`
	Email    string             `bson:"email" json:"email"`
	Password string             `bson:"password" json:"-"` // Hashed password

	// Name fields
	FirstName string `bson:"first_name" json:"first_name"`
	LastName  string `bson:"last_name" json:"last_name"`

	// Company association
	CompanyID string `bson:"company_id" json:"company_id"`

	// Role and permissions
	Role UserRole `bson:"role" json:"role"`

	// Status
	IsActive  bool      `bson:"is_active" json:"is_active"`
	LastLogin time.Time `bson:"last_login,omitempty" json:"last_login,omitempty"`

	// Metadata
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// RolePermissions defines what each role can do
type RolePermissions struct {
	Role        UserRole
	Description string
	Permissions []string
}

// GetRolePermissions returns the permissions for each role
func GetRolePermissions() map[UserRole]RolePermissions {
	return map[UserRole]RolePermissions{
		RoleCompanyAdmin: {
			Role:        RoleCompanyAdmin,
			Description: "Full access to all features",
			Permissions: []string{
				"manage_company",
				"manage_users",
				"manage_bots",
				"manage_pages",
				"view_analytics",
				"export_data",
				"handle_conversations",
				"manage_settings",
				"manage_billing",
			},
		},
		RoleBotManager: {
			Role:        RoleBotManager,
			Description: "Manage assigned bots",
			Permissions: []string{
				"manage_assigned_bots",
				"view_bot_analytics",
				"handle_bot_conversations",
				"manage_bot_settings",
				"export_bot_data",
			},
		},
		RoleHumanAgent: {
			Role:        RoleHumanAgent,
			Description: "Handle conversations",
			Permissions: []string{
				"handle_conversations",
				"view_conversation_history",
				"transfer_conversations",
				"add_notes",
			},
		},
		RoleAnalyst: {
			Role:        RoleAnalyst,
			Description: "View analytics and reports",
			Permissions: []string{
				"view_analytics",
				"generate_reports",
				"export_data",
				"view_conversation_history",
			},
		},
		RoleViewer: {
			Role:        RoleViewer,
			Description: "Read-only dashboard access",
			Permissions: []string{
				"view_dashboard",
				"view_basic_analytics",
			},
		},
	}
}

// HasPermission checks if a user role has a specific permission
func (u *User) HasPermission(permission string) bool {
	permissions := GetRolePermissions()
	if rolePerms, exists := permissions[u.Role]; exists {
		for _, perm := range rolePerms.Permissions {
			if perm == permission {
				return true
			}
		}
	}
	return false
}

// IsValidRole checks if a role is valid
func IsValidRole(role string) bool {
	validRoles := []UserRole{
		RoleCompanyAdmin,
		RoleBotManager,
		RoleHumanAgent,
		RoleAnalyst,
		RoleViewer,
	}

	for _, validRole := range validRoles {
		if UserRole(role) == validRole {
			return true
		}
	}
	return false
}
