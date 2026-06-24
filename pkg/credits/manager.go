package credits

import (
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrInsufficientCredits = errors.New("insufficient credits")
)

type DB interface {
	Begin() (*sql.Tx, error)
}

type Manager struct {
	db DB
}

func NewManager(db DB) *Manager {
	return &Manager{db: db}
}

type CreditTx struct {
	UserID       int64
	ChangeAmount int
	Reason       string
	RefID        int64
}

func (m *Manager) Deduct(userID int64, amount int, reason string, refID int64) (int, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var balance int
	err = tx.QueryRow("SELECT credits FROM users WHERE id = $1 FOR UPDATE", userID).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("query balance: %w", err)
	}

	if balance < amount {
		return 0, ErrInsufficientCredits
	}

	newBalance := balance - amount

	_, err = tx.Exec("UPDATE users SET credits = $1, total_bets = total_bets + 1 WHERE id = $2", newBalance, userID)
	if err != nil {
		return 0, fmt.Errorf("update balance: %w", err)
	}

	_, err = tx.Exec(`INSERT INTO credit_logs (user_id, change_amount, balance_after, reason, ref_id) VALUES ($1, $2, $3, $4, $5)`,
		userID, -amount, newBalance, reason, refID)
	if err != nil {
		return 0, fmt.Errorf("insert credit log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	return newBalance, nil
}

func (m *Manager) Refund(userID int64, amount int, reason string, refID int64) (int, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var balance int
	err = tx.QueryRow("SELECT credits FROM users WHERE id = $1 FOR UPDATE", userID).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("query balance: %w", err)
	}

	newBalance := balance + amount

	_, err = tx.Exec("UPDATE users SET credits = $1 WHERE id = $2", newBalance, userID)
	if err != nil {
		return 0, fmt.Errorf("update balance: %w", err)
	}

	_, err = tx.Exec(`INSERT INTO credit_logs (user_id, change_amount, balance_after, reason, ref_id) VALUES ($1, $2, $3, $4, $5)`,
		userID, amount, newBalance, reason, refID)
	if err != nil {
		return 0, fmt.Errorf("insert credit log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	return newBalance, nil
}

func (m *Manager) Award(userID int64, amount int, reason string, refID int64) (int, error) {
	return m.Refund(userID, amount, reason, refID)
}

func (m *Manager) Add(userID int64, amount int, reason string) (int, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var balance int
	err = tx.QueryRow("SELECT credits FROM users WHERE id = $1 FOR UPDATE", userID).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("query balance: %w", err)
	}

	newBalance := balance + amount

	_, err = tx.Exec("UPDATE users SET credits = $1 WHERE id = $2", newBalance, userID)
	if err != nil {
		return 0, fmt.Errorf("update balance: %w", err)
	}

	_, err = tx.Exec(`INSERT INTO credit_logs (user_id, change_amount, balance_after, reason) VALUES ($1, $2, $3, $4)`,
		userID, amount, newBalance, reason)
	if err != nil {
		return 0, fmt.Errorf("insert credit log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	return newBalance, nil
}
