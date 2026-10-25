package biz

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tainalo2/devis_generator/backend/internal/billing"
)

func (s *Store) GetIssuer(ctx context.Context, userID uuid.UUID) (IssuerProfile, error) {
	var p IssuerProfile
	var addrEnc, ibanEnc []byte
	err := s.pool.QueryRow(ctx, `
		SELECT legal_name, trade_name, email, phone, siret, siren, vat_number, ape_code, rcs,
		       default_vat_regime::text, payment_terms, address_enc, iban_enc, updated_at
		FROM issuer_profiles WHERE user_id = $1
	`, userID).Scan(
		&p.LegalName, &p.TradeName, &p.Email, &p.Phone, &p.SIRET, &p.SIREN, &p.VATNumber, &p.APECode, &p.RCS,
		&p.DefaultVATRegime, &p.PaymentTerms, &addrEnc, &ibanEnc, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return IssuerProfile{DefaultVATRegime: billing.RegimeFranchise293B, Address: Address{Country: "FR"}}, nil
	}
	if err != nil {
		return IssuerProfile{}, err
	}
	p.Address = s.decryptAddr(addrEnc)
	p.IBAN = s.decryptIBAN(ibanEnc)
	return p, nil
}

func (s *Store) UpsertIssuer(ctx context.Context, userID uuid.UUID, in IssuerProfile) (IssuerProfile, error) {
	if in.DefaultVATRegime == "" {
		in.DefaultVATRegime = billing.RegimeFranchise293B
	}
	if in.DefaultVATRegime != billing.RegimeFranchise293B && in.DefaultVATRegime != billing.RegimeStandard {
		return IssuerProfile{}, ErrInvalidInput
	}
	if in.Address.Country == "" {
		in.Address.Country = "FR"
	}
	addrEnc, err := s.encryptAddr(in.Address)
	if err != nil {
		return IssuerProfile{}, err
	}
	ibanEnc, err := s.encryptIBAN(strings.TrimSpace(in.IBAN))
	if err != nil {
		return IssuerProfile{}, err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO issuer_profiles (
			user_id, legal_name, trade_name, email, phone, siret, siren, vat_number, ape_code, rcs,
			default_vat_regime, payment_terms, address_enc, iban_enc, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::vat_regime,$12,$13,$14,NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			legal_name=EXCLUDED.legal_name, trade_name=EXCLUDED.trade_name, email=EXCLUDED.email,
			phone=EXCLUDED.phone, siret=EXCLUDED.siret, siren=EXCLUDED.siren, vat_number=EXCLUDED.vat_number,
			ape_code=EXCLUDED.ape_code, rcs=EXCLUDED.rcs, default_vat_regime=EXCLUDED.default_vat_regime,
			payment_terms=EXCLUDED.payment_terms, address_enc=EXCLUDED.address_enc, iban_enc=EXCLUDED.iban_enc,
			updated_at=NOW()
	`, userID, in.LegalName, in.TradeName, in.Email, in.Phone, in.SIRET, in.SIREN, in.VATNumber, in.APECode, in.RCS,
		in.DefaultVATRegime, in.PaymentTerms, addrEnc, ibanEnc)
	if err != nil {
		return IssuerProfile{}, err
	}
	return s.GetIssuer(ctx, userID)
}
