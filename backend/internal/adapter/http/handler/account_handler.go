package handler

import (
	"context"
	"errors"
	"net/http"

	"travel-blog/backend/internal/adapter/http/api"
	"travel-blog/backend/internal/adapter/http/auth"
	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
)

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (h *Handler) Register(_ context.Context, req api.RegisterRequestObject) (api.RegisterResponseObject, error) {
	if !h.passwordLogin() {
		return api.Register403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(errPasswordLoginOff)}, nil
	}
	in := req.Body
	email := ""
	if in.Email != nil {
		email = string(*in.Email)
	}
	u, token, err := h.accounts.Register(service.Registration{
		Email:     email,
		Username:  in.Username,
		Password:  in.Password,
		FirstName: deref(in.FirstName),
		LastName:  deref(in.LastName),
		Language:  deref(in.Language),
	})
	switch {
	case errors.Is(err, service.ErrForbidden):
		return api.Register403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(err)}, nil
	case errors.Is(err, service.ErrInvalid):
		return api.Register400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(err)}, nil
	case errors.Is(err, service.ErrConflict):
		return api.Register409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: conflict(err)}, nil
	case err != nil:
		return nil, err
	}
	_, settings, err := h.accounts.Account(u.ID)
	if err != nil {
		return nil, err
	}
	cookie := auth.SessionCookie(token, service.SessionTTL)
	return api.Register201JSONResponse{
		Body:    toAPIAccount(u, settings),
		Headers: api.Register201ResponseHeaders{SetCookie: &cookie},
	}, nil
}

func (h *Handler) CheckUsername(_ context.Context, req api.CheckUsernameRequestObject) (api.CheckUsernameResponseObject, error) {
	exists, err := h.accounts.UsernameTaken(req.Username)
	switch {
	case errors.Is(err, service.ErrInvalid):
		return api.CheckUsername400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(err)}, nil
	case err != nil:
		return nil, err
	}
	open, err := h.accounts.RegistrationOpen()
	if err != nil {
		return nil, err
	}
	return api.CheckUsername200JSONResponse{Exists: exists, RegistrationOpen: open}, nil
}

func (h *Handler) OidcConfig(_ context.Context, _ api.OidcConfigRequestObject) (api.OidcConfigResponseObject, error) {
	cfg := api.OidcConfig{PasswordLogin: h.passwordLogin()}
	if h.oidc != nil && h.oidc.Enabled() {
		cfg.Enabled, cfg.Name = true, h.oidc.Name()
	}
	return api.OidcConfig200JSONResponse(cfg), nil
}

func (h *Handler) Login(_ context.Context, req api.LoginRequestObject) (api.LoginResponseObject, error) {
	if !h.passwordLogin() {
		return api.Login403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(errPasswordLoginOff)}, nil
	}
	user, token, err := h.accounts.Login(req.Body.Username, req.Body.Password)
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		return api.Login401ApplicationProblemPlusJSONResponse{UnauthorizedApplicationProblemPlusJSONResponse: unauthorized(err)}, nil
	case err != nil:
		return nil, err
	}
	u, settings, err := h.accounts.Account(user)
	if err != nil {
		return nil, err
	}
	cookie := auth.SessionCookie(token, service.SessionTTL)
	return api.Login200JSONResponse{
		Body:    toAPIAccount(u, settings),
		Headers: api.Login200ResponseHeaders{SetCookie: &cookie},
	}, nil
}

func (h *Handler) Logout(ctx context.Context, _ api.LogoutRequestObject) (api.LogoutResponseObject, error) {
	if err := h.accounts.Logout(auth.TokenFrom(ctx)); err != nil {
		return nil, err
	}
	cookie := auth.ClearCookie()
	return api.Logout204Response{Headers: api.Logout204ResponseHeaders{SetCookie: &cookie}}, nil
}

func (h *Handler) DeleteAccount(ctx context.Context, _ api.DeleteAccountRequestObject) (api.DeleteAccountResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	err = h.accounts.DeleteAccount(user)
	if errors.Is(err, service.ErrConflict) {
		return api.DeleteAccount409ApplicationProblemPlusJSONResponse(problem(http.StatusConflict, err)), nil
	}
	if err != nil {
		return nil, err
	}
	cookie := auth.ClearCookie()
	return api.DeleteAccount204Response{Headers: api.DeleteAccount204ResponseHeaders{SetCookie: &cookie}}, nil
}

func (h *Handler) UpdateProfile(ctx context.Context, req api.UpdateProfileRequestObject) (api.UpdateProfileResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	in := req.Body
	up := domain.ProfileUpdate{
		Username:    in.Username,
		FirstName:   in.FirstName,
		LastName:    in.LastName,
		DisplayName: in.DisplayName,
		Bio:         in.Bio,
		Language:    in.Language,
		Timezone:    in.Timezone,
		HomeCountry: in.HomeCountry,
		HomeAirport: in.HomeAirport,
	}
	up.Email = in.Email
	p, err := h.accounts.UpdateProfile(user, up)
	switch {
	case errors.Is(err, service.ErrInvalid):
		return api.UpdateProfile400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(err)}, nil
	case errors.Is(err, service.ErrConflict):
		return api.UpdateProfile409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: conflict(err)}, nil
	case err != nil:
		return nil, err
	}
	etag, err := h.accountETag(user)
	if err != nil {
		return nil, err
	}
	return api.UpdateProfile200JSONResponse{Body: toAPIProfile(p), Headers: api.UpdateProfile200ResponseHeaders{ETag: etag}}, nil
}

func (h *Handler) SaveSettings(ctx context.Context, req api.SaveSettingsRequestObject) (api.SaveSettingsResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	in := req.Body
	err = h.accounts.SaveSettings(user, domain.Settings{
		DistanceUnit:      domain.DistanceUnit(in.DistanceUnit),
		Theme:             domain.Theme(in.Theme),
		DefaultView:       domain.View(in.DefaultView),
		DefaultVisibility: domain.Visibility(in.DefaultVisibility),
		ProfileVisibility: domain.Visibility(in.ProfileVisibility),
	})
	switch {
	case errors.Is(err, service.ErrInvalid):
		return api.SaveSettings400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(err)}, nil
	case err != nil:
		return nil, err
	}
	etag, err := h.accountETag(user)
	if err != nil {
		return nil, err
	}
	return api.SaveSettings204Response{Headers: api.SaveSettings204ResponseHeaders{ETag: etag}}, nil
}

func (h *Handler) ChangePassword(ctx context.Context, req api.ChangePasswordRequestObject) (api.ChangePasswordResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	err = h.accounts.ChangePassword(user, auth.TokenFrom(ctx), req.Body.CurrentPassword, req.Body.NewPassword)
	switch {
	case errors.Is(err, service.ErrInvalid):
		return api.ChangePassword400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(err)}, nil
	case errors.Is(err, service.ErrForbidden):
		return api.ChangePassword403ApplicationProblemPlusJSONResponse(problem(http.StatusForbidden, err)), nil
	case err != nil:
		return nil, err
	}
	return api.ChangePassword204Response{}, nil
}
