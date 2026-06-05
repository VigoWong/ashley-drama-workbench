package tools

import "strings"

type Product struct {
	SKU          string `json:"sku"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	SellingAngle string `json:"sellingAngle"`
}

var ashleyCatalog = []Product{
	{"ASH-SOFA-001", "Maeford Sectional", "sofa", "Family gathering centerpiece; cozy reunion scenes"},
	{"ASH-BED-001", "Realyn Queen Bed", "bed", "Fresh-start / new-home morning scenes"},
	{"ASH-DINE-001", "Haddigan Dining Set", "dining", "Celebration & confrontation dinner scenes"},
	{"ASH-RECL-001", "Boxberg Recliner", "recliner", "Reconciliation / heart-to-heart moments"},
	{"ASH-DESK-001", "Camiburg Home Office Desk", "office", "Underdog-builds-business montage"},
	{"ASH-OUT-001", "Clare View Outdoor Set", "outdoor", "Dream-home reveal / status arc"},
}

func GetProductCatalog(category string) []Product {
	if category == "" {
		return ashleyCatalog
	}
	c := strings.ToLower(category)
	var out []Product
	for _, p := range ashleyCatalog {
		if p.Category == c {
			out = append(out, p)
		}
	}
	return out
}
