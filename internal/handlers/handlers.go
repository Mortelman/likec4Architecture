package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"marketplace/internal/auth"
	"marketplace/internal/domain"
	"marketplace/internal/middleware"
	"marketplace/internal/models"
	"marketplace/internal/repository"
	"marketplace/internal/service"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Handler handles HTTP requests
type Handler struct {
	service    *service.Service
	repo       *repository.Repository
	jwtManager *auth.JWTManager
}

// NewHandler creates a new handler instance
func NewHandler(service *service.Service, repo *repository.Repository, jwtManager *auth.JWTManager) *Handler {
	return &Handler{
		service:    service,
		repo:       repo,
		jwtManager: jwtManager,
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func ptrString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code models.ErrorCode, message string, details map[string]interface{}) {
	errorResp := models.ErrorResponse{
		ErrorCode: code,
		Message:   message,
		Timestamp: time.Now(),
	}
	if details != nil {
		errorResp.Details = make(map[string]string)
		for k, v := range details {
			errorResp.Details[k] = fmt.Sprint(v)
		}
	}
	writeJSON(w, status, errorResp)
}

func getUserFromContext(r *http.Request) *auth.Claims {
	user, _ := r.Context().Value(middleware.UserContextKey).(*auth.Claims)
	return user
}

// ============================================================================
// HEALTH CHECK
// ============================================================================

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ============================================================================
// AUTH HANDLERS
// ============================================================================

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.UserRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, models.ErrorCodeVALIDATION_ERROR, "Invalid request body", nil)
		return
	}

	user, err := h.service.Register(r.Context(), req.Email, req.Password, string(req.Role))
	if err != nil {
		writeError(w, http.StatusInternalServerError, models.ErrorCodeVALIDATION_ERROR, err.Error(), nil)
		return
	}

	response := models.UserResponse{
		Id:        user.ID,
		Email:     user.Email,
		Role:      models.UserRole(user.Role),
		CreatedAt: user.CreatedAt,
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.UserLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, models.ErrorCodeVALIDATION_ERROR, "Invalid request body", nil)
		return
	}

	user, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, models.ErrorCodeVALIDATION_ERROR, "Invalid credentials", nil)
		return
	}

	tokens, err := h.jwtManager.GenerateTokens(user.ID, user.Email, user.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, models.ErrorCodeVALIDATION_ERROR, err.Error(), nil)
		return
	}

	response := models.AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: models.UserResponse{
			Id:        user.ID,
			Email:     user.Email,
			Role:      models.UserRole(user.Role),
			CreatedAt: user.CreatedAt,
		},
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, models.ErrorCodeVALIDATION_ERROR, "Invalid request body", nil)
		return
	}

	claims, err := h.jwtManager.ValidateToken(req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, models.ErrorCodeREFRESH_TOKEN_INVALID, "Invalid refresh token", nil)
		return
	}

	tokens, err := h.jwtManager.GenerateTokens(claims.UserID, claims.Email, claims.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, models.ErrorCodeVALIDATION_ERROR, err.Error(), nil)
		return
	}

	user, err := h.repo.GetUserByEmail(r.Context(), claims.Email)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, models.ErrorCodeREFRESH_TOKEN_INVALID, "User not found", nil)
		return
	}

	response := models.AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: models.UserResponse{
			Id:        user.ID,
			Email:     user.Email,
			Role:      models.UserRole(user.Role),
			CreatedAt: user.CreatedAt,
		},
	}
	writeJSON(w, http.StatusOK, response)
}

// ============================================================================
// PRODUCT HANDLERS
// ============================================================================

func (h *Handler) GetProducts(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	_ = user // All roles can view products

	page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	size, _ := strconv.ParseInt(r.URL.Query().Get("size"), 10, 32)
	if size == 0 {
		size = 20
	}
	status := r.URL.Query().Get("status")
	category := r.URL.Query().Get("category")

	products, total, err := h.repo.GetProducts(r.Context(), int32(page), int32(size), status, category)
	if err != nil {
		writeError(w, http.StatusInternalServerError, models.ErrorCodeVALIDATION_ERROR, err.Error(), nil)
		return
	}

	content := []models.ProductResponse{}
	for _, p := range products {
		resp := models.ProductResponse{
			Id:          p.ID,
			Name:        p.Name,
			Description: ptrString(p.Description),
			Price:       p.Price,
			Stock:       p.Stock,
			Category:    p.Category,
			Status:      models.ProductStatus(p.Status),
			SellerId:    p.SellerID,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		}
		content = append(content, resp)
	}

	totalPages := (total + int(size) - 1) / int(size)
	response := models.ProductPageResponse{
		Content:       content,
		TotalElements: int32(total),
		PageNumber:    int32(page),
		PageSize:      int32(size),
		TotalPages:    int32(totalPages),
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)

	if user.Role == "USER" {
		writeError(w, http.StatusForbidden, models.ErrorCodeACCESS_DENIED, "Users cannot create products", nil)
		return
	}

	var req models.ProductCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, models.ErrorCodeVALIDATION_ERROR, "Invalid request body", nil)
		return
	}

	sellerID := user.UserID
	// ADMIN can create for anyone, SELLER creates for themselves

	product, err := h.service.CreateProduct(r.Context(), req, sellerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, models.ErrorCodeVALIDATION_ERROR, err.Error(), nil)
		return
	}

	response := models.ProductResponse{
		Id:          product.ID,
		Name:        product.Name,
		Description: ptrString(product.Description),
		Price:       product.Price,
		Stock:       product.Stock,
		Category:    product.Category,
		Status:      models.ProductStatus(product.Status),
		SellerId:    product.SellerID,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	product, err := h.repo.GetProductByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, models.ErrorCodeVALIDATION_ERROR, err.Error(), nil)
		return
	}
	if product == nil {
		writeError(w, http.StatusNotFound, models.ErrorCodePRODUCT_NOT_FOUND, "Product not found", nil)
		return
	}

	response := models.ProductResponse{
		Id:          product.ID,
		Name:        product.Name,
		Description: ptrString(product.Description),
		Price:       product.Price,
		Stock:       product.Stock,
		Category:    product.Category,
		Status:      models.ProductStatus(product.Status),
		SellerId:    product.SellerID,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	id := r.PathValue("id")

	if user.Role == "USER" {
		writeError(w, http.StatusForbidden, models.ErrorCodeACCESS_DENIED, "Users cannot update products", nil)
		return
	}

	existing, err := h.repo.GetProductByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, models.ErrorCodeVALIDATION_ERROR, err.Error(), nil)
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, models.ErrorCodePRODUCT_NOT_FOUND, "Product not found", nil)
		return
	}

	if user.Role == "SELLER" && existing.SellerID != user.UserID {
		writeError(w, http.StatusForbidden, models.ErrorCodeACCESS_DENIED, "You can only update your own products", nil)
		return
	}

	var req models.ProductUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, models.ErrorCodeVALIDATION_ERROR, "Invalid request body", nil)
		return
	}

	product := &domain.Product{
		Name:        req.Name,
		Description: &req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
		Status:      string(req.Status),
	}
	if req.Description == "" {
		product.Description = nil
	}

	if err := h.repo.UpdateProduct(r.Context(), id, product); err != nil {
		writeError(w, http.StatusInternalServerError, models.ErrorCodeVALIDATION_ERROR, err.Error(), nil)
		return
	}

	product.ID = id
	product.SellerID = existing.SellerID
	product.CreatedAt = existing.CreatedAt

	response := models.ProductResponse{
		Id:          product.ID,
		Name:        product.Name,
		Description: ptrString(product.Description),
		Price:       product.Price,
		Stock:       product.Stock,
		Category:    product.Category,
		Status:      models.ProductStatus(product.Status),
		SellerId:    product.SellerID,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) SoftDeleteProduct(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	id := r.PathValue("id")

	if user.Role == "USER" {
		writeError(w, http.StatusForbidden, models.ErrorCodeACCESS_DENIED, "Users cannot delete products", nil)
		return
	}

	existing, err := h.repo.GetProductByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, models.ErrorCodeVALIDATION_ERROR, err.Error(), nil)
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, models.ErrorCodePRODUCT_NOT_FOUND, "Product not found", nil)
		return
	}

	if user.Role == "SELLER" && existing.SellerID != user.UserID {
		writeError(w, http.StatusForbidden, models.ErrorCodeACCESS_DENIED, "You can only delete your own products", nil)
		return
	}

	if err := h.repo.SoftDeleteProduct(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, models.ErrorCodePRODUCT_NOT_FOUND, "Product not found", nil)
		} else {
			writeError(w, http.StatusInternalServerError, models.ErrorCodeVALIDATION_ERROR, err.Error(), nil)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// ORDER HANDLERS
// ============================================================================

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)

	if user.Role == "SELLER" {
		writeError(w, http.StatusForbidden, models.ErrorCodeACCESS_DENIED, "Sellers cannot create orders", nil)
		return
	}

	var req models.OrderCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, models.ErrorCodeVALIDATION_ERROR, "Invalid request body", nil)
		return
	}

	order, err := h.service.CreateOrder(r.Context(), req, user.UserID)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			status := http.StatusConflict
			if bizErr.Code == string(models.ErrorCodeORDER_LIMIT_EXCEEDED) {
				status = http.StatusTooManyRequests
			} else if bizErr.Code == string(models.ErrorCodePRODUCT_NOT_FOUND) {
				status = http.StatusNotFound
			} else if strings.HasPrefix(bizErr.Code, "PROMO_CODE") {
				status = http.StatusUnprocessableEntity
			}
			writeError(w, status, models.ErrorCode(bizErr.Code), bizErr.Message, bizErr.Details)
		} else {
			writeError(w, http.StatusInternalServerError, models.ErrorCodeVALIDATION_ERROR, err.Error(), nil)
		}
		return
	}

	items := []models.OrderItemResponse{}
	for _, item := range order.Items {
		items = append(items, models.OrderItemResponse{
			Id:           item.ID,
			ProductId:    item.ProductID,
			Quantity:     item.Quantity,
			PriceAtOrder: item.PriceAtOrder,
		})
	}

	response := models.OrderResponse{
		Id:             order.ID,
		UserId:         order.UserID,
		Status:         models.OrderStatus(order.Status),
		PromoCodeId:    ptrString(order.PromoCodeID),
		TotalAmount:    order.TotalAmount,
		DiscountAmount: order.DiscountAmount,
		Items:          items,
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) GetOrderByID(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	id := r.PathValue("id")

	if user.Role == "SELLER" {
		writeError(w, http.StatusForbidden, models.ErrorCodeACCESS_DENIED, "Sellers cannot view orders", nil)
		return
	}

	order, err := h.repo.GetOrderByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, models.ErrorCodeVALIDATION_ERROR, err.Error(), nil)
		return
	}
	if order == nil {
		writeError(w, http.StatusNotFound, models.ErrorCodeORDER_NOT_FOUND, "Order not found", nil)
		return
	}

	if user.Role == "USER" && order.UserID != user.UserID {
		writeError(w, http.StatusForbidden, models.ErrorCodeORDER_OWNERSHIP_VIOLATION, "You can only view your own orders", nil)
		return
	}

	items := []models.OrderItemResponse{}
	for _, item := range order.Items {
		items = append(items, models.OrderItemResponse{
			Id:           item.ID,
			ProductId:    item.ProductID,
			Quantity:     item.Quantity,
			PriceAtOrder: item.PriceAtOrder,
		})
	}

	response := models.OrderResponse{
		Id:             order.ID,
		UserId:         order.UserID,
		Status:         models.OrderStatus(order.Status),
		PromoCodeId:    ptrString(order.PromoCodeID),
		TotalAmount:    order.TotalAmount,
		DiscountAmount: order.DiscountAmount,
		Items:          items,
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	id := r.PathValue("id")

	if user.Role == "SELLER" {
		writeError(w, http.StatusForbidden, models.ErrorCodeACCESS_DENIED, "Sellers cannot update orders", nil)
		return
	}

	var req models.OrderUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, models.ErrorCodeVALIDATION_ERROR, "Invalid request body", nil)
		return
	}

	order, err := h.service.UpdateOrder(r.Context(), id, req, user.UserID, user.Role)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			status := http.StatusConflict
			if bizErr.Code == string(models.ErrorCodeORDER_LIMIT_EXCEEDED) {
				status = http.StatusTooManyRequests
			} else if bizErr.Code == string(models.ErrorCodeORDER_NOT_FOUND) {
				status = http.StatusNotFound
			} else if bizErr.Code == string(models.ErrorCodeORDER_OWNERSHIP_VIOLATION) {
				status = http.StatusForbidden
			}
			writeError(w, status, models.ErrorCode(bizErr.Code), bizErr.Message, bizErr.Details)
		} else {
			writeError(w, http.StatusInternalServerError, models.ErrorCodeVALIDATION_ERROR, err.Error(), nil)
		}
		return
	}

	items := []models.OrderItemResponse{}
	for _, item := range order.Items {
		items = append(items, models.OrderItemResponse{
			Id:           item.ID,
			ProductId:    item.ProductID,
			Quantity:     item.Quantity,
			PriceAtOrder: item.PriceAtOrder,
		})
	}

	response := models.OrderResponse{
		Id:             order.ID,
		UserId:         order.UserID,
		Status:         models.OrderStatus(order.Status),
		PromoCodeId:    ptrString(order.PromoCodeID),
		TotalAmount:    order.TotalAmount,
		DiscountAmount: order.DiscountAmount,
		Items:          items,
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	id := r.PathValue("id")

	if user.Role == "SELLER" {
		writeError(w, http.StatusForbidden, models.ErrorCodeACCESS_DENIED, "Sellers cannot cancel orders", nil)
		return
	}

	err := h.service.CancelOrder(r.Context(), id, user.UserID, user.Role)
	if err != nil {
		if bizErr, ok := err.(*domain.BusinessError); ok {
			status := http.StatusConflict
			if bizErr.Code == string(models.ErrorCodeORDER_NOT_FOUND) {
				status = http.StatusNotFound
			} else if bizErr.Code == string(models.ErrorCodeORDER_OWNERSHIP_VIOLATION) {
				status = http.StatusForbidden
			}
			writeError(w, status, models.ErrorCode(bizErr.Code), bizErr.Message, bizErr.Details)
		} else {
			writeError(w, http.StatusInternalServerError, models.ErrorCodeVALIDATION_ERROR, err.Error(), nil)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// PROMO CODE HANDLERS
// ============================================================================

func (h *Handler) CreatePromoCode(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)

	if user.Role == "USER" {
		writeError(w, http.StatusForbidden, models.ErrorCodeACCESS_DENIED, "Users cannot create promo codes", nil)
		return
	}

	var req models.PromoCodeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, models.ErrorCodeVALIDATION_ERROR, "Invalid request body", nil)
		return
	}

	promo, err := h.service.CreatePromoCode(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, models.ErrorCodeVALIDATION_ERROR, err.Error(), nil)
		return
	}

	response := models.PromoCodeResponse{
		Id:             promo.ID,
		Code:           promo.Code,
		DiscountType:   models.DiscountType(promo.DiscountType),
		DiscountValue:  promo.DiscountValue,
		MinOrderAmount: promo.MinOrderAmount,
		MaxUses:        promo.MaxUses,
		CurrentUses:    promo.CurrentUses,
		ValidFrom:      promo.ValidFrom,
		ValidUntil:     promo.ValidUntil,
		Active:         promo.Active,
		CreatedAt:      promo.CreatedAt,
	}
	writeJSON(w, http.StatusCreated, response)
}
