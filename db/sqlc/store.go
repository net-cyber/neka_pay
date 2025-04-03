package db

import (
	"context"
	"database/sql"
	"fmt"
)

var txkey = struct{}{}

type Store interface {
	Querier
	TransferTx(ctx context.Context, args TransferTxParams) (TransferTxResult, error)
	TopUpTx(ctx context.Context, arg TopUpTxParams) (TopUpTxResult, error)
}

// store provides all functions to excute DB queries and transactions
type SQLStore struct {
	db *sql.DB
	*Queries
}

// NewStore creates an new store
func NewStore(db *sql.DB) *SQLStore {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

// execTx excutes a function within a database transaction
func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	q := New(tx)

	err = fn(q)

	if err != nil {
		// If there was an error, attempt to rollback the transaction
		rbErr := tx.Rollback()
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
func (store *SQLStore) TransferTx(ctx context.Context, args TransferTxParams) (TransferTxResult, error) {
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

			fmt.Println(txName, " update account 1")
			if args.FromAccountID < args.ToAccountID {
				result.FromAccount, result.ToAccount, err = addMoney(ctx, q, args.FromAccountID, -args.Amount, args.ToAccountID, args.Amount)
			} else {
				result.ToAccount, result.FromAccount, err = addMoney(ctx, q, args.ToAccountID, args.Amount, args.FromAccountID, -args.Amount)
			}

			return nil

		})
	return result, err
}

func addMoney(
	ctx context.Context,
	q *Queries,
	accountID1 int64,
	amount1 int64,
	accountID2 int64,
	amount2 int64,
) (account1 Account, account2 Account, err error) {
	account1, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID1,
		Amount: amount1,
	})
	if err != nil {
		return
	}

	account2, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID2,
		Amount: amount2,
	})
	return
}

// TopUpTxParams contains the input parameters for the TopUp transaction
type TopUpTxParams struct {
	AccountID int64 `json:"account_id"`
	Amount    int64 `json:"amount"`
}

// TopUpTxResult is the result of the TopUp transaction
type TopUpTxResult struct {
	Transfer Transfer `json:"transfer"`
	Account  Account  `json:"account"`
	Entry    Entry    `json:"entry"`
}

// TopUpTx performs a money top-up transaction on an account
// It creates a transfer record, an entry record, and updates the account balance within a single transaction
func (store *SQLStore) TopUpTx(ctx context.Context, arg TopUpTxParams) (TopUpTxResult, error) {
	var result TopUpTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		txName := ctx.Value(txkey)

		// Get account with lock to ensure serializable isolation
		_, err = q.GetAccountForUpdate(ctx, arg.AccountID)
		if err != nil {
			return err
		}

		// Create a transfer record for the top-up (using 0 as FromAccountID to indicate external source)
		fmt.Println(txName, "Create transfer")
		transfer, err := q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: 0, // 0 means external funding source
			ToAccountID:   arg.AccountID,
			Amount:        arg.Amount,
		})
		if err != nil {
			return err
		}
		result.Transfer = transfer

		// Create an entry record for the top-up
		fmt.Println(txName, "Create Entry")
		result.Entry, err = q.CreateEntry(ctx, CreateEntryParams(arg))
		if err != nil {
			return err
		}

		// Perform the top-up operation
		fmt.Println(txName, "Update Account")
		result.Account, err = q.TopUpAccount(ctx, TopUpAccountParams{
			ID:     arg.AccountID,
			Amount: arg.Amount,
		})

		return err
	})

	return result, err
}
