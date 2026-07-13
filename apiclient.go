package metaconvapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

/*
# Event samples:

## ViewContent

{
    "data": [
        {
            "event_name": "ViewContent",
            "event_time": 1783852813,
            "action_source": "website",
            "user_data": { ... },
            "custom_data": {
                "content_category": "Grocery",
                "content_name": "Lettuce"
            }
        }
    ]
}

## AddToCart

{
    "data": [
        {
            "event_name": "AddToCart",
            "event_time": 1783853035,
            "action_source": "website",
            "user_data": { ... },
            "custom_data": {
                "content_category": "Grocery",
                "content_name": "Lettuce"
            }
        }
    ]
}

## Purchase

{
    "data": [
        {
            "event_name": "Purchase",
            "event_time": 1783852813,
            "action_source": "website",
            "user_data": { ... },
            "custom_data": {
                "currency": "USD",
                "value": 142,
				"contents": [],
            },
        }
    ]
}

## Search

{
    "data": [
        {
            "event_name": "Search",
            "event_time": 1783853035,
            "action_source": "website",
            "user_data": { ... },
            "custom_data": {
                "search_string": "Shoes"
            }
        }
    ]
}


*/

// MetaCAClient handles communication with Meta Conversions API
type MetaCAClient struct {
	PixelID      string
	AccessToken  string
	ActionSource string
	Currency     string
	HTTPClient   *http.Client
}

// NewMetaCAClient creates a new client instance
func NewMetaCAClient(pixelID, accessToken, actionSource, currency string, timeout time.Duration) *MetaCAClient {
	return &MetaCAClient{
		PixelID:      pixelID,
		AccessToken:  accessToken,
		ActionSource: actionSource,
		Currency:     currency,
		HTTPClient:   &http.Client{Timeout: timeout},
	}
}

// SendEvents sends events to Meta Conversions API
func (c *MetaCAClient) SendEvents(events []MetaConversionEvent) (*MetaCAPIResponse, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("no events to send")
	}

	// Build request
	payload := MetaCAPIRequest{Data: events}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Meta API endpoint
	url := fmt.Sprintf("https://graph.facebook.com/v20.0/%s/events?access_token=%s",
		c.PixelID, c.AccessToken)

	// Send request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		log.Printf("HTTP request to Meta conversions API has failed: %v", err)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result MetaCAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("failed to parse response from Meta conversions API: %v", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Meta conversions API error, req payload: %s (status %d): %+v", string(jsonPayload), resp.StatusCode, result)
		return nil, fmt.Errorf("API error, req payload: %s (status %d): %+v", string(jsonPayload), resp.StatusCode, result)
	}

	return &result, nil
}

// HashString SHA256 hashes a string
func HashString(input string) string {
	if input == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// rawUserDataToMetaUserData converts RawUserData to MetaUserData, hashing email and phone
func rawUserDataToMetaUserData(raw RawUserData) MetaUserData {
	userData := MetaUserData{}

	if raw.Email != nil {
		userData.Em = HashString(*raw.Email)
	}
	if raw.Phone != nil {
		userData.Ph = HashString(*raw.Phone)
	}

	userData.ClientIPAddress = raw.ClientIP
	userData.ClientUserAgent = raw.UserAgent

	return userData
}

// generateEventID creates a unique event ID based on prefix and optional userID
func generateEventID(prefix, userID string) string {
	if userID != "" {
		return fmt.Sprintf("%s_%d_%s", prefix, time.Now().UnixNano(), userID)
	}
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// SendTestEvent sends a test event
func (c *MetaCAClient) SendTestEvent(userData RawUserData, eventId string) error {
	if !userData.ConsentAvail {
		return nil
	}

	event := MetaConversionEvent{
		EventName:    eventId,
		EventTime:    time.Now().Unix(),
		ActionSource: c.ActionSource,
		UserData:     rawUserDataToMetaUserData(userData),
	}

	// Send event
	_, err := c.SendEvents([]MetaConversionEvent{event})
	return err
}

// SendAddToCartEvent builds and sends a single AddToCart event
func (c *MetaCAClient) SendAddToCartEvent(
	userData RawUserData,
	productName string, quantity int,
) error {
	if !userData.ConsentAvail {
		return nil
	}

	var userID string
	if userData.UserID != nil {
		userID = *userData.UserID
	}

	event := MetaConversionEvent{
		EventName:    "AddToCart",
		EventTime:    time.Now().Unix(),
		ActionSource: c.ActionSource,
		EventID:      generateEventID("addtocart", userID),
		UserData:     rawUserDataToMetaUserData(userData),
		CustomData: MetaCustomData{
			Contents: []MetaContent{
				{Name: productName, Quantity: quantity},
			},
		},
	}

	_, err := c.SendEvents([]MetaConversionEvent{event})
	if err != nil {
		log.Printf("Failed to send AddToCart event to Meta, error: %v\n", err)
		return err
	}

	return nil
}

// SendPurchaseEvent builds and sends a single Purchase event
func (c *MetaCAClient) SendPurchaseEvent(
	userData RawUserData,
	orderID string,
	items []MetaContent, total float64,
) error {
	if !userData.ConsentAvail {
		return nil
	}

	event := MetaConversionEvent{
		EventName:    "Purchase",
		EventTime:    time.Now().Unix(),
		ActionSource: c.ActionSource,
		EventID:      fmt.Sprintf("purchase_%s", orderID),
		UserData:     rawUserDataToMetaUserData(userData),
		CustomData: MetaCustomData{
			Currency: c.Currency,
			Value:    total,
			OrderID:  orderID,
			Contents: items,
		},
	}

	_, err := c.SendEvents([]MetaConversionEvent{event})
	if err != nil {
		log.Printf("Failed to send Purchase event to Meta, error: %v\n", err)
		return err
	}

	return nil
}

// SendPageViewEvent builds and sends a single PageView event
func (c *MetaCAClient) SendPageViewEvent(
	userData RawUserData,
	pageURL string,
) error {
	if !userData.ConsentAvail {
		return nil
	}

	var userID string
	if userData.UserID != nil {
		userID = *userData.UserID
	}

	event := MetaConversionEvent{
		EventName:    "PageView",
		EventTime:    time.Now().Unix(),
		ActionSource: c.ActionSource,
		EventID:      generateEventID("pageview", userID),
		UserData:     rawUserDataToMetaUserData(userData),
		CustomData: MetaCustomData{
			Extra: map[string]interface{}{
				"page_url": pageURL,
			},
		},
	}

	_, err := c.SendEvents([]MetaConversionEvent{event})
	if err != nil {
		log.Printf("Failed to send PageView event to Meta, error: %v\n", err)
		return err
	}

	return nil
}

// SendViewContentEvent builds and sends a single ViewContent event
func (c *MetaCAClient) SendViewContentEvent(
	userData RawUserData,
	contentName string,
) error {
	if !userData.ConsentAvail {
		return nil
	}

	var userID string
	if userData.UserID != nil {
		userID = *userData.UserID
	}

	event := MetaConversionEvent{
		EventName:    "ViewContent",
		EventTime:    time.Now().Unix(),
		ActionSource: c.ActionSource,
		EventID:      generateEventID("viewcontent", userID),
		UserData:     rawUserDataToMetaUserData(userData),
		CustomData: MetaCustomData{
			Extra: map[string]interface{}{
				"content_name": contentName,
			},
		},
	}

	_, err := c.SendEvents([]MetaConversionEvent{event})
	if err != nil {
		log.Printf("Failed to send ViewContent event to Meta, error: %v\n", err)
		return err
	}

	return nil
}

// SendSearchEvent builds and sends a single Search event
func (c *MetaCAClient) SendSearchEvent(
	userData RawUserData,
	searchString string,
) error {
	if !userData.ConsentAvail {
		return nil
	}

	var userID string
	if userData.UserID != nil {
		userID = *userData.UserID
	}

	event := MetaConversionEvent{
		EventName:    "Search",
		EventTime:    time.Now().Unix(),
		ActionSource: c.ActionSource,
		EventID:      generateEventID("search", userID),
		UserData:     rawUserDataToMetaUserData(userData),
		CustomData: MetaCustomData{
			Extra: map[string]interface{}{
				"search_string": searchString,
			},
		},
	}

	_, err := c.SendEvents([]MetaConversionEvent{event})
	if err != nil {
		log.Printf("Failed to send Search event to Meta, error: %v\n", err)
		return err
	}

	return nil
}
