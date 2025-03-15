package db

import (
	"context"
	"database/sql"
	"testing"
	"time"
	"github.com/net-cyber/neka_pay/util"
	"github.com/stretchr/testify/require"
)

func TestCreateAccount(t *testing.T) {
	arg := CreateAccountParams{
		Owner:    util.RandomOwner(),
		Balance:  util.RandomMoney(),
		Currency: util.RandomCurrency(),
	}
	account, err := testQueries.CreateAccount(context.Background(), arg)
	// check if the error is nil
	require.NoError(t, err)
	// check if the account is not empty
	require.NotEmpty(t, account)

	// check if the account owner is the same as the one we passed
	require.Equal(t, arg.Owner, account.Owner)
	// check if the account balance is the same as the one we passed
	require.Equal(t, arg.Balance, account.Balance)
	// check if the account currency is the same as the one we passed
	require.Equal(t, arg.Currency, account.Currency)

	// check if the account id is not zero
	require.NotZero(t, account.ID)
	// check if the account created at is not zero
	require.NotZero(t, account.CreatedAt)
}
func createRandomAccount(t *testing.T) Account {
	arg := CreateAccountParams{
		Owner:    util.RandomOwner(),
		Balance:  util.RandomMoney(),
		Currency: util.RandomCurrency(),
	}
	account, err := testQueries.CreateAccount(context.Background(), arg)
	// check if the error is nil
	require.NoError(t, err)
	// check if the account is not empty
	require.NotEmpty(t, account)

	// check if the account owner is the same as the one we passed
	require.Equal(t, arg.Owner, account.Owner)
	// check if the account balance is the same as the one we passed
	require.Equal(t, arg.Balance, account.Balance)
	// check if the account currency is the same as the one we passed
	require.Equal(t, arg.Currency, account.Currency)

	// check if the account id is not zero
	require.NotZero(t, account.ID)
	// check if the account created at is not zero
	require.NotZero(t, account.CreatedAt)  

	return account
}

func TestGetAccount(t *testing.T) {
	account1 := createRandomAccount(t)
	account2, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, account2)

	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, account1.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	require.WithinDuration(t, account1.CreatedAt, account2.CreatedAt, time.Second)
}

func TestUpdateAccount(t *testing.T) {
	account1 := createRandomAccount((t))

	arg := UpdateAccountParams{
		ID:      account1.ID,
		Balance: account1.Balance + util.RandomMoney(),
	}
	account2, err := testQueries.UpdateAccount(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, account2)

	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, arg.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	require.WithinDuration(t, account1.CreatedAt, account2.CreatedAt, time.Second)
}

func TestDeleteAccount(t *testing.T) {
	account1 := createRandomAccount(t)
	err := testQueries.DeleteAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	account2, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, account2)
}

func TestListAccounts(t *testing.T) {
	// Create a random owner that we'll use for all accounts
	owner := util.RandomOwner()
	
	// Create 10 random accounts with the same owner
	for i := 0; i < 10; i++ {
		arg := CreateAccountParams{
			Owner: owner,
			Balance:  util.RandomMoney(),
			Currency: util.RandomCurrency(),
		}
		account, err := testQueries.CreateAccount(context.Background(), arg)
		require.NoError(t, err)
		require.NotEmpty(t, account)
	}

	arg := ListAccountsParams{
		Limit:  5,
		Offset: 5,
	}
	accounts, err := testQueries.ListAccounts(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, accounts, 5)

	for _, account := range accounts {
		require.NotEmpty(t, account)
		require.Equal(t, owner, account.Owner)
	}
}
