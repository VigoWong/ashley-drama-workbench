# 设计文档:AI Agent 驱动的「短剧生产工作台」MVP

- **日期**:2026-06-05
- **背景**:Ashley(家具制造商)面试作业
- **截止**:2026-06-07(周日)24:00
- **技术栈**:后端 Go,前端 Next.js + Tailwind CSS,LLM 用 Gemini

---

## 1. 业务定位与产品立意

### 1.1 为什么家具公司要「短剧生产工作台」

短剧(竖屏微短剧)已成为家居品牌在海外的高 ROI 营销渠道。Ashley 是美国家具制造商,本工作台定位为:

> **面向美国市场的英文竖屏家居品牌营销短剧生产工具。剧情内自然植入 Ashley 家具产品,目标是品牌曝光 + 独立站/门店带货。**

这把一个通用的「短剧生成」作业,接到了面试公司的真实业务上——这是本作业 product sense 的核心。

### 1.2 家居适配的爆款题材

家居/家具天然能在以下题材里「有戏份」(场景即卖场):

- **家装改造逆袭**(makeover / from-broke-to-dream-home)
- **搬家新生活 / 离婚重启**(fresh start, rebuilding a home)
- **梦想之家**(building the dream home as a status arc)
- **家庭和解**(reconciliation set in a warm, well-furnished home)

每个题材都让沙发、床、餐桌等成为情绪载体与转化触点。

### 1.3 短剧创作的行业常识(融入 Prompt)

- 竖屏 9:16,单集 1~2 分钟
- **黄金 3 秒**开场钩子(hook)
- 每集**结尾悬念**(cliffhanger)留住观众
- **爽点 / 反转**(payoff / reversal)节奏密集
- 品牌营销短剧用**转化卡点**(CTA)替代付费卡点

---

## 2. 核心交付物:结构化「制作方案」

Pipeline 最终产出一个结构化方案(内部 JSON,对外渲染为 Markdown),包含 8 个区块:

| # | 区块 | 内容 |
|---|------|------|
| 1 | 立意 / Concept | logline、主题、目标受众、市场、基调、核心「爽点引擎」、核心冲突 |
| 2 | 剧集圣经 / Series Bible | 爆款式标题、题材标签、集数、单集时长、总时长、分发平台、品牌植入策略总纲 |
| 3 | 人物小传 / Characters | 主角/对手/情感线人物 + 人物弧光 + 关系图 |
| 4 | 分集大纲 / Episodes | 每集:标题、梗概、关键节拍、黄金3秒钩子、结尾悬念、爽点/反转 |
| 5 | 产品植入 / Brand Integration | 哪个 Ashley 品类植入哪集哪场景、情绪节点、CTA 时机(选品工具 grounding) |
| 6 | 英雄集脚本 / Hero Script | 1~2 个英雄集的分镜表(shot list)+ 样例台词 |
| 7 | 制作参数 / Production | 拍摄格式、预算档位、镜头数、卡司规模、场景/家具道具清单 |
| 8 | 分发转化 / Distribution | CTA 文案、挂链位置、hashtags |

### 默认参数(可输入覆盖)

- 默认 **12 集 × 90 秒**
- 英雄集脚本只全量产出 **1~2 集**(控制 token)

---

## 3. 系统与 Agent 架构

### 3.1 架构选型

**编排器驱动的分阶段流水线 + 确定性工具闸门 + 自纠正回路。**

- 每个阶段 = 独立 Prompt 模板 + Gemini 结构化输出
- 工具(题材库/选品)在固定点注入「事实数据」做 grounding
- **节奏校验器是纯 Go 确定性代码**,对分集阶段做闸门校验 + 一次 refine 重写

> 设计理念:LLM 负责创作,确定性代码负责校验,二者构成反馈回路。这既是真正的 agentic orchestration,也体现「知道何时该/不该用 LLM」的工程判断。

被否决的替代方案:
- 每阶段 Function-Calling 自主 ReAct——不可预测、难测、token 贵
- 单 Mega-Prompt——谈不上编排

### 3.2 流水线阶段

```
[0] 需求解析 Brief Parser   → 归一化输入(题材/集数/时长/市场/品牌重点)
[1] 创意总监 Concept Agent  → 立意           (工具: 题材库检索)
[2] 剧集架构 Architect      → 圣经 / 节奏弧    (工具: 节奏校验器)
[3] 人物设计 Character Agent→ 人物小传 + 关系图
[4] 分集编剧 Episode Agent  → 分集大纲        (工具: 节奏校验器 → 不达标重写一次) ★自纠正
[5] 品牌植入 Brand Agent    → 植入方案        (工具: Ashley 选品)
[6] 分镜编剧 Scene Agent    → 英雄集分镜 + 台词
[7] 制片 Producer Agent     → 制作参数 + 分发转化
[8] 装配 Assembler          → 合并完整方案 + 渲染 Markdown
```

每阶段实现统一接口:

```go
type Stage interface {
    Name() string
    Run(ctx context.Context, s *PlanState) error // 读写共享的方案状态
}
```

编排器顺序执行各阶段,每阶段开始/结束都通过 SSE 推送事件给前端。

### 3.3 工具设计(3 个)

| 工具 | 签名 | 性质 | 作用 |
|------|------|------|------|
| 题材库检索 | `GetWinningTropes(market, vertical)` | 精选静态数据 | 家居向爆款题材 grounding(RAG-lite) |
| 选品 | `GetProductCatalog(category?)` | 静态数据 | Ashley 品类 + 卖点,植入方案 grounding |
| 节奏校验器 | `ValidatePacing(episodes) → Report` | **纯 Go 确定性** | 钩子/悬念/爽点分布打分 + 问题清单,驱动自纠正 |

`ValidatePacing` 校验规则示例:每集必须有非空 hook 与 cliffhanger;爽点/反转在全季的分布密度达标;首集开场钩子强度等。返回结构化 `Report{Score, Issues[]}`,Episode Agent 据此重写一次。

---

## 4. 技术实现

### 4.1 后端 Go(极简依赖:标准库 + chi 路由)

```
backend/
  cmd/server/        HTTP + SSE 服务
  cmd/cli/           CLI 入口(与 server 共用 pipeline)
  internal/llm/      Provider 接口 + Gemini 实现 + Mock 实现
  internal/agent/    Stage 接口 + 编排器 + 各阶段实现
  internal/tools/    题材库 / 选品 / 节奏校验器
  internal/prompts/  Prompt 模板(embed.FS)
  internal/model/    领域类型(整个方案的结构体)
  internal/sse/      SSE 事件推送
```

**LLM Provider 接口可插拔**:

```go
type Provider interface {
    GenerateJSON(ctx context.Context, prompt string, schema any) (json.RawMessage, error)
}
```

- `GeminiProvider`:真实调用 Gemini,要求结构化(JSON)输出
- `MockProvider`:返回固定/模板化的合理产物——**让单测和无 Key 演示都能跑通整条链路**

**错误处理**:
- LLM 调用失败 → 指数退避重试(最多 2~3 次)
- JSON 解析失败 → 自动 repair 重试(把错误回传模型让其修正)
- 阶段持久失败 → 推送 error 事件,流程中止并保留已完成部分

### 4.2 前端 Next.js + Tailwind(App Router)

单页工作台:

1. **输入表单**:题材、集数、单集时长、市场、品牌重点
2. 点击「生成」→ `POST /api/generate` 触发,经 SSE 订阅进度
3. **实时阶段时间线**:每个阶段展示状态(待办/进行中/完成)与产出,流式渲染
4. **最终方案**:8 区块结构化展示
5. **导出**:Markdown / JSON 下载

视觉打磨阶段使用 frontend-design skill。

### 4.3 数据流

```
用户输入 → POST /api/generate → 编排器顺序跑各阶段
        → 每阶段 emit SSE 事件 {stage, status, output}
        → 前端时间线渲染
        → 最终 event 携带完整方案 → 前端展示 + 导出
CLI:    同一编排器,终端打印阶段进度,输出 JSON/Markdown 文件
```

### 4.4 测试

- 工具单测(尤其 `ValidatePacing` 确定性逻辑)
- 用 `MockProvider` 跑 pipeline 集成测试(无需 API Key)
- CLI 冒烟测试

---

## 5. 交付物

1. **代码仓库**:`backend/` + `frontend/`,可一键运行(含 `.env.example`、Makefile/脚本)
2. **README + 设计说明**:业务理解、Agent/工具设计、Prompt 工程决策、product sense、运行方式

---

## 6. 范围边界(YAGNI)

MVP **不做**:用户账号/鉴权、方案持久化数据库、多用户、人机协作逐阶段编辑(本期一键生成 + 流式展示)、视频实际生成、多市场动态切换(默认锁定美国/英文/家居)。这些在 README 的「后续演进」里点到为止即可。
