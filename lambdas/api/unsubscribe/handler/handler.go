package handler

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"net/mail"

	"github.com/aws/aws-lambda-go/events"
	"github.com/gofor-little/xlambda"

	"github.com/strongishllama/millhouse.dev-cdk/pkg/db"
	"github.com/strongishllama/millhouse.dev-cdk/pkg/tmpl"
)

var (
	//go:embed templates
	templates embed.FS
)

func Handler(ctx context.Context, request *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	template, err := tmpl.NewTemplateFromFile(templates, "templates/unsubscribe-successful.tmpl.html", nil)
	if err != nil {
		return xlambda.ProxyResponseJSON(http.StatusInternalServerError, fmt.Errorf("failed to create template from file: %w", err), nil)
	}

	data := &RequestData{}
	if err := xlambda.ParseAndValidate(request, data); err != nil {
		return xlambda.ProxyResponseJSON(http.StatusBadRequest, err, nil)
	}

	if err := db.DeleteSubscription(ctx, data.ID, data.EmailAddress); err != nil {
		return xlambda.ProxyResponseJSON(http.StatusInternalServerError, fmt.Errorf("failed to delete subscription: %w", err), template)
	}

	return xlambda.ProxyResponseJSON(http.StatusOK, nil, template)
}

type RequestData struct {
	ID           string `mapstructure:"id"`
	EmailAddress string `mapstructure:"emailAddress"`
}

func (r *RequestData) Validate() error {
	if len(r.ID) == 0 {
		return errors.New("id cannot be empty")
	}

	if _, err := mail.ParseAddress(r.EmailAddress); err != nil {
		return fmt.Errorf("failed to validate EmailAddress: %w", err)
	}

	return nil
}
