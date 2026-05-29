package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go-commerce-api/internal/domain"
	"go-commerce-api/internal/service"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) CreateOrder(ctx context.Context, command service.CreateOrderCommand) (domain.Order, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.Order{}, err
	}
	defer tx.Rollback()

	var stock int
	err = tx.QueryRowContext(ctx, `SELECT stock FROM items WHERE id = $1 FOR UPDATE`, command.Order.ItemID).Scan(&stock)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Order{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Order{}, err
	}
	if stock < command.Order.Quantity {
		return domain.Order{}, domain.ErrOutOfStock
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE items SET stock = stock - $1, updated_at = $2 WHERE id = $3`,
		command.Order.Quantity,
		command.Order.CreatedAt,
		command.Order.ItemID,
	); err != nil {
		return domain.Order{}, err
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO orders (id, user_id, item_id, quantity, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		command.Order.ID,
		command.Order.UserID,
		command.Order.ItemID,
		command.Order.Quantity,
		string(command.Order.Status),
		command.Order.CreatedAt,
	); err != nil {
		return domain.Order{}, err
	}

	payload, err := json.Marshal(command.Event)
	if err != nil {
		return domain.Order{}, err
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO outbox_events (id, topic, payload, status, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		command.Event.EventID,
		domain.OutboxTopicOrderCreated,
		payload,
		string(domain.OutboxStatusPending),
		command.Order.CreatedAt,
	); err != nil {
		return domain.Order{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Order{}, err
	}
	return command.Order, nil
}

func (s *Store) FindOrderByID(ctx context.Context, id string) (domain.Order, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, item_id, quantity, status, created_at
		 FROM orders
		 WHERE id = $1`,
		id,
	)
	return scanOrder(row)
}

func (s *Store) ListOrders(ctx context.Context) ([]domain.Order, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, user_id, item_id, quantity, status, created_at
		 FROM orders
		 ORDER BY created_at, id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (s *Store) FindItemByID(ctx context.Context, id string) (domain.Item, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, stock, price_cents, created_at, updated_at
		 FROM items
		 WHERE id = $1`,
		id,
	)
	return scanItem(row)
}

func (s *Store) ListItems(ctx context.Context) ([]domain.Item, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, name, stock, price_cents, created_at, updated_at
		 FROM items
		 ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) FetchPendingOutboxEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	if limit < 1 {
		limit = 1
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(
		ctx,
		`UPDATE outbox_events
		 SET status = $1, attempts = attempts + 1
		 WHERE id IN (
		 	SELECT id
		 	FROM outbox_events
		 	WHERE status = $2
		 	ORDER BY created_at, id
		 	LIMIT $3
		 	FOR UPDATE SKIP LOCKED
		 )
		 RETURNING id, topic, payload, status, attempts, last_error, created_at, published_at`,
		string(domain.OutboxStatusPublishing),
		string(domain.OutboxStatusPending),
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []domain.OutboxEvent
	for rows.Next() {
		event, err := scanOutboxEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *Store) MarkOutboxPublished(ctx context.Context, id string, publishedAt time.Time) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE outbox_events
		 SET status = $1, published_at = $2, last_error = NULL
		 WHERE id = $3`,
		string(domain.OutboxStatusPublished),
		publishedAt,
		id,
	)
	return err
}

func (s *Store) MarkOutboxFailed(ctx context.Context, id string, errText string, maxAttempts int) error {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE outbox_events
		 SET status = CASE WHEN attempts >= $2 THEN $3 ELSE $4 END,
		     last_error = $5
		 WHERE id = $1`,
		id,
		maxAttempts,
		string(domain.OutboxStatusFailed),
		string(domain.OutboxStatusPending),
		errText,
	)
	return err
}

func (s *Store) MarkEventProcessed(ctx context.Context, eventID, topic string, processedAt time.Time) (bool, error) {
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO processed_events (event_id, topic, processed_at)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (event_id) DO NOTHING`,
		eventID,
		topic,
		processedAt,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanOrder(row scanner) (domain.Order, error) {
	var order domain.Order
	var status string
	err := row.Scan(&order.ID, &order.UserID, &order.ItemID, &order.Quantity, &status, &order.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Order{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Order{}, err
	}
	order.Status = domain.OrderStatus(status)
	return order, nil
}

func scanItem(row scanner) (domain.Item, error) {
	var item domain.Item
	err := row.Scan(&item.ID, &item.Name, &item.Stock, &item.PriceCents, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Item{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

func scanOutboxEvent(row scanner) (domain.OutboxEvent, error) {
	var event domain.OutboxEvent
	var status string
	var lastError sql.NullString
	var publishedAt sql.NullTime
	if err := row.Scan(
		&event.ID,
		&event.Topic,
		&event.Payload,
		&status,
		&event.Attempts,
		&lastError,
		&event.CreatedAt,
		&publishedAt,
	); err != nil {
		return domain.OutboxEvent{}, fmt.Errorf("scan outbox event: %w", err)
	}
	event.Status = domain.OutboxStatus(status)
	if lastError.Valid {
		event.LastError = lastError.String
	}
	if publishedAt.Valid {
		event.PublishedAt = &publishedAt.Time
	}
	return event, nil
}
