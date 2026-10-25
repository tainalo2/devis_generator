package legal

// Canonical keys stored in legal_values / legal_variables.
const (
	KeyMentions             = "mentions.v1"
	KeyManualMentions       = "manual.mentions"
	KeyFranchiseThresholds  = "manual.franchise_thresholds"
	KeyMicroThresholds      = "manual.micro_thresholds"
	KeyFlatIndemnity        = "manual.flat_indemnity"
	KeyLegalInterestRate    = "manual.legal_interest_rate"
	KeyEinvoicingCalendar   = "manual.einvoicing_calendar"
	KeyTEDBVATRatesFR       = "tedb.vat_rates.fr"
	KeyBCEKeyRate           = "bce.key_rate"
	KeyLatePenaltyEffective = "derived.late_penalty_rate"
)

// Status values for append-only history.
const (
	StatusVerified = "verified"
	StatusStale    = "stale"
	StatusError    = "error"
	StatusManual   = "manual"
)
