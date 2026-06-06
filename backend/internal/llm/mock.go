package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ashley/drama-workbench/internal/model"
)

type Mock struct{ fixtures map[string]string }

func NewMock() *Mock { return &Mock{fixtures: map[string]string{}} }

func (m *Mock) Register(stage, jsonOut string) { m.fixtures[stage] = jsonOut }

func (m *Mock) GenerateJSON(_ context.Context, stage, _ string, _ []model.Image, _ map[string]any) ([]byte, error) {
	v, ok := m.fixtures[stage]
	if !ok {
		return nil, fmt.Errorf("mock: no fixture registered for stage %q", stage)
	}
	return []byte(v), nil
}

// GenerateImage is a no-op for the Mock: it implements ImageProvider only to
// report that image generation is unavailable, so the no-key demo produces a
// complete text plan with an empty Visuals slice (no images).
func (m *Mock) GenerateImage(_ context.Context, _ string) ([]byte, string, error) {
	return nil, "", ErrImagesUnsupported
}

// DemoMock 返回一个为全部 7 个流水线阶段预注册了丰富、可信示例的 *Mock，这样在没有
// API Key 的情况下，服务/CLI 也能产出一份完整、合理的方案。题材是面向中国国内市场、
// 植入 Ashley（爱室丽）家具的「离婚后爆改出租屋逆袭」短剧，使用产品库中真实的 SKU。
// Mock 忽略 prompt，对每个阶段返回固定的示例，因此无论请求多少集，分集数组都是固定的
// ~12 集整季。
func DemoMock() *Mock {
	m := NewMock()
	m.Register("concept", `{
		"logline": "被丈夫和小三联手赶出家门、净身出户的女设计师，蜗居进一间破出租屋——却凭一双巧手把它爆改成全网爆款样板间，一边翻身打脸渣男，一边藏起自己豪门继承人的真实身份。",
		"theme": "自我价值、向死而生，以及一个完全属于自己的家",
		"audience": "抖音 / 红果短剧上 25-45 岁、爱看逆袭打脸 + 家装改造爽剧的女性观众",
		"tone": "高级质感、情绪浓烈、每集都有让人解气的反转",
		"payoffEngine": "阶层跃升式打脸逆袭：每一集女主都夺回一分主动权，并揭晓一个比对手所能负担的更美、更完整布置的居家空间。",
		"coreConflict": "净身出户、一无所有的她，必须一边重建生活与自己的家，一边藏住继承的家底，反手把背叛她的人一个个掀翻。",
		"tropesUsed": ["离婚后爆改出租屋", "家装改造逆袭", "重生之打造梦想之家"]
	}`)

	m.Register("bible", `{
		"title": "爆改出租屋后我成了豪门继承人",
		"genreTags": ["逆袭打脸", "家装改造", "都市情感", "扮猪吃虎"],
		"episodes": 12,
		"episodeSecs": 90,
		"totalRuntimeMin": 18,
		"platform": "抖音 / 红果短剧（竖屏 9:16）",
		"integrationThesis": "女主翻身的每一步都落在一个被彻底改造、布置完整的 Ashley 居家空间里——家具就是她地位不断攀升的可视化记分牌，因此品牌的揭晓时刻同时也是情绪爽点的落地时刻。"
	}`)

	m.Register("characters", `{"characters": [
		{
			"name": "苏念",
			"role": "protagonist",
			"bio": "天赋出众的室内设计师，为支持丈夫的公司放弃了自己的事业，却在变得「碍事」的那一刻被一脚踢开。",
			"arc": "从被羞辱、身无分文的弃妇，逆袭成把每个房间都改造成独立宣言的当红设计大佬。",
			"relationships": "陆既明名存实亡的妻子；已故地产大亨苏振邦秘而不宣的孙女。"
		},
		{
			"name": "陆既明",
			"role": "antagonist",
			"bio": "苏念那位攀附权贵的丈夫，在她继承家底被揭晓的前一晚，把她换成了更有钱的新欢。",
			"arc": "从志得意满、不可一世，到被苏念买下他所有过度举债的产业、当众破产。",
			"relationships": "苏念即将离婚的丈夫；江岸的生意对手。"
		},
		{
			"name": "江岸",
			"role": "love-interest",
			"bio": "有原则、有手艺的定制家具匠人，帮苏念布置她的第一个新家，一点点赢得她的信任。",
			"arc": "从戒备心重的独行者，成为苏念在生活与设计事业上的搭档。",
			"relationships": "苏念的合作伙伴，也是渐生情愫的爱人；陆既明的行业死敌。"
		}
	]}`)

	m.Register("episodes", episodesFixture())

	m.Register("placements", `{"placements": [
		{"episode": 1, "scene": "苏念被赶出家门，在空荡荡的出租屋地板上熬过第一夜", "productSku": "ASH-SOFA-001", "category": "sofa", "emotionalBeat": "空房间映衬下的人生谷底", "ctaTiming": "第 75 秒画面下方软性卡片"},
		{"episode": 2, "scene": "在爆改后的小出租屋里醒来的第一个清晨", "productSku": "ASH-BED-001", "category": "bed", "emotionalBeat": "重新开始的希望", "ctaTiming": "片尾购物卡片"},
		{"episode": 4, "scene": "苏念设下一桌饭局，当面摊牌陆既明的新欢", "productSku": "ASH-DINE-001", "category": "dining", "emotionalBeat": "餐桌上的摊牌与气场压制", "ctaTiming": "第 45 秒中插商品挂链"},
		{"episode": 6, "scene": "深夜与江岸推心置腹的和解谈心", "productSku": "ASH-RECL-001", "category": "recliner", "emotionalBeat": "暖意与信任的建立", "ctaTiming": "画面下方软性卡片"},
		{"episode": 8, "scene": "苏念在家中创立自己的设计工作室", "productSku": "ASH-DESK-001", "category": "office", "emotionalBeat": "草根逆袭、打造事业的劲头", "ctaTiming": "片尾购物卡片"},
		{"episode": 12, "scene": "苏念接管整片豪宅区，举办梦想之家揭晓派对", "productSku": "ASH-OUT-001", "category": "outdoor", "emotionalBeat": "扬眉吐气的阶层跃升爽点", "ctaTiming": "第 80 秒高光可购买片尾卡片"}
	]}`)

	m.Register("hero", `{"heroScenes": [
		{"episode": 12, "title": "梦想之家揭晓", "shots": [
			{"number": 1, "shotType": "WS", "action": "竖屏航拍式推进，掠过苏念新庄园里灯火通明的揭晓派对", "dialogue": ""},
			{"number": 2, "shotType": "MS", "action": "苏念走上露台，身后是整套 Clare View 户外家具，宾客一片惊呼", "dialogue": "欢迎来到你说我这辈子都得不到的家。"},
			{"number": 3, "shotType": "CU", "action": "陆既明认出苏念手中的房产证，脸色骤然垮掉", "dialogue": "这……全都被你买下来了？"},
			{"number": 4, "shotType": "POV", "action": "苏念的目光扫过布置完整的客厅，停在那张 Maeford 转角沙发上", "dialogue": ""},
			{"number": 5, "shotType": "MS", "action": "江岸握住苏念的手，镜头拉远，定格在灯火璀璨的房子上", "dialogue": "现在，这才真正是个家。"}
		]},
		{"episode": 4, "title": "摊牌饭局", "shots": [
			{"number": 1, "shotType": "WS", "action": "Haddigan 餐桌为一场暗流涌动的饭局摆好，烛光摇曳", "dialogue": ""},
			{"number": 2, "shotType": "CU", "action": "苏念不紧不慢地放下酒杯", "dialogue": "坐吧，我们有很多账要算。"},
			{"number": 3, "shotType": "MS", "action": "陆既明的新欢局促不安，苏念把一份文件夹推过桌面", "dialogue": "你知道你花的是谁的钱吗？"}
		]}
	]}`)

	m.Register("production_distribution", `{
		"production": {
			"format": "9:16 竖屏",
			"budgetTier": "中等品牌定制（以单一场景为主，3 天拍摄周期）",
			"shotCount": 240,
			"castSize": 6,
			"locations": ["破旧的小出租屋", "现代风设计工作室", "豪华揭晓庄园", "餐厅包间"],
			"furnitureProps": ["Maeford 转角沙发（ASH-SOFA-001）", "Realyn 大床（ASH-BED-001）", "Haddigan 餐桌套装（ASH-DINE-001）", "Boxberg 休闲躺椅（ASH-RECL-001）", "Camiburg 家用办公书桌（ASH-DESK-001）", "Clare View 户外家具套装（ASH-OUT-001）"]
		},
		"distribution": {
			"ctaCopy": "她从一无所有爆改出梦想之家。同款好物，上 Ashley 爱室丽一键带回家。",
			"linkPlacement": "片尾可购买卡片 + 置顶评论挂 SKU 链接",
			"hashtags": ["#短剧", "#逆袭打脸", "#爆改出租屋", "#家装改造", "#梦想之家", "#Ashley爱室丽"]
		}
	}`)

	return m
}

// episodesFixture 构建一个真实的 12 集整季 JSON 字符串，每一集的 hook、cliffhanger 与
// payoff 都非空且为中文，以保证在无 Key 的 demo 模式下节奏校验顺利通过。
func episodesFixture() string {
	type ep struct {
		Number      int      `json:"number"`
		Title       string   `json:"title"`
		Synopsis    string   `json:"synopsis"`
		Beats       []string `json:"beats"`
		Hook        string   `json:"hook"`
		Cliffhanger string   `json:"cliffhanger"`
		Payoff      string   `json:"payoff"`
	}
	titles := []string{
		"扫地出门", "一只行李箱", "迟来的继承信", "摊牌饭局",
		"买回整条街", "家具匠人", "工作室开张", "事业崛起",
		"渣男的追缴令", "揭晓邀请函", "真相曝光", "我亲手造的家",
	}
	hooks := []string{
		"一枚婚戒砸在大理石地板上，门当着她的脸狠狠摔上。",
		"天还没亮，她在空荡荡的出租屋地板上数着兜里最后四十块钱。",
		"一通律师电话打来：「你外公把一切都留给了你。」",
		"她给陆既明发去一句话：「今晚，饭局，把她也带上。」",
		"陆既明最想要的那栋房子门口，竖起了一块「已售」的牌子。",
		"火花四溅——是真的火花，在江岸的家具工坊里。",
		"她的第一个客户走进门，竟是陆既明的新欢。",
		"一夜之间，杂志把她评为「年度设计师」。",
		"陆既明的手机被打爆：所有贷款被同时催收。",
		"烫金的邀请函，落到了每一个仇人的家门口。",
		"一份藏起来的文件夹，揭穿了到底是谁背叛了她。",
		"探照灯齐刷刷亮起，照亮一座没人知道是她名下的庄园。",
	}
	cliffs := []string{
		"她转身离开时，一辆黑色轿车停下——专程来接她。",
		"律师递来的名片上，印着她外公的名字。",
		"她签下文件，才得知陆既明欠了她公司一大笔钱。",
		"陆既明的新欢看到苏念手里的房产证，当场晕了过去。",
		"江岸认出了苏念——从一张他本不该看到的照片里。",
		"苏念发现自己的设计稿被盗，正挂着陆既明的品牌在卖。",
		"工作室最大的金主，竟是陆既明背后那家银行。",
		"对手开出价码：要么被她收购，要么被她埋葬。",
		"陆既明出现在她门口，狼狈而危险。",
		"宾客名单上，有一个本不该还活着的名字。",
		"那个叛徒，是她曾经深爱过的人。",
		"她把第二份房产证递给江岸——一个写着他俩名字的家。",
	}
	payoffs := []string{
		"那个被当成「无名之辈」的她，揭晓为这座城市最新晋的隐形富豪。",
		"她一夜之间把空房间爆改成惊艳全网的 Ashley 样板间。",
		"她拿下继承权，也攥住了能让陆既明身败名裂的筹码。",
		"她在陆既明自己的饭桌上，把他当众打脸到无地自容。",
		"她抢在陆既明前头，拍下了他到处炫耀的那处地标豪宅。",
		"她和江岸联手设计出一套刷屏全网、瞬间售罄的爆款。",
		"她拿下了陆既明追了好几年都没拿到的那张合同。",
		"她的品牌在一个季度内就把陆既明的公司彻底盖了过去。",
		"陆既明的产业轰然崩塌，因为苏念买下了他全部的债务。",
		"每一个曾经亏待过她的人，都眼睁睁看着她扶摇直上。",
		"她在一场直播里，把那场背叛公之于众。",
		"她走进一个布置完整、终于真正属于自己的梦想之家。",
	}
	eps := make([]ep, 12)
	for i := 0; i < 12; i++ {
		eps[i] = ep{
			Number:      i + 1,
			Title:       titles[i],
			Synopsis:    "《" + titles[i] + "》：苏念把逆袭再往前推一步，一个全新的 Ashley 居家空间承载起本集的情绪转折。",
			Beats:       []string{"铺垫", "升级", "反转"},
			Hook:        hooks[i],
			Cliffhanger: cliffs[i],
			Payoff:      payoffs[i],
		}
	}
	wrap := map[string]any{"episodes": eps}
	b, _ := json.Marshal(wrap)
	return string(b)
}
