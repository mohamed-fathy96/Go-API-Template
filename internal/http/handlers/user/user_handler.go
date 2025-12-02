package user

import (
	"encoding/json"
	appuser "kabsa/internal/app/user"
	"kabsa/internal/http/responses"
	"kabsa/internal/logging"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service appuser.Service
	logger  logging.Logger
}

func NewHandler(service appuser.Service, logger logging.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With("component", "user_http_handler"),
	}
}

// List godoc
//
//	@Summary	List users
//	@Tags		users
//	@Produce	json
//	@Success	200	{object}	apidocs.UsersListResponse
//	@Failure	500	{object}	apidocs.ErrorEnvelope
//	@Router		/users [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	users, err := h.service.List(ctx, appuser.ListUsersInput{
		Limit:  50, // TODO: pull from query params
		Offset: 0,
	})
	if err != nil {
		h.logger.Error("failed to list users", "error", err)
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	responses.WriteJSON(w, http.StatusOK, users)
}

// Create godoc
//
//	@Summary	Create user
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		body	body		user.CreateUserRequest	true	"Create payload"
//	@Success	201		{object}	apidocs.UserItemResponse
//	@Failure	400		{object}	apidocs.ErrorEnvelope
//	@Failure	500		{object}	apidocs.ErrorEnvelope
//	@Router		/users [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var input struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.logger.Error("invalid create user payload", "error", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dto, err := h.service.Create(ctx, appuser.CreateUserInput{
		Email: input.Email,
		Name:  input.Name,
	})
	if err != nil {
		h.logger.Error("failed to create user", "error", err)
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	responses.WriteJSON(w, http.StatusCreated, dto)
}

// GetByID godoc
//
//	@Summary	Get user by ID
//	@Tags		users
//	@Produce	json
//	@Param		id	path		int	true	"User ID"
//	@Success	200	{object}	apidocs.UserItemResponse
//	@Failure	404	{object}	apidocs.ErrorEnvelope
//	@Failure	500	{object}	apidocs.ErrorEnvelope
//	@Router		/users/{id} [get]
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	dto, err := h.service.GetById(ctx, id)
	if err != nil {
		if appuser.IsNotFound(err) {
			responses.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("failed to get user", "error", err, "id", id)
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	responses.WriteJSON(w, http.StatusOK, dto)
}

// Update godoc
//
//	@Summary	Update user
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Param		id		path		int								true	"User ID"
//	@Param		body	body		user.UpdateUserRequest	true	"Update payload"
//	@Success	200		{object}	apidocs.UserItemResponse
//	@Failure	400		{object}	apidocs.ErrorEnvelope
//	@Failure	404		{object}	apidocs.ErrorEnvelope
//	@Failure	500		{object}	apidocs.ErrorEnvelope
//	@Router		/users/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var input struct {
		Name *string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.logger.Error("invalid update user payload", "error", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dto, err := h.service.Update(ctx, appuser.UpdateUserInput{
		ID:   id,
		Name: input.Name,
	})
	if err != nil {
		if appuser.IsNotFound(err) {
			responses.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("failed to update user", "error", err, "id", id)
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	responses.WriteJSON(w, http.StatusOK, dto)
}

// Delete godoc
//
//	@Summary	Delete user
//	@Tags		users
//	@Produce	json
//	@Param		id	path		int		true	"User ID"
//	@Success	204	{string}	string	"No Content"
//	@Failure	404	{object}	apidocs.ErrorEnvelope
//	@Failure	500	{object}	apidocs.ErrorEnvelope
//	@Router		/users/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := h.service.Delete(ctx, id); err != nil {
		if appuser.IsNotFound(err) {
			responses.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("failed to delete user", "error", err, "id", id)
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
