package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// storedIssuerSnapshot est la forme persistée de l'émetteur figé à l'émission
// (documents.issuer_snapshot) : comme issuer_profiles, l'adresse et l'IBAN
// n'y figurent que chiffrés.
type storedIssuerSnapshot struct {
	IssuerProfile
	AddressEnc []byte `json:"address_enc,omitempty"`
	IBANEnc    []byte `json:"iban_enc,omitempty"`
}

// sealIssuerSnapshot chiffre les champs sensibles avant persistance.
// Sans clé de chiffrement configurée, l'instantané reste en clair, comme le
// reste des données de l'application dans ce cas.
func (s *Store) sealIssuerSnapshot(p IssuerProfile) ([]byte, error) {
	if s.box == nil {
		return json.Marshal(p)
	}
	addrEnc, err := s.encryptAddr(p.Address)
	if err != nil {
		return nil, err
	}
	ibanEnc, err := s.encryptIBAN(p.IBAN)
	if err != nil {
		return nil, err
	}
	stored := storedIssuerSnapshot{IssuerProfile: p, AddressEnc: addrEnc, IBANEnc: ibanEnc}
	stored.Address = Address{}
	stored.IBAN = ""
	return json.Marshal(stored)
}

// openIssuerSnapshot rend l'émetteur en clair à partir de la forme persistée.
// Les instantanés antérieurs au chiffrement (adresse et IBAN en clair) sont lus tels quels.
func (s *Store) openIssuerSnapshot(raw []byte) (IssuerProfile, error) {
	var stored storedIssuerSnapshot
	if err := json.Unmarshal(raw, &stored); err != nil {
		return IssuerProfile{}, err
	}
	p := stored.IssuerProfile
	if len(stored.AddressEnc) > 0 {
		p.Address = s.decryptAddr(stored.AddressEnc)
	}
	if len(stored.IBANEnc) > 0 {
		p.IBAN = s.decryptIBAN(stored.IBANEnc)
	}
	return p, nil
}

// isPlainSnapshot indique un instantané qui contient encore l'adresse ou l'IBAN en clair.
func isPlainSnapshot(raw []byte) bool {
	var probe struct {
		IBAN    string  `json:"iban"`
		Address Address `json:"address"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return false
	}
	a := probe.Address
	return probe.IBAN != "" || a.Line1 != "" || a.Line2 != "" || a.PostalCode != "" || a.City != ""
}

// SealLegacySnapshots chiffre les instantanés émetteur enregistrés en clair
// avant l'introduction du chiffrement. Idempotent : à lancer au démarrage.
func (s *Store) SealLegacySnapshots(ctx context.Context) (int, error) {
	if s.box == nil {
		return 0, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT id, issuer_snapshot FROM documents WHERE issuer_snapshot IS NOT NULL`)
	if err != nil {
		return 0, err
	}
	type pending struct {
		id  uuid.UUID
		raw []byte
	}
	var todo []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.raw); err != nil {
			rows.Close()
			return 0, err
		}
		if isPlainSnapshot(p.raw) {
			todo = append(todo, p)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	sealed := 0
	for _, p := range todo {
		profile, err := s.openIssuerSnapshot(p.raw)
		if err != nil {
			slog.Warn("instantané émetteur illisible", "document", p.id, "error", err)
			continue
		}
		out, err := s.sealIssuerSnapshot(profile)
		if err != nil {
			return sealed, fmt.Errorf("chiffrement de l'instantané %s : %w", p.id, err)
		}
		if _, err := s.pool.Exec(ctx, `UPDATE documents SET issuer_snapshot=$1 WHERE id=$2`, out, p.id); err != nil {
			return sealed, err
		}
		sealed++
	}
	return sealed, nil
}
