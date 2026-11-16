package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RecordSignature stores signature metadata on the document record using Sig-T2 schema.
func (s *Service) RecordSignature(ctx context.Context, sig Signature) error {
	if err := s.db.WithTx(ctx, func(tx DB) error {
		docHash := generateDocumentHash(sig.DocumentID.String())
		sig.DocumentHash = docHash
		sig.ID = uuid.New()

		err := tx.ExecContext(
			ctx,
			`UPDATE documents SET signed_at = $1, document_hash = $2, signer_ip = $3, signer_user_agent = $4, signer_name_declared = $5, updated_at = $6 WHERE id = $7`,
			sig.SignedAt, sig.DocumentHash, sig.SignerIP, sig.SignerUserAgent, sig.SignerNameDeclared, time.Now().UTC(), sig.DocumentID,
		)
		if err != nil {
			return fmt.Errorf("update document signature: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func generateDocumentHash(docID string) string {
	h := sha256.New()
	h.Write([]byte(docID))
	return hex.EncodeToString(h.Sum(nil))
}
