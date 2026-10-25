package handler

import (
	"context"
	"errors"

	"travel-blog/backend/internal/adapter/http/api"
	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
)

func toAPIAdminUser(u domain.User) api.AdminUser {
	out := api.AdminUser{
		Id:                 string(u.ID),
		Username:           u.Profile.Username,
		DisplayName:        u.Profile.DisplayName,
		Role:               api.Role(u.Role),
		MustChangePassword: u.MustChangePassword,
		CreatedAt:          u.Profile.CreatedAt,
	}
	if u.Profile.Email != "" {
		out.Email = &u.Profile.Email
	}
	return out
}

func (h *Handler) ListUsers(ctx context.Context, _ api.ListUsersRequestObject) (api.ListUsersResponseObject, error) {
	actor, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	users, err := h.accounts.Users(actor)
	switch {
	case errors.Is(err, service.ErrForbidden):
		return api.ListUsers403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(err)}, nil
	case err != nil:
		return nil, err
	}
	out := make(api.ListUsers200JSONResponse, 0, len(users))
	for _, u := range users {
		out = append(out, toAPIAdminUser(u))
	}
	return out, nil
}

func (h *Handler) UpdateUser(ctx context.Context, req api.UpdateUserRequestObject) (api.UpdateUserResponseObject, error) {
	actor, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	u, err := h.accounts.SetRole(actor, domain.UserID(req.Id), domain.Role(req.Body.Role))
	switch {
	case errors.Is(err, service.ErrForbidden):
		return api.UpdateUser403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(err)}, nil
	case errors.Is(err, service.ErrInvalid):
		return api.UpdateUser400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(err)}, nil
	case errors.Is(err, service.ErrNotFound):
		return api.UpdateUser404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(err)}, nil
	case errors.Is(err, service.ErrConflict):
		return api.UpdateUser409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: conflict(err)}, nil
	case err != nil:
		return nil, err
	}
	return api.UpdateUser200JSONResponse(toAPIAdminUser(u)), nil
}

func (h *Handler) DeleteUser(ctx context.Context, req api.DeleteUserRequestObject) (api.DeleteUserResponseObject, error) {
	actor, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	err = h.accounts.DeleteUser(actor, domain.UserID(req.Id))
	switch {
	case errors.Is(err, service.ErrForbidden):
		return api.DeleteUser403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(err)}, nil
	case errors.Is(err, service.ErrInvalid):
		return api.DeleteUser400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(err)}, nil
	case errors.Is(err, service.ErrNotFound):
		return api.DeleteUser404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(err)}, nil
	case err != nil:
		return nil, err
	}
	return api.DeleteUser204Response{}, nil
}

func (h *Handler) ResetUserPassword(ctx context.Context, req api.ResetUserPasswordRequestObject) (api.ResetUserPasswordResponseObject, error) {
	actor, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	err = h.accounts.ResetPassword(actor, domain.UserID(req.Id), req.Body.NewPassword)
	switch {
	case errors.Is(err, service.ErrForbidden):
		return api.ResetUserPassword403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(err)}, nil
	case errors.Is(err, service.ErrInvalid):
		return api.ResetUserPassword400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(err)}, nil
	case errors.Is(err, service.ErrNotFound):
		return api.ResetUserPassword404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(err)}, nil
	case err != nil:
		return nil, err
	}
	return api.ResetUserPassword204Response{}, nil
}

func (h *Handler) GetAdminSettings(ctx context.Context, _ api.GetAdminSettingsRequestObject) (api.GetAdminSettingsResponseObject, error) {
	actor, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	open, err := h.accounts.AdminSettings(actor)
	switch {
	case errors.Is(err, service.ErrForbidden):
		return api.GetAdminSettings403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(err)}, nil
	case err != nil:
		return nil, err
	}
	return api.GetAdminSettings200JSONResponse{RegistrationOpen: open}, nil
}

func (h *Handler) SaveAdminSettings(ctx context.Context, req api.SaveAdminSettingsRequestObject) (api.SaveAdminSettingsResponseObject, error) {
	actor, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	err = h.accounts.SetRegistrationOpen(actor, req.Body.RegistrationOpen)
	switch {
	case errors.Is(err, service.ErrForbidden):
		return api.SaveAdminSettings403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: forbidden(err)}, nil
	case err != nil:
		return nil, err
	}
	return api.SaveAdminSettings204Response{}, nil
}
