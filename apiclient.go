package conversions

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
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
// MetaConversionEvent represents a single event to send to Meta
type MetaConversionEvent struct {
	EventName             string         `json:"event_name"`
	EventTime             int64          `json:"event_time"`
	ActionSource          string         `json:"action_source"`
	EventID               string         `json:"event_id,omitempty"`
	UserData              MetaUserData   `json:"user_data"`
	CustomData            MetaCustomData `json:"custom_data,omitempty"`
	DataProcessingOptions []string       `json:"data_processing_options,omitempty"`
}

// MetaUserData contains hashed user identifiers
type MetaUserData struct {
	Em              string `json:"em,omitempty"` // hashed email
	Ph              string `json:"ph,omitempty"` // hashed phone
	ClientIPAddress string `json:"client_ip_address,omitempty"`
	ClientUserAgent string `json:"client_user_agent,omitempty"`
	Fbp             string `json:"fbp,omitempty"`
	Fbc             string `json:"fbc,omitempty"`
}

// MetaCustomData contains event-specific parameters
type MetaCustomData struct {
	Currency string                 `json:"currency,omitempty"`
	Value    float64                `json:"value,omitempty"`
	Contents []MetaContent          `json:"contents,omitempty"`
	OrderID  string                 `json:"order_id,omitempty"`
	Extra    map[string]interface{} `json:"-"`
}

// MetaContent represents an item in the cart/purchase
type MetaContent struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price,omitempty"`
}

// MetaCAPIRequest is the wrapper for the API call
type MetaCAPIRequest struct {
	Data []MetaConversionEvent `json:"data"`
}

// MetaCAPIResponse is the API response structure
type MetaCAPIResponse struct {
	EventsReceived int `json:"events_received"`
	Messages       []struct {
		Level   string `json:"level"`
		Message string `json:"message"`
	} `json:"messages,omitempty"`
}

// MetaCAClient handles communication with Meta Conversions API
type MetaCAClient struct {
	PixelID     string
	AccessToken string
	HTTPClient  *http.Client
}

// NewMetaCAClient creates a new client instance
func NewMetaCAClient(pixelID, accessToken string) *MetaCAClient {
	return &MetaCAClient{
		PixelID:     pixelID,
		AccessToken: accessToken,
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *MetaCAClient) SendTestEvent(userEmail string, userPhone string, req http.Request) error {
	//TODO: event firing shouldn't fail, only log error and return void
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return err
	}
	event := MetaConversionEvent{
		EventName:    "TEST75302",
		EventTime:    time.Now().Unix(),
		ActionSource: "website",
		EventID:      "TEST75302",
		UserData: MetaUserData{
			Em:              HashString(userEmail),
			Ph:              HashString(userPhone),
			ClientIPAddress: ip,
			ClientUserAgent: req.UserAgent(),
		},
	}

	// Send event
	response, err := c.SendEvents([]MetaConversionEvent{event})
	if err != nil {
		log.Printf("Failed to send user action to Meta, error: %v\n", err)
		return err
	}

	log.Printf("Events received by Meta: %d\n", response.EventsReceived)
	return err
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
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var result MetaCAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
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

// BuildAddToCartEvent constructs an AddToCart event
func BuildAddToCartEvent(
	userID, email, phone, ip, userAgent string,
	productID string, quantity int, price float64, currency string,
) MetaConversionEvent {
	return MetaConversionEvent{
		EventName:    "AddToCart",
		EventTime:    time.Now().Unix(),
		ActionSource: "mobile_app",
		EventID:      fmt.Sprintf("addtocart_%d_%s", time.Now().UnixNano(), userID),
		UserData: MetaUserData{
			Em:              HashString(email),
			Ph:              HashString(phone),
			ClientIPAddress: ip,
			ClientUserAgent: userAgent,
		},
		CustomData: MetaCustomData{
			Currency: currency,
			Value:    price * float64(quantity),
			Contents: []MetaContent{
				{ID: productID, Quantity: quantity, Price: price},
			},
		},
	}
}

// BuildPurchaseEvent constructs a Purchase event
func BuildPurchaseEvent(
	userID, email, phone, ip, userAgent, orderID string,
	items []MetaContent, total float64, currency string,
) MetaConversionEvent {
	return MetaConversionEvent{
		EventName:    "Purchase",
		EventTime:    time.Now().Unix(),
		ActionSource: "mobile_app",
		EventID:      fmt.Sprintf("purchase_%s", orderID),
		UserData: MetaUserData{
			Em:              HashString(email),
			Ph:              HashString(phone),
			ClientIPAddress: ip,
			ClientUserAgent: userAgent,
		},
		CustomData: MetaCustomData{
			Currency: currency,
			Value:    total,
			OrderID:  orderID,
			Contents: items,
		},
	}
}

// BuildPageViewEvent constructs a PageView event
func BuildPageViewEvent(
	userID, email, ip, userAgent, pageURL string,
) MetaConversionEvent {
	return MetaConversionEvent{
		EventName:    "PageView",
		EventTime:    time.Now().Unix(),
		ActionSource: "mobile_app",
		EventID:      fmt.Sprintf("pageview_%d_%s", time.Now().UnixNano(), userID),
		UserData: MetaUserData{
			Em:              HashString(email),
			ClientIPAddress: ip,
			ClientUserAgent: userAgent,
		},
		CustomData: MetaCustomData{
			Extra: map[string]interface{}{
				"page_url": pageURL,
			},
		},
	}
}

// Example usage
func (c *MetaCAClient) test() {
	// Build event
	event := BuildAddToCartEvent(
		"user123",          // userID
		"user@example.com", // email
		"+20123456789",     // phone
		"192.168.1.1",      // IP
		"Mozilla/5.0...",   // UserAgent
		"product_456",      // productID
		2,                  // quantity
		29.99,              // price
		"USD",              // currency
	)

	// Send event
	response, err := c.SendEvents([]MetaConversionEvent{event})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Events received: %d\n", response.EventsReceived)
}
