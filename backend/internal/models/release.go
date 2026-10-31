package models

import "time"

type Migration struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	Name      string `gorm:"size:100"`
}
type AuditEvent struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	UserID    uint      `json:"user_id"`
	Action    string    `json:"action" gorm:"size:100;index"`
	Resource  string    `json:"resource" gorm:"size:500"`
}
type Application struct {
	ID              uint                  `json:"id" gorm:"primaryKey"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	Name            string                `json:"name" gorm:"size:100;uniqueIndex"`
	Description     string                `json:"description"`
	Kind            string                `json:"kind" gorm:"size:20;default:compose"`
	Scripts         DeploymentScripts     `json:"scripts" gorm:"serializer:json;type:mediumtext"`
	Category        string                `json:"category" gorm:"size:30"`
	TemplateID      string                `json:"template_id" gorm:"size:100"`
	TemplateVersion string                `json:"template_version" gorm:"size:100"`
	Definition      ApplicationDefinition `json:"definition" gorm:"serializer:json;type:mediumtext"`
}

type ApplicationParameter struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Type     string `json:"type"`
	Default  string `json:"default"`
	Required bool   `json:"required"`
	Help     string `json:"help"`
}
type ApplicationEndpoint struct {
	Name     string `json:"name"`
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
	Path     string `json:"path"`
}
type ApplicationDefinition struct {
	Compose        string                 `json:"compose"`
	Configs        StringMap              `json:"configs"`
	Parameters     []ApplicationParameter `json:"parameters"`
	Stateful       bool                   `json:"stateful"`
	Topology       string                 `json:"topology"`
	Instructions   string                 `json:"instructions"`
	Storage        string                 `json:"storage"`
	Initialization string                 `json:"initialization"`
	Endpoints      []ApplicationEndpoint  `json:"endpoints"`
}
type ApplicationTemplate struct {
	ID          uint                  `json:"id" gorm:"primaryKey"`
	CreatedAt   time.Time             `json:"created_at"`
	Name        string                `json:"name" gorm:"size:100;uniqueIndex:template_version"`
	Version     string                `json:"version" gorm:"size:100;uniqueIndex:template_version"`
	Description string                `json:"description"`
	Category    string                `json:"category" gorm:"size:30"`
	Definition  ApplicationDefinition `json:"definition" gorm:"serializer:json;type:mediumtext"`
	Scripts     DeploymentScripts     `json:"scripts" gorm:"serializer:json;type:mediumtext"`
}

// DeploymentScripts 由管理员配置，创建发布版本时保存应用脚本快照。
type DeploymentScripts struct {
	Before string `json:"before"`
	Deploy string `json:"deploy"`
	After  string `json:"after"`
}

type Environment struct {
	RestoreConfiguration bool              `json:"restore_configuration,omitempty" gorm:"-"`
	ID                   uint              `json:"id" gorm:"primaryKey"`
	ApplicationID        uint              `json:"application_id" gorm:"uniqueIndex:app_env"`
	Name                 string            `json:"name" gorm:"size:100;uniqueIndex:app_env"`
	TargetID             uint              `json:"target_id"`
	TargetIDs            string            `json:"target_ids" gorm:"type:text"`
	Parameters           StringMap         `json:"parameters" gorm:"type:json"`
	CurrentReleaseID     uint              `json:"current_release_id"`
	Stateless            bool              `json:"stateless"`
	AutoRollback         bool              `json:"auto_rollback"`
	AutoDeploy           bool              `json:"auto_deploy"`
	RegistryID           uint              `json:"registry_id"`
	Repository           string            `json:"repository" gorm:"size:255"`
	TagPattern           string            `json:"tag_pattern" gorm:"size:100"`
	AllowedPublishers    string            `json:"allowed_publishers" gorm:"type:text"`
	ScriptMode           string            `json:"script_mode" gorm:"size:20"`
	Scripts              DeploymentScripts `json:"scripts" gorm:"serializer:json;type:mediumtext"`
	Health               string            `json:"health" gorm:"size:50"`
	RuntimeState         string            `json:"runtime_state" gorm:"size:50"`
}
type Release struct {
	ID            uint              `json:"id" gorm:"primaryKey"`
	CreatedAt     time.Time         `json:"created_at"`
	ApplicationID uint              `json:"application_id" gorm:"index"`
	Version       string            `json:"version" gorm:"size:200"`
	Source        string            `json:"source" gorm:"size:20"`
	RegistryID    uint              `json:"registry_id"`
	BundleID      uint              `json:"bundle_id"`
	Platform      string            `json:"platform" gorm:"size:50"`
	Compose       string            `json:"compose" gorm:"type:mediumtext"`
	Images        string            `json:"images" gorm:"type:mediumtext"`
	Configs       string            `json:"configs" gorm:"type:mediumtext"`
	GitCommit     string            `json:"git_commit" gorm:"size:100"`
	PublisherID   uint              `json:"publisher_id"`
	Retained      bool              `json:"retained" gorm:"default:true"`
	Scripts       DeploymentScripts `json:"scripts" gorm:"serializer:json;type:mediumtext"`
}
type Deployment struct {
	LegacyScriptID    uint       `json:"legacy_script_id"`
	TargetID          uint       `json:"target_id"`
	Batch             string     `json:"batch" gorm:"size:100;index"`
	Sequence          int        `json:"sequence"`
	ID                uint       `json:"id" gorm:"primaryKey"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	EnvironmentID     uint       `json:"environment_id" gorm:"index"`
	ReleaseID         uint       `json:"release_id"`
	PreviousReleaseID uint       `json:"previous_release_id"`
	UserID            uint       `json:"user_id"`
	IdempotencyKey    string     `json:"idempotency_key" gorm:"size:200;uniqueIndex"`
	Status            string     `json:"status" gorm:"size:30;index"`
	Stage             string     `json:"stage" gorm:"size:50"`
	Automatic         bool       `json:"automatic"`
	EventID           uint       `json:"event_id"`
	Snapshot          string     `json:"-" gorm:"type:mediumtext"`
	Result            string     `json:"result" gorm:"type:text"`
	ActualImages      string     `json:"actual_images" gorm:"type:text"`
	RecoveryStatus    string     `json:"recovery_status" gorm:"size:30"`
	StartedAt         *time.Time `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at"`
	LeaseUntil        *time.Time `json:"lease_until"`
	// 历史记录删除后保留执行快照，供运行检查、恢复配置和回滚使用。
	HistoryDeletedAt *time.Time `json:"-" gorm:"index"`
}
type TaskLog struct {
	EventKey     *string   `json:"-" gorm:"size:100;uniqueIndex"`
	ID           uint      `json:"id" gorm:"primaryKey"`
	DeploymentID uint      `json:"deployment_id" gorm:"index"`
	CreatedAt    time.Time `json:"created_at"`
	Stage        string    `json:"stage" gorm:"size:50"`
	Message      string    `json:"message" gorm:"type:mediumtext"`
}
type Bundle struct {
	LocalLoaded   bool      `json:"local_loaded"`
	LocalLoadNote string    `json:"local_load_note" gorm:"type:text"`
	Kind          string    `json:"kind" gorm:"size:20;default:bundle"`
	ID            uint      `json:"id" gorm:"primaryKey"`
	CreatedAt     time.Time `json:"created_at"`
	Name          string    `json:"name" gorm:"size:255"`
	SHA256        string    `json:"sha256" gorm:"size:64;uniqueIndex"`
	Status        string    `json:"status" gorm:"size:30"`
	Path          string    `json:"-"`
	Manifest      string    `json:"manifest" gorm:"type:mediumtext"`
	Error         string    `json:"error" gorm:"type:text"`
	UserID        uint      `json:"user_id"`
}
type ImageEvent struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	CreatedAt  time.Time `json:"created_at"`
	RegistryID uint      `json:"registry_id"`
	Repository string    `json:"repository" gorm:"size:255"`
	Reference  string    `json:"reference" gorm:"size:255"`
	Digest     string    `json:"digest" gorm:"size:100"`
	MediaType  string    `json:"media_type" gorm:"size:200"`
	UserID     uint      `json:"user_id"`
	Key        string    `json:"key" gorm:"size:200;uniqueIndex"`
	Imported   bool      `json:"imported"`
	Committed  bool      `json:"committed"`
	Processed  bool      `json:"processed" gorm:"index"`
}
type AutomationToken struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	CreatedAt   time.Time `json:"created_at"`
	UserID      uint      `json:"user_id" gorm:"index"`
	Name        string    `json:"name" gorm:"size:100"`
	Hash        string    `json:"-" gorm:"size:64;uniqueIndex"`
	Scopes      string    `json:"scopes" gorm:"size:200"`
	AuthVersion uint      `json:"-"`
	ExpiresAt   time.Time `json:"expires_at"`
	Revoked     bool      `json:"revoked"`
}
type Target struct {
	InstanceID string     `json:"instance_id" gorm:"size:100"`
	ID         uint       `json:"id" gorm:"primaryKey"`
	Name       string     `json:"name" gorm:"size:100;uniqueIndex"`
	TokenHash  string     `json:"-" gorm:"size:64"`
	Enabled    bool       `json:"enabled"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	Platform   string     `json:"platform" gorm:"size:50"`
}

type TargetState struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	EnvironmentID    uint      `json:"environment_id" gorm:"uniqueIndex:environment_target"`
	TargetID         uint      `json:"target_id" gorm:"uniqueIndex:environment_target"`
	CurrentReleaseID uint      `json:"current_release_id"`
	Health           string    `json:"health" gorm:"size:50"`
	RuntimeState     string    `json:"runtime_state" gorm:"size:50"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type MaintenanceJob struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Kind      string    `json:"kind" gorm:"size:30"`
	Status    string    `json:"status" gorm:"size:30;index"`
	File      string    `json:"file" gorm:"size:255"`
	Error     string    `json:"error" gorm:"type:text"`
	UserID    uint      `json:"user_id"`
}
