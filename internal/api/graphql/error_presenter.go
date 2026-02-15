package graphql

import (
	"context"
	"errors"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/maraichr/lattice/pkg/apierr"
)

// ErrorPresenter returns a gqlgen ErrorPresenterFunc that puts apierr codes
// into the GraphQL extensions field.
func ErrorPresenter() graphql.ErrorPresenterFunc {
	return func(ctx context.Context, err error) *gqlerror.Error {
		// Default presentation (handles path, locations, etc.)
		gqlErr := graphql.DefaultErrorPresenter(ctx, err)

		// If the underlying error is an *apierr.Error, add code to extensions.
		var apiErr *apierr.Error
		if errors.As(err, &apiErr) {
			if gqlErr.Extensions == nil {
				gqlErr.Extensions = make(map[string]interface{})
			}
			gqlErr.Extensions["code"] = string(apiErr.Code())
			gqlErr.Message = apiErr.Message()
		}

		return gqlErr
	}
}
