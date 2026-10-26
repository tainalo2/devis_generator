-- Suivi d'un document émis : envoi, acceptation (devis), paiement (factures).
-- Le contenu reste immuable ; seuls le statut et ces dates évoluent.
ALTER TABLE documents ADD COLUMN IF NOT EXISTS sent_at TIMESTAMPTZ;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS accepted_at TIMESTAMPTZ;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ;
