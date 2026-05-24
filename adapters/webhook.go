package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"prakhar-website-backend/config"
)

func SignupEmail(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, `{"error":"failed to read body"}`, http.StatusBadRequest)
			return
		}

		if err := verifySvix(
			cfg.ClerkWebhookSecret,
			r.Header.Get("svix-id"),
			r.Header.Get("svix-timestamp"),
			r.Header.Get("svix-signature"),
			body,
		); err != nil {
			http.Error(w, `{"error":"invalid webhook signature"}`, http.StatusBadRequest)
			return
		}

		var evt struct {
			Type string `json:"type"`
			Data struct {
				EmailAddresses []struct {
					EmailAddress string `json:"email_address"`
				} `json:"email_addresses"`
				FirstName string `json:"first_name"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &evt); err != nil {
			http.Error(w, `{"error":"failed to parse webhook"}`, http.StatusBadRequest)
			return
		}

		if evt.Type != "user.created" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"message": "skipping non-user.created event"})
			return
		}

		if len(evt.Data.EmailAddresses) == 0 {
			http.Error(w, `{"error":"no email in payload"}`, http.StatusBadRequest)
			return
		}

		firstName := evt.Data.FirstName
		if firstName == "" {
			firstName = "there"
		}

		if err := sendWelcomeEmail(cfg.SendgridAPIKey, evt.Data.EmailAddresses[0].EmailAddress, firstName); err != nil {
			http.Error(w, `{"error":"failed to send email"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}
}

func verifySvix(secret, msgID, msgTimestamp, msgSignatures string, body []byte) error {
	ts, err := strconv.ParseInt(msgTimestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp")
	}
	if diff := time.Now().Unix() - ts; diff < -300 || diff > 300 {
		return fmt.Errorf("timestamp out of tolerance")
	}

	secretBytes, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	if err != nil {
		return fmt.Errorf("invalid secret format")
	}

	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(msgID + "." + msgTimestamp + "."))
	mac.Write(body)
	computed := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	for sig := range strings.SplitSeq(msgSignatures, " ") {
		parts := strings.SplitN(sig, ",", 2)
		if len(parts) == 2 && parts[0] == "v1" && hmac.Equal([]byte(parts[1]), []byte(computed)) {
			return nil
		}
	}
	return fmt.Errorf("signature mismatch")
}

func sendWelcomeEmail(apiKey, to, firstName string) error {
	payload := map[string]any{
		"personalizations": []map[string]any{
			{"to": []map[string]string{{"email": to}}},
		},
		"from":    map[string]string{"email": "prakhar@em101.prakhargaming.com"},
		"subject": "Thanks for Signing Up with prakhargaming.com",
		"content": []map[string]string{
			{
				"type":  "text/plain",
				"value": fmt.Sprintf("Hi %s,\n\nI'm Prakhar! I'm really glad you signed up! It means a lot. There's not much going on here as of yet but stay tuned!\n\nBest,\nPrakhar Gaming", firstName),
			},
			{
				"type": "text/html",
				"value": fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Welcome Email</title>
</head>
<body style="margin:0;padding:0;font-family:Arial,sans-serif;background-color:#ffffff;color:#000000;">
  <table role="presentation" style="width:100%;border-spacing:0;margin:0;padding:0;background-color:#ffffff;">
    <tr>
      <td align="center" style="padding:20px;">
        <table role="presentation" style="max-width:600px;width:100%;border-spacing:0;text-align:left;background-color:#ffffff;border:1px solid #e0e0e0;">
          <tr>
            <td style="padding:20px;font-size:16px;line-height:1.5;color:#000000;">
              <h1 style="font-size:24px;margin:0 0 20px;color:#000000;">Hi %s,</h1>
              <p style="margin:0 0 20px;color:#000000;">I'm really glad you signed up! It means a lot. There's not much going on here as of yet, but stay tuned for updates and exciting content in the future!</p>
              <p style="margin:0 0 10px;font-weight:bold;color:#000000;">Best,</p>
              <p style="margin:0;font-weight:bold;color:#000000;">Prakhar Gaming</p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, firstName),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("sendgrid returned status %d", resp.StatusCode)
	}
	return nil
}
