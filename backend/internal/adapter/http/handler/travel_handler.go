package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"travel-blog/backend/internal/adapter/http/api"
	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
)

func (h *Handler) GetAccount(ctx context.Context, req api.GetAccountRequestObject) (api.GetAccountResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	account, etag, err := h.accountOf(user)
	if err != nil {
		return nil, err
	}
	if matches(req.Params.IfNoneMatch, etag) {
		return api.GetAccount304Response{Headers: api.GetAccount304ResponseHeaders{ETag: &etag}}, nil
	}
	return api.GetAccount200JSONResponse{Body: account, Headers: api.GetAccount200ResponseHeaders{ETag: &etag}}, nil
}

func (h *Handler) ListCountries(ctx context.Context, req api.ListCountriesRequestObject) (api.ListCountriesResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	entries, etag, err := h.entriesOf(user)
	if err != nil {
		return nil, err
	}
	if matches(req.Params.IfNoneMatch, etag) {
		return api.ListCountries304Response{Headers: api.ListCountries304ResponseHeaders{ETag: &etag}}, nil
	}
	return api.ListCountries200JSONResponse{Body: entries, Headers: api.ListCountries200ResponseHeaders{ETag: &etag}}, nil
}

func (h *Handler) ListFlights(ctx context.Context, req api.ListFlightsRequestObject) (api.ListFlightsResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	flights, etag, err := h.flightsOf(user)
	if err != nil {
		return nil, err
	}
	if matches(req.Params.IfNoneMatch, etag) {
		return api.ListFlights304Response{Headers: api.ListFlights304ResponseHeaders{ETag: &etag}}, nil
	}
	return api.ListFlights200JSONResponse{Body: flights, Headers: api.ListFlights200ResponseHeaders{ETag: &etag}}, nil
}

func (h *Handler) GetCountry(ctx context.Context, req api.GetCountryRequestObject) (api.GetCountryResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	entry, err := h.travel.Country(user, req.Iso)
	if errors.Is(err, service.ErrNotFound) {
		return api.GetCountry404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(err)}, nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetCountry200JSONResponse(toAPIEntry(entry)), nil
}

func (h *Handler) SaveCountry(ctx context.Context, req api.SaveCountryRequestObject) (api.SaveCountryResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	in := req.Body
	entry, err := h.travel.SaveCountry(user, req.Iso, domain.CountryStatus(in.Status), visibilityOf(in.Visibility), domain.CountryDetails{
		Food:        in.Details.Food,
		Sightseeing: in.Details.Sightseeing,
		Notes:       in.Details.Notes,
		Rating:      in.Details.Rating,
		Tags:        in.Details.Tags,
	})
	switch {
	case errors.Is(err, service.ErrInvalid):
		return api.SaveCountry400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(err)}, nil
	case err != nil:
		return nil, err
	}

	etag, err := h.entriesETag(user)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return api.SaveCountry204Response{Headers: api.SaveCountry204ResponseHeaders{ETag: etag}}, nil
	}
	return api.SaveCountry200JSONResponse{Body: toAPIEntry(*entry), Headers: api.SaveCountry200ResponseHeaders{ETag: etag}}, nil
}

func (h *Handler) CreateFlight(ctx context.Context, req api.CreateFlightRequestObject) (api.CreateFlightResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	in := req.Body
	flight, updated, err := h.travel.CreateFlight(user, in.From, in.To, in.Countries, visibilityOf(in.Visibility))
	switch {
	case errors.Is(err, service.ErrInvalid):
		return api.CreateFlight400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(err)}, nil
	case errors.Is(err, service.ErrDuplicate):
		return api.CreateFlight409ApplicationProblemPlusJSONResponse(problem(http.StatusConflict, err)), nil
	case errors.Is(err, service.ErrSameAirport):
		return api.CreateFlight422ApplicationProblemPlusJSONResponse(problem(http.StatusUnprocessableEntity, err)), nil
	case err != nil:
		return nil, err
	}

	etag, err := h.flightsETag(user)
	if err != nil {
		return nil, err
	}
	return api.CreateFlight201JSONResponse{
		Body: api.FlightCreated{
			Id:               uuid.MustParse(flight.ID),
			Visibility:       api.Visibility(flight.Visibility),
			CreatedAt:        flight.CreatedAt,
			CountriesUpdated: updated,
		},
		Headers: api.CreateFlight201ResponseHeaders{ETag: etag},
	}, nil
}

func (h *Handler) UpdateFlight(ctx context.Context, req api.UpdateFlightRequestObject) (api.UpdateFlightResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	in := req.Body
	err = h.travel.UpdateFlight(user, req.Id.String(), domain.FlightUpdate{
		Visibility:      visibilityOf(in.Visibility),
		Number:          in.FlightNumber,
		Date:            in.Date,
		DurationMinutes: in.DurationMinutes,
		Aircraft:        in.Aircraft,
	})
	switch {
	case errors.Is(err, service.ErrInvalid):
		return api.UpdateFlight400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(err)}, nil
	case errors.Is(err, service.ErrNotFound):
		return api.UpdateFlight404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(err)}, nil
	case err != nil:
		return nil, err
	}
	etag, err := h.flightsETag(user)
	if err != nil {
		return nil, err
	}
	return api.UpdateFlight204Response{Headers: api.UpdateFlight204ResponseHeaders{ETag: etag}}, nil
}

func (h *Handler) DeleteFlight(ctx context.Context, req api.DeleteFlightRequestObject) (api.DeleteFlightResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	err = h.travel.DeleteFlight(user, req.Id.String())
	if errors.Is(err, service.ErrNotFound) {
		return api.DeleteFlight404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(err)}, nil
	}
	if err != nil {
		return nil, err
	}
	etag, err := h.flightsETag(user)
	if err != nil {
		return nil, err
	}
	return api.DeleteFlight204Response{Headers: api.DeleteFlight204ResponseHeaders{ETag: etag}}, nil
}

func (h *Handler) GetPublicTravel(_ context.Context, req api.GetPublicTravelRequestObject) (api.GetPublicTravelResponseObject, error) {
	p, entries, flights, err := h.travel.PublicTravel(req.Username)
	if errors.Is(err, service.ErrNotFound) {
		return api.GetPublicTravel404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(err)}, nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetPublicTravel200JSONResponse{
		Profile: api.PublicProfile{
			Username:    p.Username,
			DisplayName: p.DisplayName,
			Bio:         p.Bio,
			HomeCountry: p.HomeCountry,
		},
		Entries: toAPIEntries(entries),
		// Date and flight number would tell when the user was where
		Flights: toAPIFlights(flights, false),
	}, nil
}
