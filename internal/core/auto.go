package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Auto directory and file constants
const (
	AutoDir                  = ".claude/auto"
	AutoPRDFile              = "prd.json"
	AutoProgressFile         = "progress.md"
	AutoPromptFile           = "prompt.md"
	AutoDiscoveryPromptFile  = "discovery-prompt.md"
	AutoSchemaVer            = "1.0"
)

// Iteration type constants for pilot mode
const (
	IterationTypeDiscovery      = "discovery"
	IterationTypeImplementation = "implementation"
)

// Pilot mode default constants
const (
	DefaultPilotIterations      = 30
	DefaultDiscoverInterval     = 5
	DefaultMaxDiscoveryTasks    = 10
	DefaultPilotPauseSecs       = 2
	DefaultPilotMaxConsecFails  = 3
	MinPendingTasksForDiscovery = 2
	MaxEmptyDiscoveries         = 2
)

// Task status constants
const (
	TaskStatusPending    = "pending"
	TaskStatusInProgress = "in_progress"
	TaskStatusCompleted  = "completed"
	TaskStatusSkipped    = "skipped"
	TaskStatusBlocked    = "blocked"
)

// Task priority constants
const (
	TaskPriorityCritical = "critical"
	TaskPriorityHigh     = "high"
	TaskPriorityMedium   = "medium"
	TaskPriorityLow      = "low"
)

// Task complexity constants
const (
	TaskComplexitySimple  = "simple"
	TaskComplexityMedium  = "medium"
	TaskComplexityComplex = "complex"
)

// Loop status constants
const (
	LoopStatusNotStarted = "not_started"
	LoopStatusRunning    = "running"
	LoopStatusPaused     = "paused"
	LoopStatusCompleted  = "completed"
	LoopStatusFailed     = "failed"
)

// AutoPRD represents the machine-readable task state for autonomous loops
type AutoPRD struct {
	Version  string       `json:"version"`
	Project  AutoProject  `json:"project"`
	Config   AutoConfig   `json:"config"`
	Tasks    []AutoTask   `json:"tasks"`
	Progress AutoProgress `json:"progress"`
}

// AutoProject holds project metadata
type AutoProject struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SourcePRD   string `json:"source_prd,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// AutoConfig holds loop configuration
type AutoConfig struct {
	MaxIterations   int      `json:"max_iterations"`
	QualityChecks   []string `json:"quality_checks"`
	AITool          string   `json:"ai_tool"`
	PromptFile      string   `json:"ai_prompt_file"`
	Sandbox         string   `json:"sandbox"`
	SandboxImage         string       `json:"sandbox_image,omitempty"`
	SandboxTemplate      string       `json:"sandbox_template,omitempty"`
	PilotMode            bool         `json:"pilot_mode,omitempty"`
	PilotConfig          *PilotConfig `json:"pilot_config,omitempty"`
	DiscoveryPrompt      string       `json:"discovery_prompt_file,omitempty"`
	ProgressMaxLearnings int          `json:"progress_max_learnings,omitempty"`
	ProgressMaxCompleted int          `json:"progress_max_completed,omitempty"`
	ProgressMaxLines     int          `json:"progress_max_lines,omitempty"`
}

// PilotConfig holds pilot-mode specific configuration
type PilotConfig struct {
	DiscoverInterval  int    `json:"discover_interval"`
	MaxDiscoveryTasks int    `json:"max_discovery_tasks"`
	Focus             string `json:"focus,omitempty"`
}

// AutoTask represents a single task in the autonomous loop
type AutoTask struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description,omitempty"`
	Status        string   `json:"status"`
	Priority      string   `json:"priority,omitempty"`
	Complexity    string   `json:"complexity,omitempty"`
	ParentID      string   `json:"parent_id,omitempty"`
	DependsOn     []string `json:"depends_on,omitempty"`
	FilesToCreate []string `json:"files_to_create,omitempty"`
	FilesToModify []string `json:"files_to_modify,omitempty"`
	Guardrails    []string `json:"guardrails,omitempty"`
	CompletedAt   string   `json:"completed_at,omitempty"`
	CommitSHA     string   `json:"commit_sha,omitempty"`
	Iteration     int      `json:"iteration,omitempty"`
	Source        string   `json:"source,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling for AutoTask.
// It handles numeric task IDs gracefully by converting them to strings,
// which is needed because AI tools sometimes generate "id": 1 instead of "id": "1".
func (t *AutoTask) UnmarshalJSON(data []byte) error {
	// Use a type alias to avoid infinite recursion
	type autoTaskAlias AutoTask

	// First try standard unmarshal (works when id is a string)
	var alias autoTaskAlias
	if err := json.Unmarshal(data, &alias); err == nil {
		*t = AutoTask(alias)
		return nil
	}

	// If that failed, try with a raw id field to handle numeric IDs
	var raw struct {
		autoTaskAlias
		RawID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*t = AutoTask(raw.autoTaskAlias)

	// Parse the raw ID — could be a string or number
	var numID float64
	if err := json.Unmarshal(raw.RawID, &numID); err == nil {
		// It was a number, convert to string
		if numID == float64(int(numID)) {
			t.ID = fmt.Sprintf("%d", int(numID))
		} else {
			t.ID = fmt.Sprintf("%g", numID)
		}
		return nil
	}

	// Try as string (shouldn't reach here if first unmarshal worked, but be safe)
	var strID string
	if err := json.Unmarshal(raw.RawID, &strID); err != nil {
		return fmt.Errorf("task id must be a string or number, got: %s", string(raw.RawID))
	}
	t.ID = strID

	return nil
}

// Task source constants
const (
	TaskSourceManual    = "manual"
	TaskSourcePRD       = "prd"
	TaskSourceDiscovery = "pilot-discovery"
)

// AutoProgress holds summary progress data
type AutoProgress struct {
	TotalTasks          int    `json:"total_tasks"`
	CompletedTasks      int    `json:"completed_tasks"`
	CurrentIteration    int    `json:"current_iteration"`
	TotalIterationsRun  int    `json:"total_iterations_run"`
	LastIterationAt     string `json:"last_iteration_at,omitempty"`
	Status              string `json:"status"`
	DiscoveryIterations int    `json:"discovery_iterations,omitempty"`
	ImplIterations      int    `json:"impl_iterations,omitempty"`
}

// NewAutoPRD creates a new AutoPRD with defaults
func NewAutoPRD(name, description string) *AutoPRD {
	now := time.Now().UTC().Format(time.RFC3339)
	return &AutoPRD{
		Version: AutoSchemaVer,
		Project: AutoProject{
			Name:        name,
			Description: description,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		Config: AutoConfig{
			MaxIterations: 50,
			QualityChecks: []string{"go test ./...", "go vet ./...", "go build ./..."},
			AITool:        "claude",
			PromptFile:    filepath.Join(AutoDir, AutoPromptFile),
			Sandbox:       SandboxNone,
		},
		Tasks: []AutoTask{},
		Progress: AutoProgress{
			Status: LoopStatusNotStarted,
		},
	}
}

// LoadAutoPRD loads a prd.json file from disk
func LoadAutoPRD(path string) (*AutoPRD, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read prd.json: %w", err)
	}

	var prd AutoPRD
	if err := json.Unmarshal(data, &prd); err != nil {
		return nil, fmt.Errorf("failed to parse prd.json: %w", err)
	}

	return &prd, nil
}

// Save writes the AutoPRD to disk using write-to-temp-then-rename for safety
func (p *AutoPRD) Save(path string) error {
	p.Project.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	p.RecalculateProgress()

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal prd.json: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpFile, path); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}


// GetAutoPRDPath returns the full path to prd.json in a project directory
func GetAutoPRDPath(projectDir string) string {
	return filepath.Join(projectDir, AutoDir, AutoPRDFile)
}

// GetAutoDir returns the full path to the .claude/auto directory
func GetAutoDir(projectDir string) string {
	return filepath.Join(projectDir, AutoDir)
}

// GetSupportedAITools returns the list of supported AI tools
func GetSupportedAITools() []string {
	return []string{"claude", "amp", "cursor", "codex"}
}

// IsValidAITool checks if the given tool name is supported
func IsValidAITool(tool string) bool {
	return slices.Contains(GetSupportedAITools(), strings.ToLower(tool))
}
