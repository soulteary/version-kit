package version

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	info := New("1.0.0", "abc123", "2025-01-01T00:00:00Z")

	assert.Equal(t, "1.0.0", info.Version)
	assert.Equal(t, "abc123", info.Commit)
	assert.Equal(t, "2025-01-01T00:00:00Z", info.BuildDate)
	assert.Equal(t, runtime.Version(), info.GoVersion)
	assert.NotEmpty(t, info.Platform)
	assert.NotEmpty(t, info.Compiler)
}

func TestNewWithBranch(t *testing.T) {
	info := NewWithBranch("1.0.0", "abc123", "2025-01-01T00:00:00Z", "main")

	assert.Equal(t, "1.0.0", info.Version)
	assert.Equal(t, "abc123", info.Commit)
	assert.Equal(t, "2025-01-01T00:00:00Z", info.BuildDate)
	assert.Equal(t, "main", info.Branch)
}

func TestDefault(t *testing.T) {
	// Save original values
	origVersion := Version
	origCommit := Commit
	origBuildDate := BuildDate
	origBranch := Branch
	defer func() {
		Version = origVersion
		Commit = origCommit
		BuildDate = origBuildDate
		Branch = origBranch
	}()

	// Set test values
	Version = "2.0.0"
	Commit = "def456"
	BuildDate = "2025-06-15T12:00:00Z"
	Branch = "develop"

	info := Default()

	assert.Equal(t, "2.0.0", info.Version)
	assert.Equal(t, "def456", info.Commit)
	assert.Equal(t, "2025-06-15T12:00:00Z", info.BuildDate)
	assert.Equal(t, "develop", info.Branch)
}

func TestInfo_String(t *testing.T) {
	tests := []struct {
		name     string
		info     *Info
		expected string
	}{
		{
			name:     "with commit",
			info:     New("1.0.0", "abc1234567890", ""),
			expected: "1.0.0 (abc1234)",
		},
		{
			name:     "with short commit",
			info:     New("1.0.0", "abc", ""),
			expected: "1.0.0 (abc)",
		},
		{
			name:     "without commit",
			info:     New("1.0.0", "", ""),
			expected: "1.0.0",
		},
		{
			name:     "with unknown commit",
			info:     New("1.0.0", "unknown", ""),
			expected: "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.info.String())
		})
	}
}

func TestInfo_Full(t *testing.T) {
	info := NewWithBranch("1.0.0", "abc123", "2025-01-01T00:00:00Z", "main")
	full := info.Full()

	assert.Contains(t, full, "Version:    1.0.0")
	assert.Contains(t, full, "Commit:     abc123")
	assert.Contains(t, full, "Branch:     main")
	assert.Contains(t, full, "Built:      2025-01-01T00:00:00Z")
	assert.Contains(t, full, "Go version:")
	assert.Contains(t, full, "Platform:")
	assert.Contains(t, full, "Compiler:")
}

func TestInfo_Full_Minimal(t *testing.T) {
	info := &Info{
		Version:   "1.0.0",
		GoVersion: "go1.25",
		Platform:  "linux/amd64",
		Compiler:  "gc",
	}
	full := info.Full()

	assert.Contains(t, full, "Version:    1.0.0")
	assert.NotContains(t, full, "Commit:")
	assert.NotContains(t, full, "Branch:")
	assert.NotContains(t, full, "Built:")
}

func TestInfo_JSON(t *testing.T) {
	info := New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
	jsonStr := info.JSON()

	var parsed Info
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", parsed.Version)
	assert.Equal(t, "abc123", parsed.Commit)
	assert.Equal(t, "2025-01-01T00:00:00Z", parsed.BuildDate)
}

func TestInfo_JSONPretty(t *testing.T) {
	info := New("1.0.0", "abc123", "2025-01-01T00:00:00Z")
	jsonStr := info.JSONPretty()

	// Should contain indentation
	assert.Contains(t, jsonStr, "\n")
	assert.Contains(t, jsonStr, "  ")

	var parsed Info
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", parsed.Version)
}

func TestInfo_Map(t *testing.T) {
	info := NewWithBranch("1.0.0", "abc123", "2025-01-01T00:00:00Z", "main")
	m := info.Map()

	assert.Equal(t, "1.0.0", m["version"])
	assert.Equal(t, "abc123", m["commit"])
	assert.Equal(t, "2025-01-01T00:00:00Z", m["build_date"])
	assert.Equal(t, "main", m["branch"])
	assert.NotEmpty(t, m["go_version"])
	assert.NotEmpty(t, m["platform"])
	assert.NotEmpty(t, m["compiler"])
}

func TestInfo_Map_Minimal(t *testing.T) {
	info := &Info{
		Version:   "1.0.0",
		GoVersion: "go1.25",
		Platform:  "linux/amd64",
		Compiler:  "gc",
	}
	m := info.Map()

	assert.Equal(t, "1.0.0", m["version"])
	_, hasCommit := m["commit"]
	assert.False(t, hasCommit)
	_, hasBranch := m["branch"]
	assert.False(t, hasBranch)
}

func TestInfo_Validate(t *testing.T) {
	tests := []struct {
		name    string
		info    *Info
		wantErr bool
	}{
		{
			name:    "valid",
			info:    New("1.0.0", "", ""),
			wantErr: false,
		},
		{
			name:    "empty version",
			info:    &Info{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.info.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInfo_IsDev(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{"dev", "dev", true},
		{"development", "development", true},
		{"empty", "", true},
		{"release", "1.0.0", false},
		{"v prefix", "v1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &Info{Version: tt.version}
			assert.Equal(t, tt.expected, info.IsDev())
		})
	}
}

func TestInfo_BuildTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		buildDate string
		wantZero  bool
	}{
		{
			name:      "RFC3339",
			buildDate: "2025-01-01T12:00:00Z",
			wantZero:  false,
		},
		{
			name:      "RFC3339 with timezone",
			buildDate: "2025-01-01T12:00:00+08:00",
			wantZero:  false,
		},
		{
			name:      "simple datetime",
			buildDate: "2025-01-01 12:00:00",
			wantZero:  false,
		},
		{
			name:      "date only",
			buildDate: "2025-01-01",
			wantZero:  false,
		},
		{
			name:      "empty",
			buildDate: "",
			wantZero:  true,
		},
		{
			name:      "unknown",
			buildDate: "unknown",
			wantZero:  true,
		},
		{
			name:      "invalid",
			buildDate: "not-a-date",
			wantZero:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &Info{BuildDate: tt.buildDate}
			ts := info.BuildTimestamp()
			if tt.wantZero {
				assert.True(t, ts.IsZero())
			} else {
				assert.False(t, ts.IsZero())
			}
		})
	}
}

func TestInfo_ShortCommit(t *testing.T) {
	tests := []struct {
		name     string
		commit   string
		expected string
	}{
		{"long commit", "abc1234567890def", "abc1234"},
		{"exactly 7", "abc1234", "abc1234"},
		{"short", "abc", "abc"},
		{"empty", "", ""},
		{"unknown", "unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &Info{Commit: tt.commit}
			assert.Equal(t, tt.expected, info.ShortCommit())
		})
	}
}

func TestBuilder(t *testing.T) {
	info := NewBuilder().
		WithVersion("1.0.0").
		WithCommit("abc123").
		WithBuildDate("2025-01-01T00:00:00Z").
		WithBranch("main").
		Build()

	assert.Equal(t, "1.0.0", info.Version)
	assert.Equal(t, "abc123", info.Commit)
	assert.Equal(t, "2025-01-01T00:00:00Z", info.BuildDate)
	assert.Equal(t, "main", info.Branch)
	assert.Equal(t, runtime.Version(), info.GoVersion)
	assert.NotEmpty(t, info.Platform)
}

func TestVersionVariablesNotEmpty(t *testing.T) {
	// Default values should not be empty
	if Version == "" {
		t.Fatal("Version should not be empty")
	}
	if Commit == "" {
		t.Fatal("Commit should not be empty")
	}
	if BuildDate == "" {
		t.Fatal("BuildDate should not be empty")
	}
}

func TestVersionVariablesOverride(t *testing.T) {
	origVersion := Version
	origCommit := Commit
	origBuildDate := BuildDate
	defer func() {
		Version = origVersion
		Commit = origCommit
		BuildDate = origBuildDate
	}()

	Version = "1.2.3"
	Commit = "abcdef"
	BuildDate = "2025-01-01T00:00:00Z"

	if Version != "1.2.3" {
		t.Fatalf("Version = %s, want 1.2.3", Version)
	}
	if Commit != "abcdef" {
		t.Fatalf("Commit = %s, want abcdef", Commit)
	}
	if BuildDate != "2025-01-01T00:00:00Z" {
		t.Fatalf("BuildDate = %s, want 2025-01-01T00:00:00Z", BuildDate)
	}
}

func TestInfo_Platform(t *testing.T) {
	info := New("1.0.0", "", "")

	// Platform should be in format OS/ARCH
	parts := strings.Split(info.Platform, "/")
	assert.Len(t, parts, 2)
	assert.NotEmpty(t, parts[0]) // OS
	assert.NotEmpty(t, parts[1]) // ARCH
}

func TestInfo_BuildTimestamp_RFC1123(t *testing.T) {
	// Test RFC1123 format
	buildDate := time.Now().Format(time.RFC1123)
	info := &Info{BuildDate: buildDate}
	ts := info.BuildTimestamp()
	assert.False(t, ts.IsZero())
}
