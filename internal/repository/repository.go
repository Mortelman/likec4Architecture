package repository

import (
	"context"
	"database/sql"
	"fmt"
	"marketplace/internal/domain"
)

// Repository handles database operations
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new repository instance
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ============================================================================
// USER OPERATIONS
// ============================================================================

// CreateUser creates a new user
func (r *Repository) CreateUser(ctx context.Context, email, passwordHash, role string) (*domain.User, error) {
	user := &domain.User{Email: email, PasswordHash: passwordHash, Role: role}
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3) 
		 RETURNING id, created_at`,
		email, passwordHash, role).Scan(&user.ID, &user.CreatedAt)
	return user, err
}

// GetUserByEmail retrieves a user by email
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, role, created_at FROM users WHERE email = $1`,
		email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

// ============================================================================
// PRODUCT OPERATIONS
// ============================================================================

// CreateProduct creates a new product
func (r *Repository) CreateProduct(ctx context.Context, p *domain.Product) error {
	return r.db.QueryRowContext(ctx,
		`INSERT INTO products (name, description, price, stock, category, status, seller_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at, updated_at`,
		p.Name, p.Description, p.Price, p.Stock, p.Category, p.Status, p.SellerID,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

// GetProductByID retrieves a product by ID
func (r *Repository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	var p domain.Product
	var desc sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, description, price, stock, category, status, seller_id, created_at, updated_at
		 FROM products WHERE id = $1`,
		id).Scan(&p.ID, &p.Name, &desc, &p.Price, &p.Stock, &p.Category, &p.Status, &p.SellerID, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if desc.Valid {
		p.Description = &desc.String
	}
	return &p, err
}

// GetProducts retrieves products with pagination and filtering
func (r *Repository) GetProducts(ctx context.Context, page, size int32, status, category string) ([]domain.Product, int, error) {
	query := `SELECT id, name, description, price, stock, category, status, seller_id, created_at, updated_at
			  FROM products WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM products WHERE 1=1`
	args := []interface{}{}
	argPos := 1

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		countQuery += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, status)
		argPos++
	}
	if category != "" {
		query += fmt.Sprintf(" AND category = $%d", argPos)
		countQuery += fmt.Sprintf(" AND category = $%d", argPos)
		args = append(args, category)
		argPos++
	}

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, size, page*size)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	products := []domain.Product{}
	for rows.Next() {
		var p domain.Product
		var desc sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &desc, &p.Price, &p.Stock, &p.Category, &p.Status, &p.SellerID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if desc.Valid {
			p.Description = &desc.String
		}
		products = append(products, p)
	}
	return products, total, rows.Err()
}

// UpdateProduct updates a product
func (r *Repository) UpdateProduct(ctx context.Context, id string, p *domain.Product) error {
	return r.db.QueryRowContext(ctx,
		`UPDATE products 
		 SET name = $1, description = $2, price = $3, stock = $4, category = $5, status = $6
		 WHERE id = $7
		 RETURNING updated_at`,
		p.Name, p.Description, p.Price, p.Stock, p.Category, p.Status, id,
	).Scan(&p.UpdatedAt)
}

// SoftDeleteProduct performs soft delete by setting status to ARCHIVED
func (r *Repository) SoftDeleteProduct(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE products SET status = 'ARCHIVED' WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdateProductStock updates product stock (used in transactions)
func (r *Repository) UpdateProductStock(ctx context.Context, tx *sql.Tx, productID string, delta int32) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE products SET stock = stock + $1 WHERE id = $2`,
		delta, productID)
	return err
}

// ============================================================================
// ORDER OPERATIONS
// ============================================================================

// CreateOrder creates a new order with items (within transaction)
func (r *Repository) CreateOrder(ctx context.Context, tx *sql.Tx, order *domain.Order) error {
	err := tx.QueryRowContext(ctx,
		`INSERT INTO orders (user_id, status, promo_code_id, total_amount, discount_amount)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at, updated_at`,
		order.UserID, order.Status, order.PromoCodeID, order.TotalAmount, order.DiscountAmount,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return err
	}

	for i := range order.Items {
		err = tx.QueryRowContext(ctx,
			`INSERT INTO order_items (order_id, product_id, quantity, price_at_order)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id`,
			order.ID, order.Items[i].ProductID, order.Items[i].Quantity, order.Items[i].PriceAtOrder,
		).Scan(&order.Items[i].ID)
		if err != nil {
			return err
		}
		order.Items[i].OrderID = order.ID
	}
	return nil
}

// GetOrderByID retrieves an order by ID
func (r *Repository) GetOrderByID(ctx context.Context, id string) (*domain.Order, error) {
	var order domain.Order
	var promoCodeID sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, status, promo_code_id, total_amount, discount_amount, created_at, updated_at
		 FROM orders WHERE id = $1`,
		id).Scan(&order.ID, &order.UserID, &order.Status, &promoCodeID, &order.TotalAmount, &order.DiscountAmount, &order.CreatedAt, &order.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if promoCodeID.Valid {
		order.PromoCodeID = &promoCodeID.String
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, product_id, quantity, price_at_order FROM order_items WHERE order_id = $1`,
		id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.ID, &item.ProductID, &item.Quantity, &item.PriceAtOrder); err != nil {
			return nil, err
		}
		item.OrderID = order.ID
		order.Items = append(order.Items, item)
	}
	return &order, rows.Err()
}

// UpdateOrder updates an order (within transaction)
func (r *Repository) UpdateOrder(ctx context.Context, tx *sql.Tx, order *domain.Order) error {
	// Delete old items
	_, err := tx.ExecContext(ctx, `DELETE FROM order_items WHERE order_id = $1`, order.ID)
	if err != nil {
		return err
	}

	// Update order
	err = tx.QueryRowContext(ctx,
		`UPDATE orders 
		 SET status = $1, promo_code_id = $2, total_amount = $3, discount_amount = $4
		 WHERE id = $5
		 RETURNING updated_at`,
		order.Status, order.PromoCodeID, order.TotalAmount, order.DiscountAmount, order.ID,
	).Scan(&order.UpdatedAt)
	if err != nil {
		return err
	}

	// Insert new items
	for i := range order.Items {
		err = tx.QueryRowContext(ctx,
			`INSERT INTO order_items (order_id, product_id, quantity, price_at_order)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id`,
			order.ID, order.Items[i].ProductID, order.Items[i].Quantity, order.Items[i].PriceAtOrder,
		).Scan(&order.Items[i].ID)
		if err != nil {
			return err
		}
		order.Items[i].OrderID = order.ID
	}
	return nil
}

// UpdateOrderStatus updates order status (within transaction)
func (r *Repository) UpdateOrderStatus(ctx context.Context, tx *sql.Tx, orderID, status string) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE orders SET status = $1 WHERE id = $2`,
		status, orderID)
	return err
}

// HasActiveOrder checks if user has active orders
func (r *Repository) HasActiveOrder(ctx context.Context, tx *sql.Tx, userID string) (bool, error) {
	var count int
	err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM orders 
		 WHERE user_id = $1 AND status IN ('CREATED', 'PAYMENT_PENDING')`,
		userID).Scan(&count)
	return count > 0, err
}

// ============================================================================
// USER OPERATIONS (Rate Limiting)
// ============================================================================

// GetLastOperation retrieves the last operation of a specific type for a user
func (r *Repository) GetLastOperation(ctx context.Context, tx *sql.Tx, userID, opType string) (*domain.Order, error) {
	var createdAt sql.NullTime
	err := tx.QueryRowContext(ctx,
		`SELECT created_at FROM user_operations 
		 WHERE user_id = $1 AND operation_type = $2 
		 ORDER BY created_at DESC LIMIT 1`,
		userID, opType).Scan(&createdAt)
	if err == sql.ErrNoRows || !createdAt.Valid {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.Order{CreatedAt: createdAt.Time}, nil
}

// RecordOperation records a user operation
func (r *Repository) RecordOperation(ctx context.Context, tx *sql.Tx, userID, opType string) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO user_operations (user_id, operation_type) VALUES ($1, $2)`,
		userID, opType)
	return err
}

// ============================================================================
// PROMO CODE OPERATIONS
// ============================================================================

// GetPromoCodeByCode retrieves a promo code by code (within transaction)
func (r *Repository) GetPromoCodeByCode(ctx context.Context, tx *sql.Tx, code string) (*domain.PromoCode, error) {
	var pc domain.PromoCode
	err := tx.QueryRowContext(ctx,
		`SELECT id, code, discount_type, discount_value, min_order_amount, max_uses, current_uses, 
		        valid_from, valid_until, active, created_at
		 FROM promo_codes WHERE code = $1`,
		code).Scan(&pc.ID, &pc.Code, &pc.DiscountType, &pc.DiscountValue, &pc.MinOrderAmount,
		&pc.MaxUses, &pc.CurrentUses, &pc.ValidFrom, &pc.ValidUntil, &pc.Active, &pc.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &pc, err
}

// UpdatePromoCodeUsage updates promo code usage count (within transaction)
func (r *Repository) UpdatePromoCodeUsage(ctx context.Context, tx *sql.Tx, promoCodeID string, delta int32) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE promo_codes SET current_uses = current_uses + $1 WHERE id = $2`,
		delta, promoCodeID)
	return err
}

// CreatePromoCode creates a new promo code
func (r *Repository) CreatePromoCode(ctx context.Context, pc *domain.PromoCode) error {
	return r.db.QueryRowContext(ctx,
		`INSERT INTO promo_codes (code, discount_type, discount_value, min_order_amount, max_uses, valid_from, valid_until)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, current_uses, active, created_at`,
		pc.Code, pc.DiscountType, pc.DiscountValue, pc.MinOrderAmount, pc.MaxUses, pc.ValidFrom, pc.ValidUntil,
	).Scan(&pc.ID, &pc.CurrentUses, &pc.Active, &pc.CreatedAt)
}
