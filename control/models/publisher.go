package models

import (
	"net/http"

	"bitbucket.org/tapgerine/pmp/control/database"
)

type Publisher struct {
	ID                 uint64 `gorm:"primary_key"`
	Name               string `gorm:"size:255" valid:"required"`
	CompanySite        string `valid:"url"`
	Email              string `valid:"email"`
	CustomID           uint64 `valid:"required"`
	UserID             uint64
	TargetingID        string `gorm:"size:8"`
	TargetingLink      string
	TargetingPrice     float64
	Comments           string
	BeneficiaryName    string
	BeneficiaryAccount string
	BeneficiaryAddress string
	Iban               string
	BankName           string
	BankAddress        string
	Routing            string
	Swift              string
	PaypalEmail        string
	WebMoneyWmz        string
	EpaymentsID        string
	Address            string
	Phone              string

	AdTagPublisher []AdTagPublisher
	User           User
	Links          []PublisherLink
}

func (pub *Publisher) GetByID(id interface{}) {
	pub.ID = id.(uint64)
	database.Postgres.Preload("User").Preload("Links").First(pub)
}

func (pub *Publisher) Save() (bool, []string) {
	errors := database.Postgres.Save(pub).GetErrors()
	var messages []string

	if len(errors) > 0 {
		for _, error := range errors {
			messages = append(messages, getError(error).Detail)
		}
	}

	return len(errors) == 0, messages
}

func (pub *Publisher) Create() bool {
	return len(database.Postgres.Create(pub).GetErrors()) == 0
}

func (pub *Publisher) CheckIfTargetingIDExists() {
	if pub.TargetingID == "" {
		pub.TargetingID = randStringBytes(8)
		pub.Save()
	}
}

func (pub *Publisher) PopulateData(r *http.Request) {
	pub.Name = r.Form.Get("name")
	pub.CompanySite = r.Form.Get("company_site")
	pub.Email = r.Form.Get("email")
	pub.Comments = r.Form.Get("comments")
	pub.CustomID, _ = getUintValueFromForm(r, "custom_id", true)
	pub.TargetingPrice, _ = getFloatValueFromForm(r, "targeting_price", true)
	pub.BeneficiaryName = r.Form.Get("beneficiary_name")
	pub.BeneficiaryAccount = r.Form.Get("beneficiary_account")
	pub.BeneficiaryAddress = r.Form.Get("beneficiary_address")
	pub.Iban = r.Form.Get("iban")
	pub.BankName = r.Form.Get("bank_name")
	pub.BankAddress = r.Form.Get("bank_address")
	pub.Routing = r.Form.Get("routing")
	pub.Swift = r.Form.Get("swift")
	pub.PaypalEmail = r.Form.Get("paypal_email")
	pub.WebMoneyWmz = r.Form.Get("web_money_wmz")
	pub.EpaymentsID = r.Form.Get("epayments_id")
	pub.Phone = r.Form.Get("phone")
	pub.Address = r.Form.Get("address")
}

func (pub *Publisher) UpdateData(r *http.Request) {
	pub.Name = r.Form.Get("name")
	pub.CompanySite = r.Form.Get("company_site")
	pub.Email = r.Form.Get("email")
	pub.Comments = r.Form.Get("comments")
	pub.CustomID, _ = getUintValueFromForm(r, "custom_id", true)
	pub.TargetingPrice, _ = getFloatValueFromForm(r, "targeting_price", true)
	pub.BeneficiaryName = r.Form.Get("beneficiary_name")
	pub.BeneficiaryAccount = r.Form.Get("beneficiary_account")
	pub.BeneficiaryAddress = r.Form.Get("beneficiary_address")
	pub.Iban = r.Form.Get("iban")
	pub.BankName = r.Form.Get("bank_name")
	pub.BankAddress = r.Form.Get("bank_address")
	pub.Routing = r.Form.Get("routing")
	pub.Swift = r.Form.Get("swift")
	pub.PaypalEmail = r.Form.Get("paypal_email")
	pub.WebMoneyWmz = r.Form.Get("web_money_wmz")
	pub.EpaymentsID = r.Form.Get("epayments_id")
	pub.Phone = r.Form.Get("phone")
	pub.Address = r.Form.Get("address")
}

func (pub *Publisher) GetByIDFromRequest(r *http.Request) {
	ID, adTagIDErr := getUintIDFromRequest(r, "publisher_id")

	if adTagIDErr != nil {
		return
	}
	pub.GetByID(ID)
}
