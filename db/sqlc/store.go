package db

import (
	"context"
	"database/sql"
	"fmt"
)

// store provides all functions to excute DB queries and transactions
type Store struct {
	*Queries
	db *sql.DB
}

// NewStore creates an new store
func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

// execTx excutes a function within a database transaction
func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	q := New(tx)

	err = fn(q)

	if err != nil {
		// If there was an error, attempt to rollback the transaction
		if rbErr := tx.Rollback();
		//  If there was an error during the rollback,
		rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// this will contain all the fields to perform transfer money
type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

// this will have the result of transfer transaction
type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

// perform money transaction form one account to another account
// it creates a transfer recored, add account entries, and update accounts balance within a single database transaction
func (store *Store) TransferTx(ctx context.Context, args TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(
		ctx,
		func(q *Queries) error {
			var err error
			result.Transfer, err = q.CreateTransfer(ctx,
				CreateTransferParams{
					FromAccountID: args.FromAccountID,
					ToAccountID:   args.ToAccountID,
					Amount:        args.Amount,
				})
			if err != nil {
				return err
			}
			result.FromEntry, err = q.CreateEntry(
				ctx,
				CreateEntryParams{
					AccountID: args.FromAccountID,
					Amount:    -args.Amount,
				})

			if err != nil {
				return err
			}

			result.ToEntry, err = q.CreateEntry(
				ctx,
				CreateEntryParams{
					AccountID: args.ToAccountID,
					Amount:    args.Amount,
				})

			if err != nil {
				return err
			}
			// TODO: update accounts balance
			return nil

		})
	return result, err
}
