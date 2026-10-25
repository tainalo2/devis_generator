-- Historique append-only des valeurs légales.
-- Choix : nouvelle table legal_values (append-only) plutôt que de muter legal_variables.
-- legal_variables reste comme projection « dernière valeur vérifiée » pour lectures rapides
-- et compatibilité avec le Store Upsert/Get existant.

CREATE TABLE IF NOT EXISTS legal_values (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT NOT NULL,
    value JSONB NOT NULL,
    source TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('verified', 'stale', 'error', 'manual')),
    fetched_at TIMESTAMPTZ NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS legal_values_key_fetched_idx
    ON legal_values (key, fetched_at DESC);

CREATE INDEX IF NOT EXISTS legal_values_status_idx
    ON legal_values (status);

-- Seed manuel une seule fois (append-only : pas d'UNIQUE sur key).
INSERT INTO legal_values (key, value, source, status, fetched_at)
SELECT v.key, v.value, v.source, v.status, NOW()
FROM (VALUES
  ('manual.franchise_thresholds',
   '{"services_cents":36800000,"goods_cents":91800000,"note":"Seuils indicatifs franchise en base — à valider juridiquement","currency":"EUR"}'::jsonb,
   'manual-surveillance', 'manual'),
  ('manual.micro_thresholds',
   '{"services_cents":77700000,"goods_cents":188700000,"note":"Seuils micro-entreprise indicatifs — à valider","currency":"EUR"}'::jsonb,
   'manual-surveillance', 'manual'),
  ('manual.flat_indemnity',
   '{"cents":4000,"label":"Indemnité forfaitaire pour frais de recouvrement"}'::jsonb,
   'manual-surveillance', 'manual'),
  ('manual.legal_interest_rate',
   '{"annual_percent":3.72,"note":"Taux d intérêt légal indicatif — remplacer dès source officielle branchée"}'::jsonb,
   'manual-surveillance', 'manual'),
  ('manual.einvoicing_calendar',
   '{"facturx_mandatory_from":"2027-09-01","scope":"micro-entrepreneurs émission","note":"Calendrier facturation électronique"}'::jsonb,
   'manual-surveillance', 'manual'),
  ('manual.mentions',
   '{"franchise_293b":"TVA non applicable, art. 293 B du CGI","late_penalty_template":"Pénalités de retard au taux annuel calculé selon la réglementation en vigueur.","flat_indemnity_40e":"Indemnité forfaitaire pour frais de recouvrement : 40 € (art. L441-10 et D441-5 du code de commerce).","payment_terms_default":"Paiement à réception de facture.","quote_validity_note":"Devis valable 30 jours sauf mention contraire."}'::jsonb,
   'manual-surveillance', 'manual')
) AS v(key, value, source, status)
WHERE NOT EXISTS (SELECT 1 FROM legal_values lv WHERE lv.key = v.key);


-- Projection initiale dans legal_variables (dernière valeur manuelle).
INSERT INTO legal_variables (key, value, source, fetched_at)
SELECT DISTINCT ON (key) key, value, source, fetched_at
FROM legal_values
ORDER BY key, fetched_at DESC
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value, source = EXCLUDED.source, fetched_at = EXCLUDED.fetched_at, updated_at = NOW();
