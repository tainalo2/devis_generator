package bankcoordsvault

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	appcrypto "github.com/tainalo2/devis_generator/backend/internal/crypto"
	"github.com/tainalo2/devis_generator/backend/internal/db"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

func TestCreateOrUpdateAndRead(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.RunMigrations(databaseURL); err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, display_name) VALUES ($1, $2, 'test')`, userID, fmt.Sprintf("bank-coord-%s@example.test", userID)); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)

	box, err := appcrypto.NewBox(bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}

	vaultRepo := paymentsecrets.NewPGRepository(pool)
	vault := paymentsecrets.NewVault(vaultRepo, box)
	coordRepo := NewRepository(pool, vault)

	actor := paymentsecrets.Actor{
		UserID:    userID,
		RequestID: "test-req-123",
	}

	// Create bank coordinates
	coords, err := coordRepo.CreateOrUpdate(ctx, actor, userID,
		"John Doe",
		"FR1420041010050500013M02606",
		"BNPAFRPP",
		"BNP Paribas",
	)
	if err != nil {
		t.Fatalf("CreateOrUpdate failed: %v", err)
	}

	if coords.AccountHolderName != "John Doe" {
		t.Errorf("account holder name mismatch: got %s, want John Doe", coords.AccountHolderName)
	}
	if coords.BankName != "BNP Paribas" {
		t.Errorf("bank name mismatch: got %s, want BNP Paribas", coords.BankName)
	}

	// Read and verify plaintext
	name, iban, bic, bank, err := coordRepo.Read(ctx, actor, userID)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if name != "John Doe" || iban != "FR1420041010050500013M02606" || bic != "BNPAFRPP" || bank != "BNP Paribas" {
		t.Errorf("read mismatch: got (%s, %s, %s, %s)", name, iban, bic, bank)
	}

	// Verify Exists
	exists, err := coordRepo.Exists(ctx, userID)
	if err != nil || !exists {
		t.Errorf("Exists failed or returned false: %v, %v", err, exists)
	}

	// Update bank coordinates (BIC removal)
	coords2, err := coordRepo.CreateOrUpdate(ctx, actor, userID,
		"Jane Doe",
		"FR1420041010050500013M02606",
		"", // No BIC
		"BNP Paribas",
	)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	name2, iban2, bic2, bank2, err := coordRepo.Read(ctx, actor, userID)
	if err != nil {
		t.Fatalf("Read after update failed: %v", err)
	}

	if name2 != "Jane Doe" {
		t.Errorf("updated name mismatch: got %s, want Jane Doe", name2)
	}
	if bic2 != "" {
		t.Errorf("BIC should be empty after update: got %s", bic2)
	}

	// Delete
	err = coordRepo.Delete(ctx, actor, userID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	exists, err = coordRepo.Exists(ctx, userID)
	if err != nil || exists {
		t.Errorf("Exists after delete should be false: %v, %v", err, exists)
	}

	_, _, _, _, err = coordRepo.Read(ctx, actor, userID)
	if err != ErrNotFound {
		t.Errorf("Read after delete should return ErrNotFound, got %v", err)
	}
}

func TestMultiTenantIsolation(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.RunMigrations(databaseURL); err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	// Create two users
	user1 := uuid.New()
	user2 := uuid.New()
	for _, id := range []uuid.UUID{user1, user2} {
		if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, display_name) VALUES ($1, $2, 'test')`, id, fmt.Sprintf("user-%s@example.test", id)); err != nil {
			t.Fatal(err)
		}
		defer pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	}

	box, err := appcrypto.NewBox(bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}

	vaultRepo := paymentsecrets.NewPGRepository(pool)
	vault := paymentsecrets.NewVault(vaultRepo, box)
	coordRepo := NewRepository(pool, vault)

	actor1 := paymentsecrets.Actor{UserID: user1, RequestID: "req1"}
	actor2 := paymentsecrets.Actor{UserID: user2, RequestID: "req2"}

	// User1 creates coordinates
	_, err = coordRepo.CreateOrUpdate(ctx, actor1, user1,
		"Alice",
		"FR1420041010050500013M02606",
		"BNPAFRPP",
		"Bank A",
	)
	if err != nil {
		t.Fatalf("User1 CreateOrUpdate failed: %v", err)
	}

	// User2 creates coordinates
	_, err = coordRepo.CreateOrUpdate(ctx, actor2, user2,
		"Bob",
		"DE89370400440532013000",
		"COBADEFF",
		"Bank B",
	)
	if err != nil {
		t.Fatalf("User2 CreateOrUpdate failed: %v", err)
	}

	// Verify isolation: User1 cannot access User2's coordinates
	// (They can query the DB, but the vault access control should prevent plaintext reveal)
	metadata1, err := coordRepo.GetMetadata(ctx, user1)
	if err != nil {
		t.Fatalf("GetMetadata user1 failed: %v", err)
	}
	if metadata1.AccountHolderName != "Alice" {
		t.Errorf("user1 metadata mismatch")
	}

	metadata2, err := coordRepo.GetMetadata(ctx, user2)
	if err != nil {
		t.Fatalf("GetMetadata user2 failed: %v", err)
	}
	if metadata2.AccountHolderName != "Bob" {
		t.Errorf("user2 metadata mismatch")
	}

	// Verify that reads can only happen with correct actor/owner combo
	// User1 tries to reveal User2's secret (should fail due to vault access control)
	_, _, _, _, err = coordRepo.Read(ctx, actor1, user2)
	if err == nil {
		t.Error("User1 should not be able to read User2's coordinates (cross-tenant access)")
	}
}

func TestInvalidIBAN(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.RunMigrations(databaseURL); err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, display_name) VALUES ($1, $2, 'test')`, userID, fmt.Sprintf("user-%s@example.test", userID)); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)

	box, err := appcrypto.NewBox(bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}

	vaultRepo := paymentsecrets.NewPGRepository(pool)
	vault := paymentsecrets.NewVault(vaultRepo, box)
	coordRepo := NewRepository(pool, vault)

	actor := paymentsecrets.Actor{UserID: userID, RequestID: "test"}

	_, err = coordRepo.CreateOrUpdate(ctx, actor, userID,
		"John",
		"INVALID_IBAN",
		"",
		"Bank",
	)
	if err == nil {
		t.Error("should reject invalid IBAN")
	}
}

func TestInvalidBIC(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.RunMigrations(databaseURL); err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, display_name) VALUES ($1, $2, 'test')`, userID, fmt.Sprintf("user-%s@example.test", userID)); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)

	box, err := appcrypto.NewBox(bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}

	vaultRepo := paymentsecrets.NewPGRepository(pool)
	vault := paymentsecrets.NewVault(vaultRepo, box)
	coordRepo := NewRepository(pool, vault)

	actor := paymentsecrets.Actor{UserID: userID, RequestID: "test"}

	_, err = coordRepo.CreateOrUpdate(ctx, actor, userID,
		"John",
		"FR1420041010050500013M02606",
		"INVALID_BIC_123", // Invalid format
		"Bank",
	)
	if err == nil {
		t.Error("should reject invalid BIC")
	}
}

func TestEmptyHolderName(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.RunMigrations(databaseURL); err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, display_name) VALUES ($1, $2, 'test')`, userID, fmt.Sprintf("user-%s@example.test", userID)); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)

	box, err := appcrypto.NewBox(bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}

	vaultRepo := paymentsecrets.NewPGRepository(pool)
	vault := paymentsecrets.NewVault(vaultRepo, box)
	coordRepo := NewRepository(pool, vault)

	actor := paymentsecrets.Actor{UserID: userID, RequestID: "test"}

	_, err = coordRepo.CreateOrUpdate(ctx, actor, userID,
		"", // Empty holder name
		"FR1420041010050500013M02606",
		"",
		"Bank",
	)
	if err == nil {
		t.Error("should reject empty holder name")
	}
}
