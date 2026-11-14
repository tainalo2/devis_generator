package stripeconnect

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tainalo2/devis_generator/backend/internal/db"
)

type fakeAccountClient struct {
	account *ConnectedAccount
	created int
}

func (f *fakeAccountClient) CreateAccount(context.Context, string, string) (*ConnectedAccount, error) {
	f.created++
	return f.account, nil
}
func (f *fakeAccountClient) CreateAccountLink(context.Context, string, string, string) (string, error) {
	return "https://connect.test/onboard", nil
}
func (f *fakeAccountClient) RetrieveAccount(context.Context, string) (*ConnectedAccount, error) {
	return f.account, nil
}

func TestOnboardingAndRefreshPersistOwnershipCapabilities(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	if err := db.RunMigrations(dsn); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	owner := uuid.New()
	if _, err = pool.Exec(ctx, `INSERT INTO users(id,email,display_name) VALUES($1,$2,'connect test')`, owner, fmt.Sprintf("connect-%s@example.test", owner)); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, owner)
	client := &fakeAccountClient{account: &ConnectedAccount{ID: "acct_owned", Country: "FR", DefaultCurrency: "EUR", ChargesEnabled: true, PayoutsEnabled: true, DetailsSubmitted: true, Capabilities: map[string]string{"card_payments": "active", "transfers": "active"}}}
	svc := NewService(pool, client, Config{Country: "FR", Currency: "EUR", PublicURL: "https://app.test"})
	if link, err := svc.Start(ctx, owner, "owner@example.test"); err != nil || link == "" {
		t.Fatalf("start link=%q err=%v", link, err)
	}
	client.account.Capabilities["transfers"] = "inactive"
	if _, err = svc.Refresh(ctx, owner); err != nil {
		t.Fatal(err)
	}
	var storedOwner uuid.UUID
	var cards, transfers bool
	var currency string
	if err = pool.QueryRow(ctx, `SELECT owner_id,card_payments_active,transfers_active,default_currency FROM stripe_connect_accounts WHERE account_id='acct_owned'`).Scan(&storedOwner, &cards, &transfers, &currency); err != nil {
		t.Fatal(err)
	}
	if storedOwner != owner || !cards || transfers || currency != "EUR" {
		t.Fatalf("owner=%s cards=%v transfers=%v currency=%s", storedOwner, cards, transfers, currency)
	}
}
