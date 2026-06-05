package tools

import "strings"

type Product struct {
	SKU          string `json:"sku"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	SellingAngle string `json:"sellingAngle"`
}

var ashleyCatalog = []Product{
	{"ASH-SOFA-001", "Maeford Sectional", "sofa", "合家欢的客厅中心；一家人团圆、和解的温情场景"},
	{"ASH-BED-001", "Realyn Queen Bed", "bed", "重新开始 / 爆改新居后的清晨醒来场景"},
	{"ASH-DINE-001", "Haddigan Dining Set", "dining", "庆功宴与餐桌上的摊牌对峙场景"},
	{"ASH-RECL-001", "Boxberg Recliner", "recliner", "和解谈心、卸下心防的暖心时刻"},
	{"ASH-DESK-001", "Camiburg Home Office Desk", "office", "草根逆袭、在家创业打拼的奋斗蒙太奇"},
	{"ASH-OUT-001", "Clare View Outdoor Set", "outdoor", "梦想豪宅揭晓 / 阶层跃升的高光时刻"},
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
