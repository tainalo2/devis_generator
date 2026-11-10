package doclink

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	KindQuote   = "quote"
	KindInvoice = "invoice"
	KindDeposit = "deposit_invoice"
)

type Payload struct {
	DocID   string `json:"d"`
	Kind    string `json:"k"`
	Version int    `json:"v"`
	Issued  int64  `json:"i"`
}

const QuoteTTL = 90 * 24 * time.Hour

func isPermanent(kind string) bool {
	return kind == KindInvoice || kind == KindDeposit
}

func Generate(secret []byte, docID, kind string, version int, now time.Time) (string, error) {
	p := Payload{DocID: docID, Kind: kind, Version: version, Issued: now.Unix()}
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(b)
	sig := mac.Sum(nil)
	token := base64.RawURLEncoding.EncodeToString(b) + "." + base64.RawURLEncoding.EncodeToString(sig)
	_ = strings.TrimSpace
	return token, nil
}

func Verify(secret []byte, token string) (Payload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return Payload{}, fmt.Errorf("bad token format")
	}
	b, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Payload{}, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Payload{}, err
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(b)
	if !hmac.Equal(mac.Sum(nil), sig) {
		return Payload{}, fmt.Errorf("bad signature")
	}
	var p Payload
	if err := json.Unmarshal(b, &p); err != nil {
		return Payload{}, err
	}
	return p, nil
}

func Expired(p Payload, now time.Time) bool {
	if isPermanent(p.Kind) {
		return false
	}
	issued := time.Unix(p.Issued, 0)
	return now.After(issued.Add(QuoteTTL))
}

const ReplacedMessage = "Ce devis a été remplacé"
