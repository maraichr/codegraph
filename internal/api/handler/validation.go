package handler

import (
	"regexp"

	"github.com/maraichr/lattice/pkg/apierr"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)

func validateSlug(slug string) *apierr.Error {
	if slug == "" {
		return apierr.SlugRequired()
	}
	if !slugRegex.MatchString(slug) {
		return apierr.SlugInvalid()
	}
	return nil
}

func validateName(name string) *apierr.Error {
	if name == "" {
		return apierr.NameRequired()
	}
	if len(name) > 255 {
		return apierr.NameTooLong()
	}
	return nil
}

var validSourceTypes = map[string]bool{
	"git":        true,
	"database":   true,
	"filesystem": true,
	"upload":     true,
}

func validateSourceType(st string) *apierr.Error {
	if !validSourceTypes[st] {
		return apierr.InvalidSourceType()
	}
	return nil
}
