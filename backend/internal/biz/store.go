package biz

import (
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tainalo2/devis_generator/backend/internal/crypto"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrForbidden    = errors.New("forbidden")
	ErrImmutable    = errors.New("document is issued and immutable")
	ErrInvalidInput = errors.New("invalid input")
)

type Store struct {
	pool *pgxpool.Pool
	box  *crypto.Box
}

func NewStore(pool *pgxpool.Pool, box *crypto.Box) *Store {
	return &Store{pool: pool, box: box}
}

func (s *Store) encryptAddr(a Address) ([]byte, error) {
	if s.box == nil {
		return nil, nil
	}
	if a.Line1 == "" && a.Line2 == "" && a.PostalCode == "" && a.City == "" {
		return nil, nil
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return s.box.Encrypt(raw)
}

func (s *Store) decryptAddr(blob []byte) Address {
	if s.box == nil || len(blob) == 0 {
		return Address{Country: "FR"}
	}
	plain, err := s.box.Decrypt(blob)
	if err != nil {
		return Address{Country: "FR"}
	}
	var a Address
	_ = json.Unmarshal(plain, &a)
	if a.Country == "" {
		a.Country = "FR"
	}
	return a
}

func (s *Store) encryptIBAN(iban string) ([]byte, error) {
	if s.box == nil || iban == "" {
		return nil, nil
	}
	return s.box.EncryptString(iban)
}

func (s *Store) decryptIBAN(blob []byte) string {
	if s.box == nil || len(blob) == 0 {
		return ""
	}
	v, err := s.box.DecryptString(blob)
	if err != nil {
		return ""
	}
	return v
}
