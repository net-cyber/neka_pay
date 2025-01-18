package db

import (
	"context"
	"testing"
	"github.com/stretchr/testify/require"
)

func TestCreateAccount(t *testing.T) {
	arg := createAccountParams{
		Owner:    "nati",
		Balance:  1000,
		Currency: "ETB",
	}
	account, err := testQueries.createAccount(context.Background(), arg)
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