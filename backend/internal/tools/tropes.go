package tools

type Trope struct {
	Name       string `json:"name"`
	Hook       string `json:"hook"`
	WhyItWorks string `json:"whyItWorks"`
	HomeAngle  string `json:"homeAngle"` // how furniture earns screen time
}

var homeTropes = []Trope{
	{"家装改造逆袭", "被赶出家门的女主立誓爆改出租屋、翻身打脸", "向往式阶层跃升、爽点来得快", "每一次升级 = 一件新家具登场亮相"},
	{"离婚后爆改出租屋", "她拎着一只行李箱净身出户、重新开始", "独立爽感 + 一切归零的崭新起点", "把新居一件件添置布置 = 治愈系蜕变蒙太奇"},
	{"重生之打造梦想之家", "落魄装修工其实是隐藏的霸总 / 设计大佬", "反转驱动、身份揭晓", "豪宅大改造正好展示高端家具系列"},
	{"婆媳和解之家", "因房子翻脸的一家人最终破镜重圆", "情感爽点、温情治愈", "客厅 / 餐桌的合家欢场景承载品牌温度"},
	{"赘婿战神回归", "被看不起的赘婿亮明身份、买下整片豪宅", "扮猪吃虎、连环打脸", "梦想豪宅的揭晓时刻铺满成套家具"},
}

func GetWinningTropes(market, vertical string) []Trope {
	// MVP: single curated home vertical; market reserved for future expansion.
	return homeTropes
}
