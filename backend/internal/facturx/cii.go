package facturx

import "encoding/xml"

// Structures CII D16B. L'ordre des champs est celui du schéma
// (une séquence XML différente est refusée par le validateur Factur-X).

type invoice struct {
	XMLName  xml.Name    `xml:"rsm:CrossIndustryInvoice"`
	XmlnsRSM string      `xml:"xmlns:rsm,attr"`
	XmlnsQDT string      `xml:"xmlns:qdt,attr"`
	XmlnsRAM string      `xml:"xmlns:ram,attr"`
	XmlnsUDT string      `xml:"xmlns:udt,attr"`
	Context  docContext  `xml:"rsm:ExchangedDocumentContext"`
	Document exchanged   `xml:"rsm:ExchangedDocument"`
	Trade    transaction `xml:"rsm:SupplyChainTradeTransaction"`
}

type docContext struct {
	Guideline idBlock `xml:"ram:GuidelineSpecifiedDocumentContextParameter"`
}

type idBlock struct {
	ID string `xml:"ram:ID"`
}

type exchanged struct {
	ID       string   `xml:"ram:ID"`
	TypeCode string   `xml:"ram:TypeCode"`
	Issue    dateTime `xml:"ram:IssueDateTime"`
	Notes    []note   `xml:"ram:IncludedNote,omitempty"`
}

type note struct {
	Content string `xml:"ram:Content"`
}

type dateTime struct {
	Value dateTimeString `xml:"udt:DateTimeString"`
}

type dateTimeString struct {
	Format string `xml:"format,attr"`
	Value  string `xml:",chardata"`
}

type transaction struct {
	Lines      []lineItem `xml:"ram:IncludedSupplyChainTradeLineItem"`
	Agreement  agreement  `xml:"ram:ApplicableHeaderTradeAgreement"`
	Delivery   delivery   `xml:"ram:ApplicableHeaderTradeDelivery"`
	Settlement settlement `xml:"ram:ApplicableHeaderTradeSettlement"`
}

type lineItem struct {
	Doc      lineDoc      `xml:"ram:AssociatedDocumentLineDocument"`
	Product  product      `xml:"ram:SpecifiedTradeProduct"`
	Agree    lineAgree    `xml:"ram:SpecifiedLineTradeAgreement"`
	Delivery lineDelivery `xml:"ram:SpecifiedLineTradeDelivery"`
	Settle   lineSettle   `xml:"ram:SpecifiedLineTradeSettlement"`
}

type lineDoc struct {
	LineID string `xml:"ram:LineID"`
}

type product struct {
	Name string `xml:"ram:Name"`
}

type lineAgree struct {
	Gross price `xml:"ram:GrossPriceProductTradePrice"`
	Net   price `xml:"ram:NetPriceProductTradePrice"`
}

type price struct {
	ChargeAmount string     `xml:"ram:ChargeAmount"`
	Allowance    *allowance `xml:"ram:AppliedTradeAllowanceCharge,omitempty"`
}

type allowance struct {
	ChargeIndicator indicator `xml:"ram:ChargeIndicator"`
	ActualAmount    string    `xml:"ram:ActualAmount"`
	Reason          string    `xml:"ram:Reason,omitempty"`
}

type indicator struct {
	Value bool `xml:"udt:Indicator"`
}

type lineDelivery struct {
	Billed quantity `xml:"ram:BilledQuantity"`
}

type quantity struct {
	UnitCode string `xml:"unitCode,attr"`
	Value    string `xml:",chardata"`
}

type lineSettle struct {
	Tax       tradeTax `xml:"ram:ApplicableTradeTax"`
	LineTotal lineSum  `xml:"ram:SpecifiedTradeSettlementLineMonetarySummation"`
}

type lineSum struct {
	LineTotalAmount string `xml:"ram:LineTotalAmount"`
}

type agreement struct {
	Seller party `xml:"ram:SellerTradeParty"`
	Buyer  party `xml:"ram:BuyerTradeParty"`
}

type party struct {
	GlobalID *schemeID `xml:"ram:GlobalID,omitempty"`
	Name     string    `xml:"ram:Name"`
	Legal    *legalOrg `xml:"ram:SpecifiedLegalOrganization,omitempty"`
	Contact  *contact  `xml:"ram:DefinedTradeContact,omitempty"`
	Address  address   `xml:"ram:PostalTradeAddress"`
	TaxReg   *taxReg   `xml:"ram:SpecifiedTaxRegistration,omitempty"`
}

type schemeID struct {
	Scheme string `xml:"schemeID,attr"`
	Value  string `xml:",chardata"`
}

type legalOrg struct {
	ID      schemeID `xml:"ram:ID"`
	Trading string   `xml:"ram:TradingBusinessName,omitempty"`
}

type contact struct {
	PersonName string  `xml:"ram:PersonName"`
	Phone      *number `xml:"ram:TelephoneUniversalCommunication,omitempty"`
	Email      *uri    `xml:"ram:EmailURIUniversalCommunication,omitempty"`
}

type number struct {
	CompleteNumber string `xml:"ram:CompleteNumber"`
}

type uri struct {
	URIID string `xml:"ram:URIID"`
}

type address struct {
	Postcode string `xml:"ram:PostcodeCode,omitempty"`
	LineOne  string `xml:"ram:LineOne,omitempty"`
	LineTwo  string `xml:"ram:LineTwo,omitempty"`
	City     string `xml:"ram:CityName,omitempty"`
	Country  string `xml:"ram:CountryID"`
}

type taxReg struct {
	ID schemeID `xml:"ram:ID"`
}

type delivery struct {
	Event *deliveryEvent `xml:"ram:ActualDeliverySupplyChainEvent,omitempty"`
}

type deliveryEvent struct {
	Occurrence dateTime `xml:"ram:OccurrenceDateTime"`
}

type settlement struct {
	Currency     string        `xml:"ram:InvoiceCurrencyCode"`
	PaymentMeans *paymentMeans `xml:"ram:SpecifiedTradeSettlementPaymentMeans,omitempty"`
	Taxes        []tradeTax    `xml:"ram:ApplicableTradeTax"`
	PaymentTerms *payTerms     `xml:"ram:SpecifiedTradePaymentTerms,omitempty"`
	Sum          summation     `xml:"ram:SpecifiedTradeSettlementHeaderMonetarySummation"`
}

type paymentMeans struct {
	TypeCode string   `xml:"ram:TypeCode"`
	Account  *account `xml:"ram:PayeePartyCreditorFinancialAccount,omitempty"`
}

type account struct {
	IBAN string `xml:"ram:IBANID"`
}

type tradeTax struct {
	CalculatedAmount      string `xml:"ram:CalculatedAmount,omitempty"`
	TypeCode              string `xml:"ram:TypeCode"`
	ExemptionReason       string `xml:"ram:ExemptionReason,omitempty"`
	BasisAmount           string `xml:"ram:BasisAmount,omitempty"`
	CategoryCode          string `xml:"ram:CategoryCode"`
	RateApplicablePercent string `xml:"ram:RateApplicablePercent"`
}

type payTerms struct {
	Description string    `xml:"ram:Description,omitempty"`
	Due         *dateTime `xml:"ram:DueDateDateTime,omitempty"`
}

type summation struct {
	LineTotal  string         `xml:"ram:LineTotalAmount"`
	TaxBasis   string         `xml:"ram:TaxBasisTotalAmount"`
	TaxTotal   currencyAmount `xml:"ram:TaxTotalAmount"`
	GrandTotal string         `xml:"ram:GrandTotalAmount"`
	DuePayable string         `xml:"ram:DuePayableAmount"`
}

type currencyAmount struct {
	Currency string `xml:"currencyID,attr"`
	Value    string `xml:",chardata"`
}
