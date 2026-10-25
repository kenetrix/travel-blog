package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"travel-blog/backend/internal/adapter/http/api"
	"travel-blog/backend/internal/adapter/http/auth"
	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
)

// Handler implements api.StrictServerInterface and delegates to the services,
// always for the user that auth put in the context.
type Handler struct {
	travel   *service.TravelService
	accounts *service.AccountService
	oidc     OIDC
	oidcConfig OIDCConfig
}

// OIDC tells whether and how OpenID Connect login is offered (oidc.Provider).
// Without it, OIDC is off and password login is open.
type OIDC interface {
	Enabled() bool
	Name() string
	PasswordLogin() bool
}

type Option func(*Handler)

func WithOIDC(o OIDC) Option {
	return func(h *Handler) { h.oidc = o }
}

var _ api.StrictServerInterface = (*Handler)(nil)

func New(travel *service.TravelService, accounts *service.AccountService, opts ...Option) *Handler {
	h := &Handler{travel: travel, accounts: accounts}
	for _, o := range opts {
		o(h)
	}
	return h
}

func (h *Handler) passwordLogin() bool {
	return h.oidc == nil || h.oidc.PasswordLogin()
}

var errPasswordLoginOff = errors.New("password login is disabled, use single sign-on")

// userFrom returns the user of the request. Protected routes never see a request
// without one (see router), so a missing user is a programming error: 500.
func userFrom(ctx context.Context) (domain.UserID, error) {
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return "", errors.New("no user in request context")
	}
	return user, nil
}

// etagOf hashes user and JSON form so the ETags of different users never match.
func etagOf(user domain.UserID, v any) (string, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(append([]byte(user+"\n"), body...))
	return `"` + hex.EncodeToString(sum[:8]) + `"`, nil
}

// accountOf, entriesOf and flightsOf return one resource of the user with its ETag.
// Each has its own, so a change only invalidates its own response.
func (h *Handler) accountOf(user domain.UserID) (api.Account, string, error) {
	u, settings, err := h.accounts.Account(user)
	if err != nil {
		return api.Account{}, "", err
	}
	account := toAPIAccount(u, settings)
	etag, err := etagOf(user, account)
	return account, etag, err
}

func (h *Handler) entriesOf(user domain.UserID) ([]api.CountrySummary, string, error) {
	entries, err := h.travel.Entries(user)
	if err != nil {
		return nil, "", err
	}
	out := toAPISummaries(entries)
	etag, err := etagOf(user, out)
	return out, etag, err
}

func (h *Handler) flightsOf(user domain.UserID) ([]api.Flight, string, error) {
	flights, err := h.travel.Flights(user)
	if err != nil {
		return nil, "", err
	}
	out := toAPIFlights(flights)
	etag, err := etagOf(user, out)
	return out, etag, err
}

func (h *Handler) accountETag(user domain.UserID) (*string, error) {
	_, etag, err := h.accountOf(user)
	return &etag, err
}

func (h *Handler) entriesETag(user domain.UserID) (*string, error) {
	_, etag, err := h.entriesOf(user)
	return &etag, err
}

func (h *Handler) flightsETag(user domain.UserID) (*string, error) {
	_, etag, err := h.flightsOf(user)
	return &etag, err
}

func matches(ifNoneMatch *api.IfNoneMatch, etag string) bool {
	return ifNoneMatch != nil && string(*ifNoneMatch) == etag
}

func problem(status int, err error) api.Problem {
	// Service errors read "<kind>: <text>"; the kind is already in the status
	detail := err.Error()
	if _, text, ok := strings.Cut(detail, ": "); ok {
		detail = text
	}
	return api.Problem{Status: status, Title: http.StatusText(status), Detail: &detail}
}

func badRequest(err error) api.BadRequestApplicationProblemPlusJSONResponse {
	return api.BadRequestApplicationProblemPlusJSONResponse(problem(http.StatusBadRequest, err))
}

func notFound(err error) api.NotFoundApplicationProblemPlusJSONResponse {
	return api.NotFoundApplicationProblemPlusJSONResponse(problem(http.StatusNotFound, err))
}

func unauthorized(err error) api.UnauthorizedApplicationProblemPlusJSONResponse {
	return api.UnauthorizedApplicationProblemPlusJSONResponse(problem(http.StatusUnauthorized, err))
}

func forbidden(err error) api.ForbiddenApplicationProblemPlusJSONResponse {
	return api.ForbiddenApplicationProblemPlusJSONResponse(problem(http.StatusForbidden, err))
}

func conflict(err error) api.ConflictApplicationProblemPlusJSONResponse {
	return api.ConflictApplicationProblemPlusJSONResponse(problem(http.StatusConflict, err))
}


func toAPIEntry(e domain.CountryEntry) api.CountryEntry {
	tags := e.Details.Tags
	if tags == nil {
		tags = []string{}
	}
	return api.CountryEntry{
		Iso:        e.ISO,
		Status:     api.CountryEntryStatus(e.Status),
		Visibility: api.Visibility(e.Visibility),
		Details: api.CountryDetails{
			Food:        e.Details.Food,
			Sightseeing: e.Details.Sightseeing,
			Notes:       e.Details.Notes,
			Rating:      e.Details.Rating,
			Tags:        tags,
		},
	}
}

func toAPIEntries(entries []domain.CountryEntry) []api.CountryEntry {
	out := make([]api.CountryEntry, len(entries))
	for i, e := range entries {
		out[i] = toAPIEntry(e)
	}
	return out
}

// toAPISummaries omits the long texts; GET /countries/{iso} returns them.
func toAPISummaries(entries []domain.CountryEntry) []api.CountrySummary {
	out := make([]api.CountrySummary, len(entries))
	for i, e := range entries {
		full := toAPIEntry(e)
		out[i] = api.CountrySummary{
			Iso:        full.Iso,
			Status:     api.CountrySummaryStatus(full.Status),
			Visibility: full.Visibility,
			Rating:     full.Details.Rating,
			Tags:       full.Details.Tags,
		}
	}
	return out
}

func toAPIFlights(flights []domain.Flight) []api.Flight {
	out := make([]api.Flight, len(flights))
	for i, f := range flights {
		out[i] = api.Flight{
			Id:         uuid.MustParse(f.ID),
			From:       f.From,
			To:         f.To,
			Visibility: api.Visibility(f.Visibility),
			CreatedAt:  f.CreatedAt,
		}
	}
	return out
}

func toAPIProfile(p domain.Profile) api.Profile {
	out := api.Profile{
		Username:    p.Username,
		FirstName:   p.FirstName,
		LastName:    p.LastName,
		DisplayName: p.DisplayName,
		Bio:         p.Bio,
		Language:    p.Language,
		Timezone:    p.Timezone,
		HomeCountry: p.HomeCountry,
		Color:       p.Color,
		CreatedAt:   p.CreatedAt,
	}
	if p.Email != "" {
		email := openapi_types.Email(p.Email)
		out.Email = &email
	}
	return out
}

func toAPISettings(s domain.Settings) api.Settings {
	return api.Settings{
		DistanceUnit:      api.SettingsDistanceUnit(s.DistanceUnit),
		Theme:             api.SettingsTheme(s.Theme),
		DefaultView:       api.SettingsDefaultView(s.DefaultView),
		DefaultVisibility: api.Visibility(s.DefaultVisibility),
		ProfileVisibility: api.Visibility(s.ProfileVisibility),
	}
}

func toAPIAccount(u domain.User, s domain.Settings) api.Account {
	return api.Account{
		Profile:            toAPIProfile(u.Profile),
		Settings:           toAPISettings(s),
		Role:               api.Role(u.Role),
		MustChangePassword: u.MustChangePassword,
	}
}

func visibilityOf(v *api.Visibility) *domain.Visibility {
	if v == nil {
		return nil
	}
	dv := domain.Visibility(*v)
	return &dv
}
