package model

import "time"

// Device is the resource exposed by the devices API.
type Device struct {
	ID             uint32    `json:"id"`
	Name           string    `json:"name"`
	Serial         string    `json:"serial"`
	Manufacturer   string    `json:"manufacturer"`
	Type           string    `json:"type"`
	WarrantyMonths *int      `json:"warranty_months"`
	Comment        *string   `json:"comment"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CreateDeviceRequest contains the fields accepted when creating a device.
type CreateDeviceRequest struct {
	Name           string  `json:"name"`
	Serial         string  `json:"serial"`
	Manufacturer   string  `json:"manufacturer"`
	Type           string  `json:"type"`
	WarrantyMonths *int    `json:"warranty_months"`
	Comment        *string `json:"comment"`
}
