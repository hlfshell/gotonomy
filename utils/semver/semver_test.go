package semver

import (
	"strings"
	"testing"
)

func TestNewSemVer(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    SemVer
		wantErr bool
	}{
		{
			name:  "standard format with v prefix",
			input: "v1.2.3",
			want:  SemVer{Major: 1, Minor: 2, Patch: 3, Hash: ""},
		},
		{
			name:  "standard format without v prefix",
			input: "1.2.3",
			want:  SemVer{Major: 1, Minor: 2, Patch: 3, Hash: ""},
		},
		{
			name:  "with hash using dash",
			input: "v1.2.3-abc123",
			want:  SemVer{Major: 1, Minor: 2, Patch: 3, Hash: "abc123"},
		},
		{
			name:  "with hash using colon",
			input: "1.2.3:abc123",
			want:  SemVer{Major: 1, Minor: 2, Patch: 3, Hash: "abc123"},
		},
		{
			name:  "large version numbers",
			input: "v10.20.30",
			want:  SemVer{Major: 10, Minor: 20, Patch: 30, Hash: ""},
		},
		{
			name:    "invalid format - too few parts",
			input:   "1.2",
			wantErr: true,
		},
		{
			name:    "invalid format - too many parts",
			input:   "1.2.3.4",
			wantErr: true,
		},
		{
			name:    "invalid format - non-numeric",
			input:   "v1.2.x",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSemVer(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSemVer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("NewSemVer() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSemVer_String(t *testing.T) {
	tests := []struct {
		name string
		sv   SemVer
		want string
	}{
		{
			name: "without hash",
			sv:   SemVer{Major: 1, Minor: 2, Patch: 3, Hash: ""},
			want: "v1.2.3",
		},
		{
			name: "with hash",
			sv:   SemVer{Major: 1, Minor: 2, Patch: 3, Hash: "abc123"},
			want: "v1.2.3-abc123",
		},
		{
			name: "zero version",
			sv:   SemVer{Major: 0, Minor: 0, Patch: 0, Hash: ""},
			want: "v0.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sv.String(); got != tt.want {
				t.Errorf("SemVer.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSemVer_Compare(t *testing.T) {
	tests := []struct {
		name  string
		s     SemVer
		other SemVer
		want  int
	}{
		{
			name:  "equal versions",
			s:     SemVer{Major: 1, Minor: 2, Patch: 3},
			other: SemVer{Major: 1, Minor: 2, Patch: 3},
			want:  0,
		},
		{
			name:  "less than - major",
			s:     SemVer{Major: 1, Minor: 2, Patch: 3},
			other: SemVer{Major: 2, Minor: 2, Patch: 3},
			want:  -1,
		},
		{
			name:  "greater than - major",
			s:     SemVer{Major: 2, Minor: 2, Patch: 3},
			other: SemVer{Major: 1, Minor: 2, Patch: 3},
			want:  1,
		},
		{
			name:  "less than - minor",
			s:     SemVer{Major: 1, Minor: 1, Patch: 3},
			other: SemVer{Major: 1, Minor: 2, Patch: 3},
			want:  -1,
		},
		{
			name:  "greater than - minor",
			s:     SemVer{Major: 1, Minor: 2, Patch: 3},
			other: SemVer{Major: 1, Minor: 1, Patch: 3},
			want:  1,
		},
		{
			name:  "less than - patch",
			s:     SemVer{Major: 1, Minor: 2, Patch: 2},
			other: SemVer{Major: 1, Minor: 2, Patch: 3},
			want:  -1,
		},
		{
			name:  "greater than - patch",
			s:     SemVer{Major: 1, Minor: 2, Patch: 3},
			other: SemVer{Major: 1, Minor: 2, Patch: 2},
			want:  1,
		},
		{
			name:  "hash ignored in comparison",
			s:     SemVer{Major: 1, Minor: 2, Patch: 3, Hash: "abc"},
			other: SemVer{Major: 1, Minor: 2, Patch: 3, Hash: "def"},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Compare(tt.other); got != tt.want {
				t.Errorf("SemVer.Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSemVer_ComparisonMethods(t *testing.T) {
	v1 := SemVer{Major: 1, Minor: 2, Patch: 3}
	v2 := SemVer{Major: 1, Minor: 2, Patch: 4}
	v3 := SemVer{Major: 1, Minor: 2, Patch: 3}

	// LT (LessThan)
	if !v1.LT(v2) {
		t.Error("v1.LT(v2) should be true")
	}
	if v1.LT(v3) {
		t.Error("v1.LT(v3) should be false")
	}

	// LTE (LessThanOrEqual)
	if !v1.LTE(v2) {
		t.Error("v1.LTE(v2) should be true")
	}
	if !v1.LTE(v3) {
		t.Error("v1.LTE(v3) should be true")
	}
	if v2.LTE(v1) {
		t.Error("v2.LTE(v1) should be false")
	}

	// GT (GreaterThan)
	if !v2.GT(v1) {
		t.Error("v2.GT(v1) should be true")
	}
	if v1.GT(v3) {
		t.Error("v1.GT(v3) should be false")
	}

	// GTE (GreaterThanOrEqual)
	if !v2.GTE(v1) {
		t.Error("v2.GTE(v1) should be true")
	}
	if !v1.GTE(v3) {
		t.Error("v1.GTE(v3) should be true")
	}
	if v1.GTE(v2) {
		t.Error("v1.GTE(v2) should be false")
	}

	// EQ (Equal)
	if !v1.EQ(v3) {
		t.Error("v1.EQ(v3) should be true")
	}
	if v1.EQ(v2) {
		t.Error("v1.EQ(v2) should be false")
	}
}

func TestGetBuildVersion(t *testing.T) {
	// Save original VersionInfo state
	originalMajor := VersionInfo.Major
	originalMinor := VersionInfo.Minor
	originalPatch := VersionInfo.Patch
	originalHash := VersionInfo.Hash

	// Restore original state after test
	defer func() {
		VersionInfo.Major = originalMajor
		VersionInfo.Minor = originalMinor
		VersionInfo.Patch = originalPatch
		VersionInfo.Hash = originalHash
	}()

	tests := []struct {
		name      string
		setup     func()
		want      SemVer
		wantErr   bool
		errSubstr string
	}{
		{
			name: "all fields set correctly",
			setup: func() {
				VersionInfo.Major = "1"
				VersionInfo.Minor = "2"
				VersionInfo.Patch = "3"
				VersionInfo.Hash = "abc123"
			},
			want: SemVer{Major: 1, Minor: 2, Patch: 3, Hash: "abc123"},
		},
		{
			name: "version without hash",
			setup: func() {
				VersionInfo.Major = "2"
				VersionInfo.Minor = "5"
				VersionInfo.Patch = "10"
				VersionInfo.Hash = ""
			},
			want: SemVer{Major: 2, Minor: 5, Patch: 10, Hash: ""},
		},
		{
			name: "partial version - defaults missing parts to 0",
			setup: func() {
				VersionInfo.Major = "1"
				VersionInfo.Minor = ""
				VersionInfo.Patch = ""
				VersionInfo.Hash = ""
			},
			want: SemVer{Major: 1, Minor: 0, Patch: 0, Hash: ""},
		},
		{
			name: "only major and minor set",
			setup: func() {
				VersionInfo.Major = "3"
				VersionInfo.Minor = "4"
				VersionInfo.Patch = ""
				VersionInfo.Hash = "def456"
			},
			want: SemVer{Major: 3, Minor: 4, Patch: 0, Hash: "def456"},
		},
		{
			name: "not set - all empty",
			setup: func() {
				VersionInfo.Major = ""
				VersionInfo.Minor = ""
				VersionInfo.Patch = ""
				VersionInfo.Hash = ""
			},
			wantErr:   true,
			errSubstr: "build-time version not set",
		},
		{
			name: "invalid major version",
			setup: func() {
				VersionInfo.Major = "invalid"
				VersionInfo.Minor = "2"
				VersionInfo.Patch = "3"
				VersionInfo.Hash = ""
			},
			wantErr:   true,
			errSubstr: "invalid build-time major version",
		},
		{
			name: "invalid minor version",
			setup: func() {
				VersionInfo.Major = "1"
				VersionInfo.Minor = "not-a-number"
				VersionInfo.Patch = "3"
				VersionInfo.Hash = ""
			},
			wantErr:   true,
			errSubstr: "invalid build-time minor version",
		},
		{
			name: "invalid patch version",
			setup: func() {
				VersionInfo.Major = "1"
				VersionInfo.Minor = "2"
				VersionInfo.Patch = "xyz"
				VersionInfo.Hash = ""
			},
			wantErr:   true,
			errSubstr: "invalid build-time patch version",
		},
		{
			name: "large version numbers",
			setup: func() {
				VersionInfo.Major = "10"
				VersionInfo.Minor = "20"
				VersionInfo.Patch = "30"
				VersionInfo.Hash = "large123"
			},
			want: SemVer{Major: 10, Minor: 20, Patch: 30, Hash: "large123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset VersionInfo
			VersionInfo.Major = ""
			VersionInfo.Minor = ""
			VersionInfo.Patch = ""
			VersionInfo.Hash = ""

			// Setup test state
			tt.setup()

			// Run test
			got, err := GetBuildVersion()

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBuildVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetBuildVersion() expected error but got none")
					return
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("GetBuildVersion() error = %v, want error containing %q", err, tt.errSubstr)
				}
				return
			}

			// Check result
			if got != tt.want {
				t.Errorf("GetBuildVersion() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
