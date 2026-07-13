package metaconvapi

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

// RawUserData contains optional user identifiers and context for event tracking
type RawUserData struct {
	UserID       *string `json:"user_id,omitempty"`
	Email        *string `json:"email,omitempty"`
	Phone        *string `json:"phone,omitempty"`
	ClientIP     string  `json:"client_ip,omitempty"`
	UserAgent    string  `json:"user_agent,omitempty"`
	ConsentAvail bool    `json:"consent_avail,omitempty"`
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
