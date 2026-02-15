package handler

import (
	"testing"

	"github.com/maraichr/lattice/pkg/apierr"
)

func TestValidateSlug(t *testing.T) {
	tests := []struct {
		slug     string
		wantErr  bool
		wantCode apierr.Code
	}{
		{"my-project", false, ""},
		{"abc", false, ""},
		{"a-long-slug-with-numbers-123", false, ""},
		{"", true, apierr.CodeSlugRequired},
		{"ab", true, apierr.CodeSlugInvalid},          // too short
		{"-starts-dash", true, apierr.CodeSlugInvalid}, // starts with dash
		{"ends-dash-", true, apierr.CodeSlugInvalid},   // ends with dash
		{"UPPERCASE", true, apierr.CodeSlugInvalid},    // uppercase
		{"has space", true, apierr.CodeSlugInvalid},    // space
		{"has_underscore", true, apierr.CodeSlugInvalid}, // underscore
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			err := validateSlug(tt.slug)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSlug(%q) error = %v, wantErr %v", tt.slug, err, tt.wantErr)
			}
			if err != nil && err.Code() != tt.wantCode {
				t.Errorf("validateSlug(%q) code = %v, want %v", tt.slug, err.Code(), tt.wantCode)
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name     string
		wantErr  bool
		wantCode apierr.Code
	}{
		{"My Project", false, ""},
		{"x", false, ""},
		{"", true, apierr.CodeNameRequired},
		{string(make([]byte, 256)), true, apierr.CodeNameTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
			if err != nil && err.Code() != tt.wantCode {
				t.Errorf("validateName(%q) code = %v, want %v", tt.name, err.Code(), tt.wantCode)
			}
		})
	}
}

func TestValidateSourceType(t *testing.T) {
	tests := []struct {
		st       string
		wantErr  bool
		wantCode apierr.Code
	}{
		{"git", false, ""},
		{"database", false, ""},
		{"filesystem", false, ""},
		{"upload", false, ""},
		{"invalid", true, apierr.CodeInvalidSourceType},
		{"", true, apierr.CodeInvalidSourceType},
		{"GIT", true, apierr.CodeInvalidSourceType},
	}

	for _, tt := range tests {
		t.Run(tt.st, func(t *testing.T) {
			err := validateSourceType(tt.st)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSourceType(%q) error = %v, wantErr %v", tt.st, err, tt.wantErr)
			}
			if err != nil && err.Code() != tt.wantCode {
				t.Errorf("validateSourceType(%q) code = %v, want %v", tt.st, err.Code(), tt.wantCode)
			}
		})
	}
}
