package handler

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"travel-blog/backend/internal/adapter/http/api"
	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
)

func (h *Handler) ListTrips(ctx context.Context, req api.ListTripsRequestObject) (api.ListTripsResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	trips, etag, err := h.tripsOf(user)
	if err != nil {
		return nil, err
	}
	if matches(req.Params.IfNoneMatch, etag) {
		return api.ListTrips304Response{Headers: api.ListTrips304ResponseHeaders{ETag: &etag}}, nil
	}
	return api.ListTrips200JSONResponse{Body: trips, Headers: api.ListTrips200ResponseHeaders{ETag: &etag}}, nil
}

func (h *Handler) GetTrip(ctx context.Context, req api.GetTripRequestObject) (api.GetTripResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	trip, err := h.trips.Trip(user, req.Id.String())
	if errors.Is(err, service.ErrNotFound) {
		return api.GetTrip404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(err)}, nil
	}
	if err != nil {
		return nil, err
	}
	return api.GetTrip200JSONResponse(toAPITrip(trip)), nil
}

func (h *Handler) CreateTrip(ctx context.Context, req api.CreateTripRequestObject) (api.CreateTripResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	trip, err := h.trips.CreateTrip(user, tripInputOf(*req.Body))
	switch {
	case errors.Is(err, service.ErrInvalid):
		return api.CreateTrip400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(err)}, nil
	case err != nil:
		return nil, err
	}
	etag, err := h.tripsETag(user)
	if err != nil {
		return nil, err
	}
	return api.CreateTrip201JSONResponse{Body: toAPITrip(trip), Headers: api.CreateTrip201ResponseHeaders{ETag: etag}}, nil
}

func (h *Handler) ReplaceTrip(ctx context.Context, req api.ReplaceTripRequestObject) (api.ReplaceTripResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	trip, err := h.trips.ReplaceTrip(user, req.Id.String(), tripInputOf(*req.Body))
	switch {
	case errors.Is(err, service.ErrInvalid):
		return api.ReplaceTrip400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: badRequest(err)}, nil
	case errors.Is(err, service.ErrNotFound):
		return api.ReplaceTrip404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(err)}, nil
	case err != nil:
		return nil, err
	}
	etag, err := h.tripsETag(user)
	if err != nil {
		return nil, err
	}
	return api.ReplaceTrip200JSONResponse{Body: toAPITrip(trip), Headers: api.ReplaceTrip200ResponseHeaders{ETag: etag}}, nil
}

func (h *Handler) DeleteTrip(ctx context.Context, req api.DeleteTripRequestObject) (api.DeleteTripResponseObject, error) {
	user, err := userFrom(ctx)
	if err != nil {
		return nil, err
	}
	err = h.trips.DeleteTrip(user, req.Id.String())
	if errors.Is(err, service.ErrNotFound) {
		return api.DeleteTrip404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound(err)}, nil
	}
	if err != nil {
		return nil, err
	}
	etag, err := h.tripsETag(user)
	if err != nil {
		return nil, err
	}
	return api.DeleteTrip204Response{Headers: api.DeleteTrip204ResponseHeaders{ETag: etag}}, nil
}

func (h *Handler) tripsOf(user domain.UserID) ([]api.TripSummary, string, error) {
	trips, err := h.trips.Trips(user)
	if err != nil {
		return nil, "", err
	}
	out := make([]api.TripSummary, len(trips))
	for i, t := range trips {
		out[i] = api.TripSummary{
			Id:         uuid.MustParse(t.ID),
			Title:      t.Title,
			StartsOn:   dateOf(t.StartsOn),
			EndsOn:     dateOf(t.EndsOn),
			Visibility: api.Visibility(t.Visibility),
			Countries:  t.Countries,
			StopCount:  t.StopCount,
			CreatedAt:  t.CreatedAt,
			UpdatedAt:  t.UpdatedAt,
		}
	}
	etag, err := etagOf(user, out)
	return out, etag, err
}

func (h *Handler) tripsETag(user domain.UserID) (*string, error) {
	_, etag, err := h.tripsOf(user)
	return &etag, err
}

func tripInputOf(in api.TripInput) service.TripInput {
	stops := make([]domain.Location, len(in.Stops))
	for i, s := range in.Stops {
		stops[i] = domain.Location{
			Name:      s.Name,
			CountryID: s.Country,
			Lat:       s.Lat,
			Lng:       s.Lng,
			VisitedAt: deref(s.Arrival),
			Notes:     deref(s.Notes),
		}
		if s.Id != nil {
			stops[i].ID = s.Id.String()
		}
	}
	return service.TripInput{
		Title:      in.Title,
		StartsOn:   deref(in.StartsOn),
		EndsOn:     deref(in.EndsOn),
		Notes:      deref(in.Notes),
		Visibility: visibilityOf(in.Visibility),
		Stops:      stops,
	}
}

func toAPITrip(t domain.Trip) api.Trip {
	stops := make([]api.TripStop, len(t.Stops))
	for i, s := range t.Stops {
		stops[i] = api.TripStop{
			Id:      uuid.MustParse(s.ID),
			Name:    s.Name,
			Country: s.CountryID,
			Lat:     s.Lat,
			Lng:     s.Lng,
			Arrival: dateOf(s.VisitedAt),
			Notes:   s.Notes,
		}
	}
	return api.Trip{
		Id:         uuid.MustParse(t.ID),
		Title:      t.Title,
		StartsOn:   dateOf(t.StartsOn),
		EndsOn:     dateOf(t.EndsOn),
		Notes:      t.Notes,
		Visibility: api.Visibility(t.Visibility),
		Countries:  t.Countries(),
		StopCount:  len(t.Stops),
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
		Stops:      stops,
	}
}

// dateOf returns nil for an empty or malformed day, so the field is left out of the JSON.
func dateOf(s string) *openapi_types.Date {
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return nil
	}
	return &openapi_types.Date{Time: d}
}
