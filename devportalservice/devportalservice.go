package devportalservice

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-utils/log"
)

// portalData ...
type portalData struct {
	AppleID              string              `json:"apple_id"`
	Password             string              `json:"password"`
	ConnectionExpiryDate string              `json:"connection_expiry_date"`
	SessionCookies       map[string][]cookie `json:"session_cookies"`
}

// cookie ...
type cookie struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Value     string `json:"value"`
	Domain    string `json:"domain"`
	Secure    bool   `json:"secure"`
	Expires   string `json:"expires,omitempty"`
	MaxAge    int    `json:"max_age,omitempty"`
	Httponly  bool   `json:"httponly"`
	ForDomain *bool  `json:"for_domain,omitempty"`
}

const cookieTemplate = `- !ruby/object:HTTP::Cookie
  name: <NAME>
  value: <VALUE>
  domain: <DOMAIN>
  for_domain: <FOR_DOMAIN>
  path: "<PATH>"
`

// SessionData will fetch the session from Bitrise for the connected Apple developer account
// If the BITRISE_PORTAL_DATA_JSON is provided (for debug purposes) it will use that instead.
func SessionData() (string, error) {
	p, err := getDeveloperPortalData(os.Getenv("BITRISE_BUILD_URL"), os.Getenv("BITRISE_BUILD_API_TOKEN"))
	if err != nil {
		return "", err
	}

	cookies := convertDesCookie(p.SessionCookies["https://idmsa.apple.com"])
	session := strings.Join(cookies, "")
	return session, nil
}

func getDeveloperPortalData(buildURL, buildAPIToken string) (portalData, error) {
	var p portalData

	j, exists := os.LookupEnv("BITRISE_PORTAL_DATA_JSON")
	if exists && j != "" {
		return p, json.Unmarshal([]byte(j), &p)
	}

	if buildURL == "" {
		return portalData{}, fmt.Errorf("BITRISE_BUILD_URL env is not exported")
	}

	if buildAPIToken == "" {
		return portalData{}, fmt.Errorf("BITRISE_BUILD_API_TOKEN env is not exported")
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/apple_developer_portal_data.json", buildURL), nil)
	if err != nil {
		return portalData{}, err
	}

	req.Header.Add("BUILD_API_TOKEN", buildAPIToken)

	if _, err := performRequest(req, &p); err != nil {
		return portalData{}, fmt.Errorf("Falied to fetch portal data from Bitrise, error: %s", err)
	}
	return p, nil
}

func convertDesCookie(cookies []cookie) []string {
	var convertedCookies []string
	for _, c := range cookies {
		if convertedCookies == nil {
			convertedCookies = append(convertedCookies, "---"+"\n")
		}

		if c.ForDomain == nil {
			b := true
			c.ForDomain = &b
		}

		convertedCookie := strings.Replace(cookieTemplate, "<NAME>", c.Name, 1)
		convertedCookie = strings.Replace(convertedCookie, "<VALUE>", c.Value, 1)
		convertedCookie = strings.Replace(convertedCookie, "<DOMAIN>", c.Domain, 1)
		convertedCookie = strings.Replace(convertedCookie, "<FOR_DOMAIN>", strconv.FormatBool(*c.ForDomain), 1)
		convertedCookie = strings.Replace(convertedCookie, "<PATH>", c.Path, 1)

		convertedCookies = append(convertedCookies, convertedCookie+"\n")
	}

	return convertedCookies
}

func performRequest(req *http.Request, requestResponse interface{}) ([]byte, error) {
	client := http.Client{}
	response, err := client.Do(req)
	if err != nil {
		// On error, any Response can be ignored
		return nil, fmt.Errorf("failed to perform request, error: %s", err)
	}

	// The client must close the response body when finished with it
	defer func() {
		if cerr := response.Body.Close(); cerr != nil {
			log.Warnf("Failed to close response body, error: %s", cerr)
		}
	}()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body, error: %s", err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode > http.StatusMultipleChoices {
		return nil, fmt.Errorf("Response status: %d - Body: %s", response.StatusCode, string(body))
	}

	// Parse JSON body
	if requestResponse != nil {
		if err := json.Unmarshal([]byte(body), &requestResponse); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response (%s), error: %s", body, err)
		}
	}
	return body, nil
}

func main() {
}
