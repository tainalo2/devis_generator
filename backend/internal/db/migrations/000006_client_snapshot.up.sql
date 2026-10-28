-- Client figé à l'émission : un document émis ne doit plus dépendre de la fiche
-- client (modification ou suppression). L'adresse y est stockée chiffrée.
ALTER TABLE documents ADD COLUMN IF NOT EXISTS client_snapshot JSONB;
