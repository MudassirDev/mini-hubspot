package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type MailtrapEmailSender struct {
	APIKey      string
	FromName    string
	FromEmail   string
	TemplateID  string
	CompanyInfo map[string]string
}

type MailtrapPayload struct {
	From struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"from"`
	To []struct {
		Email string `json:"email"`
	} `json:"to"`
	TemplateUUID      string         `json:"template_uuid"`
	TemplateVariables map[string]any `json:"template_variables"`
}

func NewMailtrapSender() *MailtrapEmailSender {
	return &MailtrapEmailSender{
		APIKey:     os.Getenv("MAILTRAP_API_KEY"),
		FromEmail:  os.Getenv("MAILTRAP_FROM_EMAIL"),
		FromName:   os.Getenv("MAILTRAP_FROM_NAME"),
		TemplateID: os.Getenv("MAILTRAP_TEMPLATE_ID"),
		CompanyInfo: map[string]string{
			"company_info_name":     os.Getenv("MAILTRAP_COMPANY_NAME"),
			"company_info_address":  os.Getenv("MAILTRAP_COMPANY_ADDRESS"),
			"company_info_city":     os.Getenv("MAILTRAP_COMPANY_CITY"),
			"company_info_zip_code": os.Getenv("MAILTRAP_COMPANY_ZIP"),
			"company_info_country":  os.Getenv("MAILTRAP_COMPANY_COUNTRY"),
		},
	}
}

func (m *MailtrapEmailSender) SendVerificationEmail(toEmail, name, verifyLink string) error {
	payload := MailtrapPayload{}
	payload.From.Email = m.FromEmail
	payload.From.Name = m.FromName
	payload.To = []struct {
		Email string `json:"email"`
	}{{Email: toEmail}}
	payload.TemplateUUID = m.TemplateID

	payload.TemplateVariables = map[string]any{
		"name":                    name,
		"email_verification_link": verifyLink,
	}

	// Add company info
	for k, v := range m.CompanyInfo {
		payload.TemplateVariables[k] = v
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://send.api.mailtrap.io/api/send", bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Api-Token", m.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Mailtrap API error: %s", resp.Status)
	}

	return nil
}
