package biz

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ClientInput struct {
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Address   Address `json:"address"`
	SIREN     string  `json:"siren"`
	SIRET     string  `json:"siret"`
	VATNumber string  `json:"vat_number"`
	IBAN      string  `json:"iban"`
	Notes     string  `json:"notes"`
}

func (s *Store) ListClients(ctx context.Context, userID uuid.UUID) ([]Client, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, COALESCE(email,''), COALESCE(siren,''), COALESCE(siret,''), COALESCE(vat_number,''),
		       COALESCE(notes,''), address_enc, iban_enc, created_at, updated_at,
		       COALESCE(address_line1,''), COALESCE(city,''), COALESCE(postal_code,''), COALESCE(country,'FR')
		FROM clients WHERE user_id = $1 ORDER BY name ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Client{}
	for rows.Next() {
		var c Client
		var addrEnc, ibanEnc []byte
		var a1, city, postal, country string
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.SIREN, &c.SIRET, &c.VATNumber, &c.Notes,
			&addrEnc, &ibanEnc, &c.CreatedAt, &c.UpdatedAt, &a1, &city, &postal, &country); err != nil {
			return nil, err
		}
		c.Address = s.decryptAddr(addrEnc)
		if c.Address.Line1 == "" && a1 != "" {
			c.Address = Address{Line1: a1, City: city, PostalCode: postal, Country: country}
		}
		c.IBAN = s.decryptIBAN(ibanEnc)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetClient(ctx context.Context, userID, id uuid.UUID) (Client, error) {
	var c Client
	var addrEnc, ibanEnc []byte
	var a1, city, postal, country string
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, COALESCE(email,''), COALESCE(siren,''), COALESCE(siret,''), COALESCE(vat_number,''),
		       COALESCE(notes,''), address_enc, iban_enc, created_at, updated_at,
		       COALESCE(address_line1,''), COALESCE(city,''), COALESCE(postal_code,''), COALESCE(country,'FR')
		FROM clients WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&c.ID, &c.Name, &c.Email, &c.SIREN, &c.SIRET, &c.VATNumber, &c.Notes,
		&addrEnc, &ibanEnc, &c.CreatedAt, &c.UpdatedAt, &a1, &city, &postal, &country)
	if errors.Is(err, pgx.ErrNoRows) {
		return Client{}, ErrNotFound
	}
	if err != nil {
		return Client{}, err
	}
	c.Address = s.decryptAddr(addrEnc)
	if c.Address.Line1 == "" && a1 != "" {
		c.Address = Address{Line1: a1, City: city, PostalCode: postal, Country: country}
	}
	c.IBAN = s.decryptIBAN(ibanEnc)
	return c, nil
}

func (s *Store) CreateClient(ctx context.Context, userID uuid.UUID, in ClientInput) (Client, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Client{}, ErrInvalidInput
	}
	if in.Address.Country == "" {
		in.Address.Country = "FR"
	}
	addrEnc, err := s.encryptAddr(in.Address)
	if err != nil {
		return Client{}, err
	}
	ibanEnc, err := s.encryptIBAN(strings.TrimSpace(in.IBAN))
	if err != nil {
		return Client{}, err
	}
	var c Client
	err = s.pool.QueryRow(ctx, `
		INSERT INTO clients (user_id, name, email, address_line1, address_line2, postal_code, city, country,
		                     siren, siret, vat_number, notes, address_enc, iban_enc)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, created_at, updated_at
	`, userID, name, nullStr(in.Email), nullStr(in.Address.Line1), nullStr(in.Address.Line2),
		nullStr(in.Address.PostalCode), nullStr(in.Address.City), in.Address.Country,
		nullStr(in.SIREN), nullStr(in.SIRET), nullStr(in.VATNumber), nullStr(in.Notes), addrEnc, ibanEnc,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return Client{}, err
	}
	c.Name = name
	c.Email = in.Email
	c.Address = in.Address
	c.SIREN = in.SIREN
	c.SIRET = in.SIRET
	c.VATNumber = in.VATNumber
	c.IBAN = in.IBAN
	c.Notes = in.Notes
	return c, nil
}

func (s *Store) UpdateClient(ctx context.Context, userID, id uuid.UUID, in ClientInput) (Client, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Client{}, ErrInvalidInput
	}
	if in.Address.Country == "" {
		in.Address.Country = "FR"
	}
	addrEnc, err := s.encryptAddr(in.Address)
	if err != nil {
		return Client{}, err
	}
	ibanEnc, err := s.encryptIBAN(strings.TrimSpace(in.IBAN))
	if err != nil {
		return Client{}, err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE clients SET name=$1, email=$2, address_line1=$3, address_line2=$4, postal_code=$5, city=$6, country=$7,
		                   siren=$8, siret=$9, vat_number=$10, notes=$11, address_enc=$12, iban_enc=$13, updated_at=NOW()
		WHERE id=$14 AND user_id=$15
	`, name, nullStr(in.Email), nullStr(in.Address.Line1), nullStr(in.Address.Line2),
		nullStr(in.Address.PostalCode), nullStr(in.Address.City), in.Address.Country,
		nullStr(in.SIREN), nullStr(in.SIRET), nullStr(in.VATNumber), nullStr(in.Notes), addrEnc, ibanEnc, id, userID)
	if err != nil {
		return Client{}, err
	}
	if tag.RowsAffected() == 0 {
		return Client{}, ErrNotFound
	}
	return s.GetClient(ctx, userID, id)
}

func (s *Store) DeleteClient(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM clients WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func nullStr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
