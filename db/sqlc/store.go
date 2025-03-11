package db

import (
	"context"
	"database/sql"
	"fmt"
)

var txkey = struct{}{}

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
		 rbErr := tx.Rollback();
		//  If there was an error during the rollback,
		if rbErr != nil {
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

			txName := ctx.Value(txkey)

			fmt.Println(txName, "Create transfer")

			result.Transfer, err = q.CreateTransfer(
				ctx,
				CreateTransferParams(args))
			if err != nil {
				return err
			}

			fmt.Println(txName, "Create Entry 1")

			// create from entry in the DB
			result.FromEntry, err = q.CreateEntry(
				ctx,
				CreateEntryParams{
					AccountID: args.FromAccountID,
					Amount:    -args.Amount,
				})

			if err != nil {
				return err
			}

			fmt.Println(txName, "Create Entry 2")
			// create to entry in the DB
			result.ToEntry, err = q.CreateEntry(
				ctx,
				CreateEntryParams{
					AccountID: args.ToAccountID,
					Amount:    args.Amount,
				})

			if err != nil {
				return err
			}

			fmt.Println(txName, "get account 1 for update")
			// TODO: update accounts balance
			// substract form account 1
			Account1, err := q.GetAccountForUpdate(ctx, args.FromAccountID)
			if err != nil {
				return err
			}
			fmt.Println(txName, " update account 1")
			result.FromAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
				ID: args.FromAccountID,
				Balance: Account1.Balance - args.Amount,

			})

			if err != nil {
				return err
			}
			fmt.Println(txName, "get account 2 for update")
			// add to account 2
			account2, err := q.GetAccountForUpdate(ctx, args.ToAccountID)
			if err != nil {
				return err
			}

			fmt.Println(txName, " update account 2")
			result.ToAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
				ID: args.ToAccountID,
				Balance: account2.Balance + args.Amount,

			})

			if err != nil {
				return err
			}

			return nil

		})
	return result, err
}
