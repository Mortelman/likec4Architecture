package service

import (
	"context"
	"database/sql"
	"fmt"
	"marketplace/internal/domain"
	"marketplace/internal/models"
	"marketplace/internal/repository"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Service handles business logic
type Service struct {
	repo                  *repository.Repository
	db                    *sql.DB
	orderRateLimitMinutes int
}

// NewService creates a new service instance
func NewService(repo *repository.Repository, db *sql.DB, orderRateLimitMinutes int) *Service {
	return &Service{
		repo:                  repo,
		db:                    db,
		orderRateLimitMinutes: orderRateLimitMinutes,
	}
}

// ============================================================================
// AUTH SERVICE METHODS
// ============================================================================

// Register creates a new user
func (s *Service) Register(ctx context.Context, email, password, role string) (*domain.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return s.repo.CreateUser(ctx, email, string(hash), role)
}

// Login authenticates a user
func (s *Service) Login(ctx context.Context, email, password string) (*domain.User, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return user, nil
}

// ============================================================================
// PRODUCT SERVICE METHODS
// ============================================================================

// CreateProduct creates a new product
func (s *Service) CreateProduct(ctx context.Context, req models.ProductCreateRequest, sellerID string) (*domain.Product, error) {
	product := &domain.Product{
		Name:        req.Name,
		Description: &req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
		Status:      string(req.Status),
		SellerID:    sellerID,
	}
	if req.Description == "" {
		product.Description = nil
	}
	err := s.repo.CreateProduct(ctx, product)
	return product, err
}

// ============================================================================
// ORDER SERVICE METHODS (Complex Business Logic)
// ============================================================================

// CreateOrder creates a new order with full business logic validation
func (s *Service) CreateOrder(ctx context.Context, req models.OrderCreateRequest, userID string) (*domain.Order, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Check rate limit
	lastOp, err := s.repo.GetLastOperation(ctx, tx, userID, "CREATE_ORDER")
	if err != nil {
		return nil, err
	}
	if lastOp != nil && time.Since(lastOp.CreatedAt) < time.Duration(s.orderRateLimitMinutes)*time.Minute {
		return nil, &domain.BusinessError{
			Code:    string(models.ErrorCodeORDER_LIMIT_EXCEEDED),
			Message: "Too many order creation attempts",
		}
	}

	// 2. Check active orders
	hasActive, err := s.repo.HasActiveOrder(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	if hasActive {
		return nil, &domain.BusinessError{
			Code:    string(models.ErrorCodeORDER_HAS_ACTIVE),
			Message: "User already has an active order",
		}
	}

	// 3-4. Validate products and check stock
	var totalAmount float64
	items := []domain.OrderItem{}
	for _, item := range req.Items {
		product, err := s.repo.GetProductByID(ctx, item.ProductId)
		if err != nil {
			return nil, err
		}
		if product == nil {
			return nil, &domain.BusinessError{
				Code:    string(models.ErrorCodePRODUCT_NOT_FOUND),
				Message: fmt.Sprintf("Product %s not found", item.ProductId),
			}
		}
		if product.Status != "ACTIVE" {
			return nil, &domain.BusinessError{
				Code:    string(models.ErrorCodePRODUCT_INACTIVE),
				Message: fmt.Sprintf("Product %s is not active", item.ProductId),
			}
		}
		if product.Stock < item.Quantity {
			return nil, &domain.BusinessError{
				Code:    string(models.ErrorCodeINSUFFICIENT_STOCK),
				Message: "Insufficient stock",
				Details: map[string]interface{}{
					"product_id": item.ProductId,
					"requested":  item.Quantity,
					"available":  product.Stock,
				},
			}
		}

		// 5. Reserve stock
		if err := s.repo.UpdateProductStock(ctx, tx, item.ProductId, -item.Quantity); err != nil {
			return nil, err
		}

		// 6. Snapshot price
		orderItem := domain.OrderItem{
			ProductID:    item.ProductId,
			Quantity:     item.Quantity,
			PriceAtOrder: product.Price,
		}
		items = append(items, orderItem)
		totalAmount += product.Price * float64(item.Quantity)
	}

	// 7. Calculate discount if promo code provided
	var promoCodeID *string
	discountAmount := 0.0
	if req.PromoCode != "" {
		promo, err := s.repo.GetPromoCodeByCode(ctx, tx, req.PromoCode)
		if err != nil {
			return nil, err
		}
		if promo == nil || !promo.Active || promo.CurrentUses >= promo.MaxUses ||
			time.Now().Before(promo.ValidFrom) || time.Now().After(promo.ValidUntil) {
			return nil, &domain.BusinessError{
				Code:    string(models.ErrorCodePROMO_CODE_INVALID),
				Message: "Promo code is invalid or expired",
			}
		}
		if totalAmount < promo.MinOrderAmount {
			return nil, &domain.BusinessError{
				Code:    string(models.ErrorCodePROMO_CODE_MIN_AMOUNT),
				Message: "Order amount is below minimum for promo code",
			}
		}

		// Calculate discount
		if promo.DiscountType == "PERCENTAGE" {
			discountAmount = totalAmount * promo.DiscountValue / 100
			// Max 70% discount
			if discountAmount > totalAmount*0.7 {
				discountAmount = totalAmount * 0.7
			}
		} else {
			discountAmount = promo.DiscountValue
			if discountAmount > totalAmount {
				discountAmount = totalAmount
			}
		}

		// Increment promo code usage
		if err := s.repo.UpdatePromoCodeUsage(ctx, tx, promo.ID, 1); err != nil {
			return nil, err
		}
		promoCodeID = &promo.ID
	}

	// Create order
	order := &domain.Order{
		UserID:         userID,
		Status:         "CREATED",
		PromoCodeID:    promoCodeID,
		TotalAmount:    totalAmount - discountAmount,
		DiscountAmount: discountAmount,
		Items:          items,
	}

	if err := s.repo.CreateOrder(ctx, tx, order); err != nil {
		return nil, err
	}

	// 8. Record operation
	if err := s.repo.RecordOperation(ctx, tx, userID, "CREATE_ORDER"); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return order, nil
}

// UpdateOrder updates an existing order with full business logic
func (s *Service) UpdateOrder(ctx context.Context, orderID string, req models.OrderUpdateRequest, userID, userRole string) (*domain.Order, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Get existing order
	existingOrder, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if existingOrder == nil {
		return nil, &domain.BusinessError{
			Code:    string(models.ErrorCodeORDER_NOT_FOUND),
			Message: "Order not found",
		}
	}

	// 1. Check ownership
	if userRole != "ADMIN" && existingOrder.UserID != userID {
		return nil, &domain.BusinessError{
			Code:    string(models.ErrorCodeORDER_OWNERSHIP_VIOLATION),
			Message: "You can only update your own orders",
		}
	}

	// 2. Check state
	if existingOrder.Status != "CREATED" {
		return nil, &domain.BusinessError{
			Code:    string(models.ErrorCodeINVALID_STATE_TRANSITION),
			Message: "Order can only be updated in CREATED state",
		}
	}

	// 3. Check rate limit
	lastOp, err := s.repo.GetLastOperation(ctx, tx, userID, "UPDATE_ORDER")
	if err != nil {
		return nil, err
	}
	if lastOp != nil && time.Since(lastOp.CreatedAt) < time.Duration(s.orderRateLimitMinutes)*time.Minute {
		return nil, &domain.BusinessError{
			Code:    string(models.ErrorCodeORDER_LIMIT_EXCEEDED),
			Message: "Too many order update attempts",
		}
	}

	// 4. Return previous stock
	for _, item := range existingOrder.Items {
		if err := s.repo.UpdateProductStock(ctx, tx, item.ProductID, item.Quantity); err != nil {
			return nil, err
		}
	}

	// 5. Validate and reserve new items
	var totalAmount float64
	newItems := []domain.OrderItem{}
	for _, item := range req.Items {
		product, err := s.repo.GetProductByID(ctx, item.ProductId)
		if err != nil {
			return nil, err
		}
		if product == nil {
			return nil, &domain.BusinessError{
				Code:    string(models.ErrorCodePRODUCT_NOT_FOUND),
				Message: fmt.Sprintf("Product %s not found", item.ProductId),
			}
		}
		if product.Status != "ACTIVE" {
			return nil, &domain.BusinessError{
				Code:    string(models.ErrorCodePRODUCT_INACTIVE),
				Message: fmt.Sprintf("Product %s is not active", item.ProductId),
			}
		}
		if product.Stock < item.Quantity {
			return nil, &domain.BusinessError{
				Code:    string(models.ErrorCodeINSUFFICIENT_STOCK),
				Message: "Insufficient stock",
				Details: map[string]interface{}{
					"product_id": item.ProductId,
					"requested":  item.Quantity,
					"available":  product.Stock,
				},
			}
		}

		if err := s.repo.UpdateProductStock(ctx, tx, item.ProductId, -item.Quantity); err != nil {
			return nil, err
		}

		orderItem := domain.OrderItem{
			ProductID:    item.ProductId,
			Quantity:     item.Quantity,
			PriceAtOrder: product.Price,
		}
		newItems = append(newItems, orderItem)
		totalAmount += product.Price * float64(item.Quantity)
	}

	// 6. Recalculate with promo code
	discountAmount := 0.0
	promoCodeID := existingOrder.PromoCodeID
	if promoCodeID != nil {
		promo, err := s.repo.GetPromoCodeByCode(ctx, tx, "")
		if err == nil && promo != nil {
			// Check if promo still valid
			if totalAmount >= promo.MinOrderAmount && promo.Active &&
				time.Now().After(promo.ValidFrom) && time.Now().Before(promo.ValidUntil) {
				if promo.DiscountType == "PERCENTAGE" {
					discountAmount = totalAmount * promo.DiscountValue / 100
					if discountAmount > totalAmount*0.7 {
						discountAmount = totalAmount * 0.7
					}
				} else {
					discountAmount = promo.DiscountValue
					if discountAmount > totalAmount {
						discountAmount = totalAmount
					}
				}
			} else {
				// Promo no longer valid, remove it
				s.repo.UpdatePromoCodeUsage(ctx, tx, *promoCodeID, -1)
				promoCodeID = nil
			}
		}
	}

	// Update order
	updatedOrder := &domain.Order{
		ID:             orderID,
		UserID:         existingOrder.UserID,
		Status:         existingOrder.Status,
		PromoCodeID:    promoCodeID,
		TotalAmount:    totalAmount - discountAmount,
		DiscountAmount: discountAmount,
		Items:          newItems,
		CreatedAt:      existingOrder.CreatedAt,
	}

	if err := s.repo.UpdateOrder(ctx, tx, updatedOrder); err != nil {
		return nil, err
	}

	// 7. Record operation
	if err := s.repo.RecordOperation(ctx, tx, userID, "UPDATE_ORDER"); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return updatedOrder, nil
}

// CancelOrder cancels an order with full business logic
func (s *Service) CancelOrder(ctx context.Context, orderID, userID, userRole string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get order
	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return &domain.BusinessError{
			Code:    string(models.ErrorCodeORDER_NOT_FOUND),
			Message: "Order not found",
		}
	}

	// 1. Check ownership
	if userRole != "ADMIN" && order.UserID != userID {
		return &domain.BusinessError{
			Code:    string(models.ErrorCodeORDER_OWNERSHIP_VIOLATION),
			Message: "You can only cancel your own orders",
		}
	}

	// 2. Check state
	if order.Status != "CREATED" && order.Status != "PAYMENT_PENDING" {
		return &domain.BusinessError{
			Code:    string(models.ErrorCodeINVALID_STATE_TRANSITION),
			Message: "Order can only be cancelled from CREATED or PAYMENT_PENDING state",
		}
	}

	// 3. Return stock
	for _, item := range order.Items {
		if err := s.repo.UpdateProductStock(ctx, tx, item.ProductID, item.Quantity); err != nil {
			return err
		}
	}

	// 4. Return promo code usage
	if order.PromoCodeID != nil {
		if err := s.repo.UpdatePromoCodeUsage(ctx, tx, *order.PromoCodeID, -1); err != nil {
			return err
		}
	}

	// 5. Set status to CANCELED
	if err := s.repo.UpdateOrderStatus(ctx, tx, orderID, "CANCELED"); err != nil {
		return err
	}

	return tx.Commit()
}

// ============================================================================
// PROMO CODE SERVICE METHODS
// ============================================================================

// CreatePromoCode creates a new promo code
func (s *Service) CreatePromoCode(ctx context.Context, req models.PromoCodeCreateRequest) (*domain.PromoCode, error) {
	promo := &domain.PromoCode{
		Code:           req.Code,
		DiscountType:   string(req.DiscountType),
		DiscountValue:  req.DiscountValue,
		MinOrderAmount: req.MinOrderAmount,
		MaxUses:        req.MaxUses,
		ValidFrom:      req.ValidFrom,
		ValidUntil:     req.ValidUntil,
	}
	err := s.repo.CreatePromoCode(ctx, promo)
	return promo, err
}
