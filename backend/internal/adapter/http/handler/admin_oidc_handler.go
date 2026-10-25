package handler

import (
	"context"
	"errors"

	"travel-blog/backend/internal/adapter/http/api"
	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
)

type OIDCConfig interface {
	View(actor domain.UserID) (service.OIDCConfigView, error)
	Save(actor domain.UserID, in service.OIDCConfigInput) error
	Delete(actor domain.UserID) error
	Test(actor domain.UserID, in service.OIDCConfigInput) error
}

func WithOIDCConfig(c OIDCConfig) Option {
	return func(h *Handler) { h.oidcConfig = c }
}

var errNoOIDCConfig = errors.New("OIDC configuration is not available")

func toOIDCInput(in api.OidcAdminInput) service.OIDCConfigInput {
	out := service.OIDCConfigInput{
		Issuer: in.Issuer, ClientID: in.ClientId, ClientSecret: deref(in.ClientSecret),
		RedirectURL: in.RedirectUrl, Name: deref(in.Name), Scopes: deref(in.Scopes),
	}
	if in.DisablePassword != nil {
		out.DisablePassword = *in.DisablePassword
	}
	return out
}

func (h *Handler) GetOidcAdminConfig(ctx context.Context, _ api.GetOidcAdminConfigRequestObject) (api.GetOidcAdminConfigResponseObject, error) {
	actor, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	if h.oidcConfig == nil {
		return nil, errNoOIDCConfig
	}
	v, err := h.oidcConfig.View(actor)
	switch {
	case errors.Is(err, service.ErrForbidden):
		return api.GetOidcAdminConfig403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(err)}, nil
	case err != nil:
		return nil, err
	}
	return api.GetOidcAdminConfig200JSONResponse{
		Source: api.OidcAdminConfigSource(v.Source), Enabled: v.Enabled, Issuer: v.Issuer,
		ClientId: v.ClientID, SecretSet: v.SecretSet, RedirectUrl: v.RedirectURL,
		Name: v.Name, Scopes: v.Scopes, DisablePassword: v.DisablePassword,
	}, nil
}

func (h *Handler) SaveOidcAdminConfig(ctx context.Context, req api.SaveOidcAdminConfigRequestObject) (api.SaveOidcAdminConfigResponseObject, error) {
	actor, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	if h.oidcConfig == nil {
		return nil, errNoOIDCConfig
	}
	err = h.oidcConfig.Save(actor, toOIDCInput(*req.Body))
	switch {
	case errors.Is(err, service.ErrForbidden):
		return api.SaveOidcAdminConfig403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(err)}, nil
	case errors.Is(err, service.ErrInvalid):
		return api.SaveOidcAdminConfig400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(err)}, nil
	case errors.Is(err, service.ErrConflict):
		return api.SaveOidcAdminConfig409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: conflict(err)}, nil
	case err != nil:
		return nil, err
	}
	return api.SaveOidcAdminConfig204Response{}, nil
}

func (h *Handler) DeleteOidcAdminConfig(ctx context.Context, _ api.DeleteOidcAdminConfigRequestObject) (api.DeleteOidcAdminConfigResponseObject, error) {
	actor, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	if h.oidcConfig == nil {
		return nil, errNoOIDCConfig
	}
	err = h.oidcConfig.Delete(actor)
	switch {
	case errors.Is(err, service.ErrForbidden):
		return api.DeleteOidcAdminConfig403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(err)}, nil
	case errors.Is(err, service.ErrConflict):
		return api.DeleteOidcAdminConfig409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: conflict(err)}, nil
	case err != nil:
		return nil, err
	}
	return api.DeleteOidcAdminConfig204Response{}, nil
}

func (h *Handler) TestOidcAdminConfig(ctx context.Context, req api.TestOidcAdminConfigRequestObject) (api.TestOidcAdminConfigResponseObject, error) {
	actor, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	if h.oidcConfig == nil {
		return nil, errNoOIDCConfig
	}
	err = h.oidcConfig.Test(actor, toOIDCInput(*req.Body))
	switch {
	case errors.Is(err, service.ErrForbidden):
		return api.TestOidcAdminConfig403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(err)}, nil
	case errors.Is(err, service.ErrInvalid):
		return api.TestOidcAdminConfig400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(err)}, nil
	case err != nil:
		return nil, err
	}
	return api.TestOidcAdminConfig204Response{}, nil
}
