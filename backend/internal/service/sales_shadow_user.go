package service

import "strings"

const (
	// SalesShadowUserEmailSuffix is the synthetic email suffix used for internally created sales users.
	SalesShadowUserEmailSuffix = "@sales.local"
	// SalesShadowUserNotesMarker is the marker written into notes for internally created sales users.
	SalesShadowUserNotesMarker = "created by sales partner"
)

func IsSalesShadowUserEmail(email string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(email)), SalesShadowUserEmailSuffix)
}

func IsSalesShadowUserNotes(notes string) bool {
	return strings.Contains(strings.ToLower(notes), SalesShadowUserNotesMarker)
}
