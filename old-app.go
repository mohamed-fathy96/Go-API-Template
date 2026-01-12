//package main
//
//import (
//"context"
//"database/sql"
//"encoding/csv"
//"encoding/json"
//"errors"
//"fmt"
//"log"
//"math"
//"net/http"
//"os"
//"strconv"
//"strings"
//"sync"
//"time"
//
//	"github.com/gorilla/mux"
//	_ "github.com/lib/pq"
//	"golang.org/x/crypto/bcrypt"
//)
//
//// Constants
//const (
//StatusPending   = "pending"
//StatusAssigned  = "assigned"
//StatusPickedUp  = "picked-up"
//StatusDelivered = "delivered"
//StatusCancelled = "cancelled"
//
//	locationStaleThreshold = 10 * time.Minute
//	defaultDriverSpeed     = 40.0   // km/h
//	earthRadius            = 6371.0 // km
//	statsRefreshInterval   = 10 * time.Second
//	orderAssignInterval    = 30 * time.Second
//	locationCheckInterval  = 5 * time.Minute
//	defaultPageLimit       = 10
//	minPasswordLength      = 4
//
//	adminUsername = "admin"
//	adminPassword = "admin"
//)
//
//// SQL Queries
//const (
//queryDriversOnlineFree = `
//		SELECT d.id, d.location
//		FROM drivers d
//		WHERE d.online = true
//		  AND d.is_blocked = false
//		  AND NOT EXISTS (
//			SELECT 1 FROM orders o
//			WHERE o.assigned_to = d.id
//			  AND o.status NOT IN ($1, $2)
//		)`
//
//	queryOrderStats = `
//		SELECT
//			COUNT(*) FILTER (WHERE status = $1) AS pending,
//			COUNT(*) FILTER (WHERE status = $2) AS assigned,
//			COUNT(*) FILTER (WHERE status = $3) AS picked_up,
//			COUNT(*) FILTER (WHERE status = $4) AS delivered
//		FROM orders`
//
//	queryOldestOrderByStatus = `
//		SELECT EXTRACT(EPOCH FROM (NOW() - %s))/60
//		FROM orders
//		WHERE status = $1
//		ORDER BY %s ASC
//		LIMIT 1`
//
//	queryPendingOrders = `
//		SELECT id, customer_location
//		FROM orders
//		WHERE status = $1 AND assigned_to IS NULL`
//
//	queryUpdateStaleDrivers = `
//		UPDATE drivers
//		SET online = false
//		WHERE online = true
//		  AND (location_updated_at IS NULL OR location_updated_at < NOW() - INTERVAL '10 minutes')`
//
//	orderByCriticalStatus = ` ORDER BY
//		CASE status
//			WHEN 'pending' THEN 1
//			WHEN 'assigned' THEN 2
//			WHEN 'picked-up' THEN 3
//			WHEN 'delivered' THEN 4
//			WHEN 'cancelled' THEN 5
//			ELSE 6
//		END,
//		created_at ASC`
//)
//
//var (
//ErrInvalidLocation      = errors.New("invalid location format")
//ErrInvalidCredentials   = errors.New("invalid credentials")
//ErrDriverNotEligible    = errors.New("driver is not eligible")
//ErrOrderNotCancelable   = errors.New("order not found or not cancelable")
//ErrDriverIsDelivering   = errors.New("driver is currently delivering")
//ErrPasswordTooShort     = errors.New("password too short")
//ErrDriverExists         = errors.New("driver already exists")
//ErrInvalidLatLng        = errors.New("both lat and lng must be provided together")
//ErrOrderNotReassignable = errors.New("cannot reassign delivered or cancelled orders")
//)
//
//// Domain Models
//type Location struct {
//Lat float64 `json:"lat"`
//Lng float64 `json:"lng"`
//}
//
//func (l *Location) String() string {
//return fmt.Sprintf("%f,%f", l.Lat, l.Lng)
//}
//
//func ParseLocation(s string) (*Location, error) {
//parts := strings.Split(s, ",")
//if len(parts) != 2 {
//return nil, ErrInvalidLocation
//}
//
//	lat, err := strconv.ParseFloat(parts[0], 64)
//	if err != nil {
//		return nil, fmt.Errorf("invalid latitude: %w", err)
//	}
//
//	lng, err := strconv.ParseFloat(parts[1], 64)
//	if err != nil {
//		return nil, fmt.Errorf("invalid longitude: %w", err)
//	}
//
//	return &Location{Lat: lat, Lng: lng}, nil
//}
//
//func parseNullableLocation(ns sql.NullString) *Location {
//if ns.Valid {
//if loc, err := ParseLocation(ns.String); err == nil {
//return loc
//}
//}
//return nil
//}
//
//type Driver struct {
//ID        string    `json:"id"`
//Name      string    `json:"name"`
//Phone     string    `json:"phone_number"`
//Location  *Location `json:"location,omitempty"`
//Online    bool      `json:"online"`
//IsBlocked bool      `json:"is_blocked"`
//}
//
//type Order struct {
//ID             string     `json:"id"`
//DriverID       string     `json:"assigned_to,omitempty"`
//DriverName     string     `json:"driver_name,omitempty"`
//Status         string     `json:"status"`
//CustomerLoc    *Location  `json:"customer_location,omitempty"`
//StoreLoc       *Location  `json:"store_location,omitempty"`
//CustomerPhone  string     `json:"customer_phone"`
//CustomerName   string     `json:"customer_name"`
//CustomerBP     string     `json:"customer_addr_bp"`
//CustomerDetail string     `json:"customer_addr_details"`
//CreatedAt      time.Time  `json:"created_at"`
//PickedAt       *time.Time `json:"picked_at,omitempty"`
//DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
//}
//
//type OrderStats struct {
//Pending        int `json:"pending"`
//Assigned       int `json:"assigned"`
//PickedUp       int `json:"picked_up"`
//Delivered      int `json:"delivered"`
//OldestPending  int `json:"oldest_unassigned_minutes"`
//OldestAssigned int `json:"oldest_assigned_not_picked_minutes"`
//OldestPicked   int `json:"oldest_picked_not_delivered_minutes"`
//}
//
//type OrderList struct {
//Orders     []*Order `json:"orders"`
//TotalCount int      `json:"total_count"`
//}
//
//type TrackingInfo struct {
//OrderID     string    `json:"order_id"`
//Status      string    `json:"status"`
//DriverLoc   *Location `json:"driver_location,omitempty"`
//CustomerLoc *Location `json:"customer_location,omitempty"`
//ETA         int       `json:"eta"` // minutes
//}
//
//// Services
//type StatsService struct {
//db    *sql.DB
//stats OrderStats
//mu    sync.RWMutex
//}
//
//func NewStatsService(db *sql.DB) *StatsService {
//return &StatsService{db: db}
//}
//
//func (s *StatsService) Refresh(ctx context.Context) error {
//var stats OrderStats
//
//	err := s.db.QueryRowContext(ctx, queryOrderStats,
//		StatusPending, StatusAssigned, StatusPickedUp, StatusDelivered).
//		Scan(&stats.Pending, &stats.Assigned, &stats.PickedUp, &stats.Delivered)
//	if err != nil {
//		return fmt.Errorf("failed to refresh stats: %w", err)
//	}
//
//	stats.OldestPending = s.getOldestMinutes(ctx, StatusPending, "created_at")
//	stats.OldestAssigned = s.getOldestMinutes(ctx, StatusAssigned, "created_at")
//	stats.OldestPicked = s.getOldestMinutes(ctx, StatusPickedUp, "picked_at")
//
//	s.mu.Lock()
//	s.stats = stats
//	s.mu.Unlock()
//
//	return nil
//}
//
//func (s *StatsService) getOldestMinutes(ctx context.Context, status, field string) int {
//var minutes float64
//query := fmt.Sprintf(queryOldestOrderByStatus, field, field)
//err := s.db.QueryRowContext(ctx, query, status).Scan(&minutes)
//if err != nil {
//return 0
//}
//return int(minutes)
//}
//
//func (s *StatsService) Get() OrderStats {
//s.mu.RLock()
//defer s.mu.RUnlock()
//return s.stats
//}
//
//type AssignmentService struct {
//db *sql.DB
//}
//
//func NewAssignmentService(db *sql.DB) *AssignmentService {
//return &AssignmentService{db: db}
//}
//
//func (s *AssignmentService) AssignOptimalOrders(ctx context.Context) error {
//rows, err := s.db.QueryContext(ctx, queryDriversOnlineFree, StatusDelivered, StatusCancelled)
//if err != nil {
//return fmt.Errorf("failed to query free drivers: %w", err)
//}
//defer rows.Close()
//
//	for rows.Next() {
//		var driverID, locStr string
//		if err := rows.Scan(&driverID, &locStr); err != nil {
//			log.Printf("Failed to scan driver: %v", err)
//			continue
//		}
//		if err := s.AssignBestOrder(ctx, driverID); err != nil {
//			log.Printf("Failed to assign order to driver %s: %v", driverID, err)
//		}
//	}
//
//	return rows.Err()
//}
//
//func (s *AssignmentService) AssignBestOrder(ctx context.Context, driverID string) error {
//var locStr string
//err := s.db.QueryRowContext(ctx, `
//		SELECT location FROM drivers
//		WHERE id = $1 AND online = true AND is_blocked = false`,
//driverID).Scan(&locStr)
//if err != nil {
//return err
//}
//
//	driverLoc, err := ParseLocation(locStr)
//	if err != nil {
//		return err
//	}
//
//	rows, err := s.db.QueryContext(ctx, queryPendingOrders, StatusPending)
//	if err != nil {
//		return err
//	}
//	defer rows.Close()
//
//	var (
//		bestOrderID string
//		minDistance = math.MaxFloat64
//	)
//
//	for rows.Next() {
//		var orderID, custLocStr string
//		if err := rows.Scan(&orderID, &custLocStr); err != nil {
//			continue
//		}
//
//		custLoc, err := ParseLocation(custLocStr)
//		if err != nil {
//			continue
//		}
//
//		dist := haversine(driverLoc, custLoc)
//		if dist < minDistance {
//			minDistance = dist
//			bestOrderID = orderID
//		}
//	}
//
//	if bestOrderID != "" {
//		_, err = s.db.ExecContext(ctx, `
//			UPDATE orders
//			SET assigned_to = $1, status = $2
//			WHERE id = $3 AND assigned_to IS NULL`,
//			driverID, StatusAssigned, bestOrderID)
//		return err
//	}
//
//	return nil
//}
//
//func (s *AssignmentService) FindOptimalDriver(ctx context.Context, custLat, custLng float64) (string, error) {
//rows, err := s.db.QueryContext(ctx, queryDriversOnlineFree, StatusDelivered, StatusCancelled)
//if err != nil {
//return "", err
//}
//defer rows.Close()
//
//	var (
//		bestDriverID string
//		minDistance  = math.MaxFloat64
//		custLoc      = &Location{Lat: custLat, Lng: custLng}
//	)
//
//	for rows.Next() {
//		var driverID, locStr string
//		if err := rows.Scan(&driverID, &locStr); err != nil {
//			continue
//		}
//
//		driverLoc, err := ParseLocation(locStr)
//		if err != nil {
//			continue
//		}
//
//		dist := haversine(driverLoc, custLoc)
//		if dist < minDistance {
//			minDistance = dist
//			bestDriverID = driverID
//		}
//	}
//
//	return bestDriverID, nil
//}
//
//type DriverRepository struct {
//db *sql.DB
//}
//
//func NewDriverRepository(db *sql.DB) *DriverRepository {
//return &DriverRepository{db: db}
//}
//
//func (r *DriverRepository) Create(ctx context.Context, id, name, phone, password string, location *Location) error {
//if len(password) < minPasswordLength {
//return ErrPasswordTooShort
//}
//
//	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
//	if err != nil {
//		return fmt.Errorf("failed to hash password: %w", err)
//	}
//
//	var existingID string
//	err = r.db.QueryRowContext(ctx, "SELECT id FROM drivers WHERE name = $1", name).Scan(&existingID)
//	if err != sql.ErrNoRows {
//		return ErrDriverExists
//	}
//
//	var locationStr sql.NullString
//	if location != nil {
//		locationStr = sql.NullString{String: location.String(), Valid: true}
//	}
//
//	_, err = r.db.ExecContext(ctx, `
//		INSERT INTO drivers (id, name, location, online, password_hash, phone_number)
//		VALUES ($1, $2, $3, $4, $5, $6)`,
//		id, name, locationStr, false, string(hashedPassword), phone)
//
//	return err
//}
//
//func (r *DriverRepository) Authenticate(ctx context.Context, name, password string) (*Driver, error) {
//var driver Driver
//var storedHash string
//
//	err := r.db.QueryRowContext(ctx, `
//		SELECT id, name, password_hash
//		FROM drivers
//		WHERE is_blocked = false AND name = $1`, name).
//		Scan(&driver.ID, &driver.Name, &storedHash)
//
//	if err == sql.ErrNoRows {
//		return nil, ErrInvalidCredentials
//	}
//	if err != nil {
//		return nil, err
//	}
//
//	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err != nil {
//		return nil, ErrInvalidCredentials
//	}
//
//	return &driver, nil
//}
//
//func (r *DriverRepository) SetOnline(ctx context.Context, driverID string, online bool) error {
//_, err := r.db.ExecContext(ctx, "UPDATE drivers SET online = $1 WHERE id = $2", online, driverID)
//return err
//}
//
//func (r *DriverRepository) UpdateLocation(ctx context.Context, driverID string, loc *Location, battery, network string) error {
//_, err := r.db.ExecContext(ctx, `
//		UPDATE drivers
//		SET location = $1, battery_level = $2, network_info = $3, location_updated_at = NOW()
//		WHERE id = $4`,
//loc.String(), battery, network, driverID)
//return err
//}
//
//func (r *DriverRepository) Block(ctx context.Context, driverID string) error {
//// Check for active deliveries
//var activeCount int
//err := r.db.QueryRowContext(ctx, `
//		SELECT COUNT(*)
//		FROM orders
//		WHERE assigned_to = $1 AND status = $2`, driverID, StatusPickedUp).Scan(&activeCount)
//if err != nil {
//return err
//}
//if activeCount > 0 {
//return ErrDriverIsDelivering
//}
//
//	// Unassign pending orders
//	_, err = r.db.ExecContext(ctx, `
//		UPDATE orders
//		SET assigned_to = NULL, status = $1
//		WHERE assigned_to = $2 AND status = $3`, StatusPending, driverID, StatusAssigned)
//	if err != nil {
//		return err
//	}
//
//	_, err = r.db.ExecContext(ctx, "UPDATE drivers SET is_blocked = true WHERE id = $1", driverID)
//	return err
//}
//
//func (r *DriverRepository) Unblock(ctx context.Context, driverID string) error {
//_, err := r.db.ExecContext(ctx, "UPDATE drivers SET is_blocked = false WHERE id = $1", driverID)
//return err
//}
//
//func (r *DriverRepository) ResetPassword(ctx context.Context, driverID, newPassword string) error {
//if len(newPassword) < minPasswordLength {
//return ErrPasswordTooShort
//}
//
//	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
//	if err != nil {
//		return fmt.Errorf("failed to hash password: %w", err)
//	}
//
//	result, err := r.db.ExecContext(ctx,
//		"UPDATE drivers SET password_hash = $1 WHERE id = $2",
//		string(hashedPassword), driverID)
//	if err != nil {
//		return err
//	}
//
//	rowsAffected, _ := result.RowsAffected()
//	if rowsAffected == 0 {
//		return sql.ErrNoRows
//	}
//
//	return nil
//}
//
//func (r *DriverRepository) GetByID(ctx context.Context, driverID string) (*Driver, error) {
//var d Driver
//var locStr sql.NullString
//
//	err := r.db.QueryRowContext(ctx, `
//		SELECT id, name, location, online, is_blocked
//		FROM drivers
//		WHERE id = $1 AND is_blocked = false`, driverID).
//		Scan(&d.ID, &d.Name, &locStr, &d.Online, &d.IsBlocked)
//
//	if err != nil {
//		return nil, err
//	}
//
//	d.Location = parseNullableLocation(locStr)
//	return &d, nil
//}
//
//func (r *DriverRepository) List(ctx context.Context) ([]*Driver, error) {
//rows, err := r.db.QueryContext(ctx, `
//		SELECT d.id, d.name, d.location, d.online, d.is_blocked
//		FROM drivers d
//		ORDER BY d.online DESC, d.id`)
//if err != nil {
//return nil, err
//}
//defer rows.Close()
//
//	var drivers []*Driver
//	for rows.Next() {
//		var d Driver
//		var locStr sql.NullString
//
//		if err := rows.Scan(&d.ID, &d.Name, &locStr, &d.Online, &d.IsBlocked); err != nil {
//			continue
//		}
//
//		d.Location = parseNullableLocation(locStr)
//		drivers = append(drivers, &d)
//	}
//
//	return drivers, rows.Err()
//}
//
//func (r *DriverRepository) ListEligible(ctx context.Context) ([]*Driver, error) {
//rows, err := r.db.QueryContext(ctx, `
//		SELECT d.id, d.name, d.location, d.online, d.is_blocked
//		FROM drivers d
//		WHERE d.online = true
//		  AND d.is_blocked = false
//		  AND NOT EXISTS (
//		    SELECT 1 FROM orders o
//		    WHERE o.assigned_to = d.id AND o.status NOT IN ($1, $2)
//		)`, StatusDelivered, StatusCancelled)
//if err != nil {
//return nil, err
//}
//defer rows.Close()
//
//	var drivers []*Driver
//	for rows.Next() {
//		var d Driver
//		var locStr sql.NullString
//
//		if err := rows.Scan(&d.ID, &d.Name, &locStr, &d.Online, &d.IsBlocked); err != nil {
//			continue
//		}
//
//		d.Location = parseNullableLocation(locStr)
//		drivers = append(drivers, &d)
//	}
//
//	return drivers, rows.Err()
//}
//
//func (r *DriverRepository) MarkStaleDriversOffline(ctx context.Context) error {
//_, err := r.db.ExecContext(ctx, queryUpdateStaleDrivers)
//return err
//}
//
//type OrderRepository struct {
//db *sql.DB
//}
//
//func NewOrderRepository(db *sql.DB) *OrderRepository {
//return &OrderRepository{db: db}
//}
//
//func (r *OrderRepository) Create(ctx context.Context, order *Order, driverID string) error {
//status := StatusPending
//if driverID != "" {
//status = StatusAssigned
//}
//
//	var err error
//	if driverID != "" {
//		_, err = r.db.ExecContext(ctx, `
//			INSERT INTO orders (
//				id, assigned_to, status, customer_location, store_location,
//				customer_phone, customer_name, customer_addr_bp, customer_addr_details
//			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
//			order.ID, driverID, status, order.CustomerLoc.String(), order.StoreLoc.String(),
//			order.CustomerPhone, order.CustomerName, order.CustomerBP, order.CustomerDetail)
//	} else {
//		_, err = r.db.ExecContext(ctx, `
//			INSERT INTO orders (
//				id, status, customer_location, store_location,
//				customer_phone, customer_name, customer_addr_bp, customer_addr_details
//			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
//			order.ID, status, order.CustomerLoc.String(), order.StoreLoc.String(),
//			order.CustomerPhone, order.CustomerName, order.CustomerBP, order.CustomerDetail)
//	}
//
//	return err
//}
//
//func (r *OrderRepository) List(ctx context.Context, searchByID string, offset, limit int) ([]*Order, int, error) {
//var rows *sql.Rows
//var err error
//var total int
//
//	if searchByID != "" {
//		rows, err = r.db.QueryContext(ctx, `
//			SELECT o.id, o.assigned_to, d.name, o.status, o.customer_location,
//			       o.customer_name, o.customer_phone, o.store_location,
//			       o.created_at, o.picked_at, o.delivered_at
//			FROM orders o
//			LEFT JOIN drivers d ON o.assigned_to = d.id
//			WHERE o.id = $1`, searchByID)
//		total = 1
//	} else {
//		rows, err = r.db.QueryContext(ctx, `
//			SELECT o.id, o.assigned_to, d.name, o.status, o.customer_location,
//				o.customer_name, o.customer_phone, o.store_location,
//				o.created_at, o.picked_at, o.delivered_at
//			FROM orders o
//			LEFT JOIN drivers d ON o.assigned_to = d.id
//			`+orderByCriticalStatus+`
//			LIMIT $1 OFFSET $2`, limit, offset)
//
//		if err == nil {
//			err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders").Scan(&total)
//		}
//	}
//
//	if err != nil {
//		return nil, 0, err
//	}
//	defer rows.Close()
//
//	var orders []*Order
//	for rows.Next() {
//		order, err := r.scanOrder(rows)
//		if err != nil {
//			log.Printf("Failed to scan order: %v", err)
//			continue
//		}
//		orders = append(orders, order)
//	}
//
//	return orders, total, rows.Err()
//}
//
//func (r *OrderRepository) scanOrder(rows *sql.Rows) (*Order, error) {
//var o Order
//var driverID, driverName, custLocStr, storeLocStr sql.NullString
//var pickedAt, deliveredAt sql.NullTime
//
//	err := rows.Scan(
//		&o.ID, &driverID, &driverName, &o.Status, &custLocStr,
//		&o.CustomerName, &o.CustomerPhone, &storeLocStr,
//		&o.CreatedAt, &pickedAt, &deliveredAt,
//	)
//	if err != nil {
//		return nil, err
//	}
//
//	if driverID.Valid {
//		o.DriverID = driverID.String
//	}
//	if driverName.Valid {
//		o.DriverName = driverName.String
//	}
//	if custLocStr.Valid {
//		o.CustomerLoc, _ = ParseLocation(custLocStr.String)
//	}
//	if storeLocStr.Valid {
//		o.StoreLoc, _ = ParseLocation(storeLocStr.String)
//	}
//	if pickedAt.Valid {
//		o.PickedAt = &pickedAt.Time
//	}
//	if deliveredAt.Valid {
//		o.DeliveredAt = &deliveredAt.Time
//	}
//
//	return &o, nil
//}
//
//func (r *OrderRepository) GetByDriver(ctx context.Context, driverID string) ([]*Order, error) {
//rows, err := r.db.QueryContext(ctx, `
//		SELECT id, status, customer_location, customer_addr_bp,
//		       customer_addr_details, store_location, picked_at, delivered_at
//		FROM orders
//		WHERE assigned_to = $1`+orderByCriticalStatus, driverID)
//if err != nil {
//return nil, err
//}
//defer rows.Close()
//
//	var orders []*Order
//	for rows.Next() {
//		var o Order
//		var custLoc, storeLoc string
//		var pickedAt, deliveredAt sql.NullTime
//		var bp, details sql.NullString
//
//		if err := rows.Scan(&o.ID, &o.Status, &custLoc, &bp, &details, &storeLoc, &pickedAt, &deliveredAt); err != nil {
//			continue
//		}
//
//		o.DriverID = driverID
//		o.CustomerLoc, _ = ParseLocation(custLoc)
//		o.StoreLoc, _ = ParseLocation(storeLoc)
//		if bp.Valid {
//			o.CustomerBP = bp.String
//		}
//		if details.Valid {
//			o.CustomerDetail = details.String
//		}
//		if pickedAt.Valid {
//			o.PickedAt = &pickedAt.Time
//		}
//		if deliveredAt.Valid {
//			o.DeliveredAt = &deliveredAt.Time
//		}
//
//		orders = append(orders, &o)
//	}
//
//	return orders, rows.Err()
//}
//
//func (r *OrderRepository) UpdateStatus(ctx context.Context, orderID, driverID, status string) error {
//var query string
//var args []interface{}
//
//	switch status {
//	case StatusPickedUp:
//		query = "UPDATE orders SET status = $1, picked_at = $2 WHERE id = $3 AND assigned_to = $4"
//		args = []interface{}{status, time.Now(), orderID, driverID}
//	case StatusDelivered:
//		query = "UPDATE orders SET status = $1, delivered_at = $2 WHERE id = $3 AND assigned_to = $4"
//		args = []interface{}{status, time.Now(), orderID, driverID}
//	default:
//		query = "UPDATE orders SET status = $1 WHERE id = $2 AND assigned_to = $3"
//		args = []interface{}{status, orderID, driverID}
//	}
//
//	result, err := r.db.ExecContext(ctx, query, args...)
//	if err != nil {
//		return err
//	}
//
//	rowsAffected, _ := result.RowsAffected()
//	if rowsAffected == 0 {
//		return sql.ErrNoRows
//	}
//
//	return nil
//}
//
//func (r *OrderRepository) Cancel(ctx context.Context, orderID string) error {
//result, err := r.db.ExecContext(ctx, `
//		UPDATE orders
//		SET status = $1, assigned_to = NULL
//		WHERE id = $2 AND status IN ($3, $4)`,
//StatusCancelled, orderID, StatusPending, StatusAssigned)
//if err != nil {
//return err
//}
//
//	rowsAffected, _ := result.RowsAffected()
//	if rowsAffected == 0 {
//		return ErrOrderNotCancelable
//	}
//
//	return nil
//}
//
//func (r *OrderRepository) Reassign(ctx context.Context, orderID, driverID string) error {
//// Validate order status
//var status string
//err := r.db.QueryRowContext(ctx, "SELECT status FROM orders WHERE id = $1", orderID).Scan(&status)
//if err == sql.ErrNoRows {
//return sql.ErrNoRows
//}
//if err != nil {
//return err
//}
//
//	if status == StatusDelivered || status == StatusCancelled {
//		return ErrOrderNotReassignable
//	}
//
//	// Validate driver eligibility
//	var eligible bool
//	err = r.db.QueryRowContext(ctx, `
//		SELECT EXISTS (
//			SELECT 1 FROM drivers d
//			WHERE d.id = $1 AND d.online = true AND d.is_blocked = false
//			AND NOT EXISTS (
//				SELECT 1 FROM orders o
//				WHERE o.assigned_to = d.id AND o.status NOT IN ($2, $3)
//			)
//		)`, driverID, StatusDelivered, StatusCancelled).Scan(&eligible)
//
//	if err != nil || !eligible {
//		return ErrDriverNotEligible
//	}
//
//	_, err = r.db.ExecContext(ctx,
//		"UPDATE orders SET assigned_to = $1, status = $2 WHERE id = $3",
//		driverID, StatusAssigned, orderID)
//
//	return err
//}
//
//func (r *OrderRepository) GetTracking(ctx context.Context, orderID string) (*TrackingInfo, error) {
//var status, driverID, custLocStr string
//err := r.db.QueryRowContext(ctx, `
//		SELECT status, assigned_to, customer_location
//		FROM orders
//		WHERE id = $1`, orderID).
//Scan(&status, &driverID, &custLocStr)
//
//	if err != nil {
//		return nil, err
//	}
//
//	customerLoc, _ := ParseLocation(custLocStr)
//
//	if driverID == "" {
//		return &TrackingInfo{
//			OrderID:     orderID,
//			Status:      status,
//			CustomerLoc: customerLoc,
//			ETA:         0,
//		}, nil
//	}
//
//	var driverLocStr string
//	err = r.db.QueryRowContext(ctx, `
//		SELECT location
//		FROM drivers
//		WHERE is_blocked = false AND id = $1`, driverID).
//		Scan(&driverLocStr)
//
//	if err != nil {
//		return nil, err
//	}
//
//	driverLoc, _ := ParseLocation(driverLocStr)
//	eta := 0
//	if driverLoc != nil && customerLoc != nil {
//		eta = estimateETA(driverLoc, customerLoc)
//	}
//
//	return &TrackingInfo{
//		OrderID:     orderID,
//		Status:      status,
//		DriverLoc:   driverLoc,
//		CustomerLoc: customerLoc,
//		ETA:         eta,
//	}, nil
//}
//
//func (r *OrderRepository) ExportCSV(ctx context.Context, w http.ResponseWriter) error {
//rows, err := r.db.QueryContext(ctx, `
//		SELECT o.id, o.status, o.customer_name, o.customer_phone, o.store_location, o.customer_location,
//		       o.created_at, d.name AS driver_name
//		FROM orders o
//		LEFT JOIN drivers d ON o.assigned_to = d.id
//		ORDER BY o.created_at DESC`)
//if err != nil {
//return err
//}
//defer rows.Close()
//
//	w.Header().Set("Content-Disposition", "attachment; filename=orders_export.csv")
//	w.Header().Set("Content-Type", "text/csv")
//
//	writer := csv.NewWriter(w)
//	defer writer.Flush()
//
//	headers := []string{
//		"Order ID", "Status", "Customer Name", "Customer Phone",
//		"Store Location", "Customer Location", "Created At", "Driver Name",
//	}
//	writer.Write(headers)
//
//	for rows.Next() {
//		var id, status, custName, custPhone, storeLoc, custLoc, driverName sql.NullString
//		var createdAt time.Time
//
//		if err := rows.Scan(&id, &status, &custName, &custPhone, &storeLoc, &custLoc, &createdAt, &driverName); err != nil {
//			continue
//		}
//
//		writer.Write([]string{
//			id.String, status.String, custName.String, custPhone.String,
//			storeLoc.String, custLoc.String, createdAt.Format("2006-01-02 15:04:05"),
//			driverName.String,
//		})
//	}
//
//	return rows.Err()
//}
//
//// Utility functions
//func haversine(a, b *Location) float64 {
//a1 := a.Lat * math.Pi / 180
//a2 := b.Lat * math.Pi / 180
//b1 := (b.Lat - a.Lat) * math.Pi / 180
//b2 := (b.Lng - a.Lng) * math.Pi / 180
//
//	com := math.Sin(b1/2)*math.Sin(b1/2) +
//		math.Cos(a1)*math.Cos(a2)*math.Sin(b2/2)*math.Sin(b2/2)
//	c := 2 * math.Atan2(math.Sqrt(com), math.Sqrt(1-com))
//
//	return earthRadius * c
//}
//
//func estimateETA(from, to *Location) int {
//distance := haversine(from, to)
//return int((distance / defaultDriverSpeed) * 60)
//}
//
//// HTTP Handlers
//type App struct {
//DB                *sql.DB
//Router            *mux.Router
//statsService      *StatsService
//assignmentService *AssignmentService
//driverRepo        *DriverRepository
//orderRepo         *OrderRepository
//}
//
//func NewApp() *App {
//return &App{}
//}
//
//func (a *App) InitDB() error {
//var err error
//dsn := os.Getenv("DATABASE_URL")
//a.DB, err = sql.Open("postgres", dsn)
//if err != nil {
//return fmt.Errorf("failed to open database: %w", err)
//}
//
//	a.DB.SetMaxOpenConns(25)
//	a.DB.SetMaxIdleConns(25)
//	a.DB.SetConnMaxLifetime(5 * time.Minute)
//
//	if err = a.DB.Ping(); err != nil {
//		return fmt.Errorf("failed to ping database: %w", err)
//	}
//
//	// Initialize services and repositories
//	a.statsService = NewStatsService(a.DB)
//	a.assignmentService = NewAssignmentService(a.DB)
//	a.driverRepo = NewDriverRepository(a.DB)
//	a.orderRepo = NewOrderRepository(a.DB)
//
//	return nil
//}
//
//func (a *App) InitRouter() {
//a.Router = mux.NewRouter()
//a.Router.Use(recoveryMiddleware)
//
//	a.Router.HandleFunc("/healthz", a.handleHealthCheck)
//
//	// Admin API
//	admin := a.Router.PathPrefix("/admin").Subrouter()
//	admin.HandleFunc("/create_order", a.handleCreateOrder).Methods("POST")
//	admin.HandleFunc("/orders", a.handleListOrders).Methods("GET")
//	admin.HandleFunc("/drivers", a.handleListDrivers).Methods("GET")
//	admin.HandleFunc("/create_driver", a.handleCreateDriver).Methods("POST")
//	admin.HandleFunc("/login", a.handleAdminLogin).Methods("GET")
//	admin.HandleFunc("/stats", a.handleGetStats).Methods("GET")
//	admin.HandleFunc("/driver/{id}/block", a.handleBlockDriver).Methods("POST")
//	admin.HandleFunc("/driver/{id}/unblock", a.handleUnblockDriver).Methods("POST")
//	admin.HandleFunc("/driver/{id}/reset_password", a.handleResetDriverPassword).Methods("POST")
//	admin.HandleFunc("/order/{id}/cancel", a.handleCancelOrder).Methods("POST")
//	admin.HandleFunc("/order/{id}/assign", a.handleReassignOrder).Methods("POST")
//	admin.HandleFunc("/eligible-drivers", a.handleListEligibleDrivers).Methods("GET")
//	admin.HandleFunc("/orders/export", a.handleExportOrdersCSV).Methods("GET")
//
//	// Driver API
//	driver := a.Router.PathPrefix("/driver").Subrouter()
//	driver.HandleFunc("/login", a.handleDriverLogin).Methods("POST")
//	driver.HandleFunc("/{id}/location", a.handleUpdateLocation).Methods("POST")
//	driver.HandleFunc("/{id}/orders", a.handleDriverOrders).Methods("GET")
//	driver.HandleFunc("/{id}/status", a.handleUpdateStatus).Methods("POST")
//	driver.HandleFunc("/{id}/logout", a.handleDriverLogout).Methods("POST")
//	driver.HandleFunc("/{id}", a.handleGetDriver).Methods("GET")
//
//	// Public API
//	public := a.Router.PathPrefix("/public").Subrouter()
//	public.HandleFunc("/driver/{id}/location", a.handleGetDriverLocation).Methods("GET")
//	public.HandleFunc("/order/{id}/tracking", a.handleOrderTracking).Methods("GET")
//
//	// Frontend routes
//	a.Router.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
//		http.ServeFile(w, r, "./static/admin/index.html")
//	})
//	a.Router.HandleFunc("/driver", func(w http.ResponseWriter, r *http.Request) {
//		http.ServeFile(w, r, "./static/driver/index.html")
//	})
//	a.Router.PathPrefix("/admin/").Handler(http.StripPrefix("/admin/", http.FileServer(http.Dir("./static/admin"))))
//	a.Router.PathPrefix("/driver/").Handler(http.StripPrefix("/driver/", http.FileServer(http.Dir("./static/driver"))))
//}
//
//func (a *App) StartBackgroundJobs() {
//// Initial stats refresh
//ctx := context.Background()
//a.statsService.Refresh(ctx)
//
//	// Start periodic jobs
//	go a.runStatsJob()
//	go a.runAssignmentJob()
//	go a.runLocationStalenessJob()
//}
//
//func (a *App) runStatsJob() {
//defer recoverAndLog("Stats Job")
//ticker := time.NewTicker(statsRefreshInterval)
//defer ticker.Stop()
//
//	for range ticker.C {
//		ctx := context.Background()
//		if err := a.statsService.Refresh(ctx); err != nil {
//			log.Printf("[Stats Job] Failed: %v", err)
//		}
//	}
//}
//
//func (a *App) runAssignmentJob() {
//defer recoverAndLog("Assignment Job")
//ticker := time.NewTicker(orderAssignInterval)
//defer ticker.Stop()
//
//	for range ticker.C {
//		ctx := context.Background()
//		if err := a.assignmentService.AssignOptimalOrders(ctx); err != nil {
//			log.Printf("[Assignment Job] Failed: %v", err)
//		}
//	}
//}
//
//func (a *App) runLocationStalenessJob() {
//defer recoverAndLog("Location Staleness Job")
//ticker := time.NewTicker(locationCheckInterval)
//defer ticker.Stop()
//
//	for range ticker.C {
//		ctx := context.Background()
//		if err := a.driverRepo.MarkStaleDriversOffline(ctx); err != nil {
//			log.Printf("[Location Staleness Job] Failed: %v", err)
//		} else {
//			log.Println("[Location Staleness Job] Inactive drivers set offline")
//		}
//	}
//}
//
//func recoverAndLog(jobName string) {
//if r := recover(); r != nil {
//log.Printf("[Panic in %s] %v", jobName, r)
//}
//}
//
//func (a *App) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
//if err := a.DB.Ping(); err != nil {
//http.Error(w, "DB not ready", http.StatusServiceUnavailable)
//return
//}
//w.WriteHeader(http.StatusOK)
//}
//
//func (a *App) handleGetStats(w http.ResponseWriter, r *http.Request) {
//stats := a.statsService.Get()
//respondJSON(w, http.StatusOK, stats)
//}
//
//func (a *App) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
//var req struct {
//OrderID    string  `json:"order_id"`
//CustLat    float64 `json:"customer_lat"`
//CustLng    float64 `json:"customer_lng"`
//StoreLat   float64 `json:"store_lat"`
//StoreLng   float64 `json:"store_lng"`
//CustName   string  `json:"cust_name"`
//CustPhone  string  `json:"cust_phone"`
//CustBP     string  `json:"cust_address_bp"`
//CustDetail string  `json:"cust_address_details"`
//}
//
//	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
//		respondError(w, http.StatusBadRequest, "Invalid request")
//		return
//	}
//
//	ctx := r.Context()
//	driverID, _ := a.assignmentService.FindOptimalDriver(ctx, req.CustLat, req.CustLng)
//
//	order := &Order{
//		ID:             req.OrderID,
//		CustomerLoc:    &Location{Lat: req.CustLat, Lng: req.CustLng},
//		StoreLoc:       &Location{Lat: req.StoreLat, Lng: req.StoreLng},
//		CustomerPhone:  req.CustPhone,
//		CustomerName:   req.CustName,
//		CustomerBP:     req.CustBP,
//		CustomerDetail: req.CustDetail,
//	}
//
//	if err := a.orderRepo.Create(ctx, order, driverID); err != nil {
//		respondError(w, http.StatusInternalServerError, "Failed to create order")
//		return
//	}
//
//	order.DriverID = driverID
//	if driverID != "" {
//		order.Status = StatusAssigned
//	} else {
//		order.Status = StatusPending
//	}
//
//	respondJSON(w, http.StatusCreated, order)
//}
//
//func (a *App) handleListOrders(w http.ResponseWriter, r *http.Request) {
//searchByID := r.URL.Query().Get("searchByID")
//offset, limit := getPagination(r)
//
//	ctx := r.Context()
//	orders, total, err := a.orderRepo.List(ctx, searchByID, offset, limit)
//	if err != nil {
//		respondError(w, http.StatusInternalServerError, "Failed to fetch orders")
//		return
//	}
//
//	respondJSON(w, http.StatusOK, OrderList{
//		Orders:     orders,
//		TotalCount: total,
//	})
//}
//
//func (a *App) handleListDrivers(w http.ResponseWriter, r *http.Request) {
//ctx := r.Context()
//drivers, err := a.driverRepo.List(ctx)
//if err != nil {
//respondError(w, http.StatusInternalServerError, "Failed to fetch drivers")
//return
//}
//
//	respondJSON(w, http.StatusOK, drivers)
//}
//
//func (a *App) handleListEligibleDrivers(w http.ResponseWriter, r *http.Request) {
//ctx := r.Context()
//drivers, err := a.driverRepo.ListEligible(ctx)
//if err != nil {
//respondError(w, http.StatusInternalServerError, "Failed to fetch eligible drivers")
//return
//}
//
//	respondJSON(w, http.StatusOK, drivers)
//}
//
//func (a *App) handleCreateDriver(w http.ResponseWriter, r *http.Request) {
//var req struct {
//ID       string   `json:"id"`
//Name     string   `json:"name"`
//Password string   `json:"password"`
//Lat      *float64 `json:"lat"`
//Lng      *float64 `json:"lng"`
//Phone    string   `json:"phone_number"`
//}
//
//	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
//		respondError(w, http.StatusBadRequest, "Invalid request")
//		return
//	}
//
//	// Validation: if one of lat/lng is set, the other must be as well
//	if (req.Lat != nil && req.Lng == nil) || (req.Lat == nil && req.Lng != nil) {
//		respondError(w, http.StatusBadRequest, ErrInvalidLatLng.Error())
//		return
//	}
//
//	var location *Location
//	if req.Lat != nil && req.Lng != nil {
//		location = &Location{Lat: *req.Lat, Lng: *req.Lng}
//	}
//
//	ctx := r.Context()
//	err := a.driverRepo.Create(ctx, req.ID, req.Name, req.Phone, req.Password, location)
//	if err != nil {
//		switch err {
//		case ErrPasswordTooShort:
//			respondError(w, http.StatusBadRequest, err.Error())
//		case ErrDriverExists:
//			respondError(w, http.StatusConflict, err.Error())
//		default:
//			respondError(w, http.StatusInternalServerError, err.Error())
//		}
//		return
//	}
//
//	a.assignmentService.AssignBestOrder(ctx, req.ID)
//	respondJSON(w, http.StatusCreated, map[string]string{"status": "created", "id": req.ID})
//}
//
//func (a *App) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
//orderID := mux.Vars(r)["id"]
//ctx := r.Context()
//
//	err := a.orderRepo.Cancel(ctx, orderID)
//	if err != nil {
//		if err == ErrOrderNotCancelable {
//			respondError(w, http.StatusBadRequest, err.Error())
//		} else {
//			respondError(w, http.StatusInternalServerError, "Failed to cancel order")
//		}
//		return
//	}
//
//	respondJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
//}
//
//func (a *App) handleReassignOrder(w http.ResponseWriter, r *http.Request) {
//orderID := mux.Vars(r)["id"]
//
//	var req struct {
//		DriverID string `json:"driver_id"`
//	}
//	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
//		respondError(w, http.StatusBadRequest, "Invalid request")
//		return
//	}
//
//	ctx := r.Context()
//	err := a.orderRepo.Reassign(ctx, orderID, req.DriverID)
//	if err != nil {
//		switch err {
//		case sql.ErrNoRows:
//			respondError(w, http.StatusNotFound, "Order not found")
//		case ErrOrderNotReassignable:
//			respondError(w, http.StatusBadRequest, err.Error())
//		case ErrDriverNotEligible:
//			respondError(w, http.StatusBadRequest, err.Error())
//		default:
//			respondError(w, http.StatusInternalServerError, "Failed to reassign order")
//		}
//		return
//	}
//
//	respondJSON(w, http.StatusOK, map[string]string{
//		"status":    "reassigned",
//		"order_id":  orderID,
//		"driver_id": req.DriverID,
//	})
//}
//
//func (a *App) handleExportOrdersCSV(w http.ResponseWriter, r *http.Request) {
//ctx := r.Context()
//if err := a.orderRepo.ExportCSV(ctx, w); err != nil {
//http.Error(w, "Failed to export orders", http.StatusInternalServerError)
//}
//}
//
//func (a *App) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
//username := r.URL.Query().Get("username")
//password := r.URL.Query().Get("password")
//
//	if username == adminUsername && password == adminPassword {
//		respondJSON(w, http.StatusOK, map[string]string{"token": "admin-token"})
//		return
//	}
//
//	respondError(w, http.StatusUnauthorized, "unauthorized")
//}
//
//func (a *App) handleBlockDriver(w http.ResponseWriter, r *http.Request) {
//driverID := mux.Vars(r)["id"]
//ctx := r.Context()
//
//	err := a.driverRepo.Block(ctx, driverID)
//	if err != nil {
//		if err == ErrDriverIsDelivering {
//			respondError(w, http.StatusBadRequest, err.Error())
//		} else {
//			respondError(w, http.StatusInternalServerError, "Failed to block driver")
//		}
//		return
//	}
//
//	respondJSON(w, http.StatusOK, map[string]string{"status": "blocked"})
//}
//
//func (a *App) handleUnblockDriver(w http.ResponseWriter, r *http.Request) {
//driverID := mux.Vars(r)["id"]
//ctx := r.Context()
//
//	err := a.driverRepo.Unblock(ctx, driverID)
//	if err != nil {
//		respondError(w, http.StatusInternalServerError, "Failed to unblock driver")
//		return
//	}
//
//	respondJSON(w, http.StatusOK, map[string]string{"status": "unblocked"})
//}
//
//func (a *App) handleResetDriverPassword(w http.ResponseWriter, r *http.Request) {
//driverID := mux.Vars(r)["id"]
//
//	var req struct {
//		NewPassword string `json:"new_password"`
//	}
//
//	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
//		respondError(w, http.StatusBadRequest, "Invalid request")
//		return
//	}
//
//	ctx := r.Context()
//	err := a.driverRepo.ResetPassword(ctx, driverID, req.NewPassword)
//	if err != nil {
//		switch err {
//		case ErrPasswordTooShort:
//			respondError(w, http.StatusBadRequest, err.Error())
//		case sql.ErrNoRows:
//			respondError(w, http.StatusNotFound, "Driver not found")
//		default:
//			respondError(w, http.StatusInternalServerError, "Failed to update password")
//		}
//		return
//	}
//
//	respondJSON(w, http.StatusOK, map[string]string{"status": "password updated"})
//}
//
//func (a *App) handleDriverLogin(w http.ResponseWriter, r *http.Request) {
//var req struct {
//Name     string `json:"name"`
//Password string `json:"password"`
//}
//
//	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
//		respondError(w, http.StatusBadRequest, "Invalid request")
//		return
//	}
//
//	ctx := r.Context()
//	driver, err := a.driverRepo.Authenticate(ctx, req.Name, req.Password)
//	if err != nil {
//		if err == ErrInvalidCredentials {
//			respondError(w, http.StatusUnauthorized, err.Error())
//		} else {
//			respondError(w, http.StatusInternalServerError, err.Error())
//		}
//		return
//	}
//
//	if err := a.driverRepo.SetOnline(ctx, driver.ID, true); err != nil {
//		respondError(w, http.StatusInternalServerError, "Error updating driver status")
//		return
//	}
//
//	driver.Online = true
//	a.assignmentService.AssignBestOrder(ctx, driver.ID)
//	respondJSON(w, http.StatusOK, driver)
//}
//
//func (a *App) handleUpdateLocation(w http.ResponseWriter, r *http.Request) {
//driverID := mux.Vars(r)["id"]
//
//	var req struct {
//		Lat     float64 `json:"lat"`
//		Lng     float64 `json:"lng"`
//		Battery string  `json:"battery_level,omitempty"`
//		Network string  `json:"network_info,omitempty"`
//	}
//
//	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
//		respondError(w, http.StatusBadRequest, "Invalid request")
//		return
//	}
//
//	if req.Battery == "" {
//		req.Battery = "n/a"
//	}
//	if req.Network == "" {
//		req.Network = "n/a"
//	}
//
//	location := &Location{Lat: req.Lat, Lng: req.Lng}
//	ctx := r.Context()
//
//	err := a.driverRepo.UpdateLocation(ctx, driverID, location, req.Battery, req.Network)
//	if err != nil {
//		respondError(w, http.StatusInternalServerError, err.Error())
//		return
//	}
//
//	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
//}
//
//func (a *App) handleDriverOrders(w http.ResponseWriter, r *http.Request) {
//driverID := mux.Vars(r)["id"]
//ctx := r.Context()
//
//	orders, err := a.orderRepo.GetByDriver(ctx, driverID)
//	if err != nil {
//		respondError(w, http.StatusInternalServerError, err.Error())
//		return
//	}
//
//	respondJSON(w, http.StatusOK, orders)
//}
//
//func (a *App) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
//driverID := mux.Vars(r)["id"]
//
//	var req struct {
//		OrderID string `json:"order_id"`
//		Status  string `json:"status"`
//	}
//
//	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
//		respondError(w, http.StatusBadRequest, "Invalid request")
//		return
//	}
//
//	ctx := r.Context()
//	err := a.orderRepo.UpdateStatus(ctx, req.OrderID, driverID, req.Status)
//	if err != nil {
//		if err == sql.ErrNoRows {
//			respondError(w, http.StatusNotFound, "Order not found or not assigned to driver")
//		} else {
//			respondError(w, http.StatusInternalServerError, err.Error())
//		}
//		return
//	}
//
//	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
//}
//
//func (a *App) handleDriverLogout(w http.ResponseWriter, r *http.Request) {
//driverID := mux.Vars(r)["id"]
//ctx := r.Context()
//
//	err := a.driverRepo.SetOnline(ctx, driverID, false)
//	if err != nil {
//		respondError(w, http.StatusInternalServerError, err.Error())
//		return
//	}
//
//	respondJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
//}
//
//func (a *App) handleGetDriver(w http.ResponseWriter, r *http.Request) {
//driverID := mux.Vars(r)["id"]
//ctx := r.Context()
//
//	driver, err := a.driverRepo.GetByID(ctx, driverID)
//	if err != nil {
//		if err == sql.ErrNoRows {
//			respondError(w, http.StatusNotFound, "Driver not found")
//		} else {
//			respondError(w, http.StatusInternalServerError, err.Error())
//		}
//		return
//	}
//
//	respondJSON(w, http.StatusOK, driver)
//}
//
//func (a *App) handleGetDriverLocation(w http.ResponseWriter, r *http.Request) {
//driverID := mux.Vars(r)["id"]
//ctx := r.Context()
//
//	var locationStr string
//	err := a.DB.QueryRowContext(ctx, "SELECT location FROM drivers WHERE id = $1", driverID).Scan(&locationStr)
//	if err != nil {
//		respondError(w, http.StatusNotFound, "Driver not found")
//		return
//	}
//
//	location, err := ParseLocation(locationStr)
//	if err != nil {
//		respondError(w, http.StatusInternalServerError, "Invalid location format")
//		return
//	}
//
//	respondJSON(w, http.StatusOK, location)
//}
//
//func (a *App) handleOrderTracking(w http.ResponseWriter, r *http.Request) {
//orderID := mux.Vars(r)["id"]
//ctx := r.Context()
//
//	tracking, err := a.orderRepo.GetTracking(ctx, orderID)
//	if err != nil {
//		respondError(w, http.StatusNotFound, "Order not found")
//		return
//	}
//
//	respondJSON(w, http.StatusOK, tracking)
//}
//
//// HTTP Utilities
//func getPagination(r *http.Request) (offset int, limit int) {
//page := 1
//limit = defaultPageLimit
//
//	if p := r.URL.Query().Get("pageNumber"); p != "" {
//		if pn, err := strconv.Atoi(p); err == nil && pn > 0 {
//			page = pn
//		}
//	}
//
//	if l := r.URL.Query().Get("pageLimit"); l != "" {
//		if pl, err := strconv.Atoi(l); err == nil && pl > 0 {
//			limit = pl
//		}
//	}
//
//	return (page - 1) * limit, limit
//}
//
//func respondJSON(w http.ResponseWriter, status int, data interface{}) {
//w.Header().Set("Content-Type", "application/json")
//w.WriteHeader(status)
//json.NewEncoder(w).Encode(data)
//}
//
//func respondError(w http.ResponseWriter, status int, message string) {
//respondJSON(w, status, map[string]string{"error": message})
//}
//
//func recoveryMiddleware(next http.Handler) http.Handler {
//return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//defer func() {
//if err := recover(); err != nil {
//log.Printf("[PANIC] %v", err)
//respondError(w, http.StatusInternalServerError, "Internal server error")
//}
//}()
//next.ServeHTTP(w, r)
//})
//}
//
//func main() {
//app := NewApp()
//
//	if err := app.InitDB(); err != nil {
//		log.Fatalf("Failed to initialize database: %v", err)
//	}
//
//	app.InitRouter()
//	app.StartBackgroundJobs()
//
//	log.Println("Server running on :8080")
//	log.Fatal(http.ListenAndServe(":8080", app.Router))
//}
