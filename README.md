# 短剧生产工作台 · Ashley 家具品牌带货

输入**一句需求**（题材、集数、单集秒数、品牌植入重点，可选参考图），多 Agent 流水线即可产出一份结构化、可直接交付编剧室与剧组的**中文短剧制作方案**。方案面向**中国国内市场**的竖屏短剧（抖音 / 快手 / 红果短剧），剧情中自然植入 **Ashley（爱室丽）家具**，把"看剧"变成"带货"。

它解决的核心问题：**家具品牌如何用短剧做内容带货。** 短剧最常见的场景——爆改出租屋、离婚后重新开始、打造梦想之家——本身就发生在客厅、卧室、餐桌旁。**布景即展厅**：一张沙发不是和解戏里的道具，它就是那场戏的情绪锚点。把"情绪节点 → SKU → CTA"绑定起来，剧情卖的是情绪，CTA 卖的是商品。

两个入口共用同一套编排器：

- **HTTP + SSE 服务**（`backend/cmd/server`）：`POST /api/generate` 流式推送每个阶段的进度与产物，由 **Next.js 16 + Tailwind v4** 前端驱动。
- **CLI**（`backend/cmd/cli`）：同一条流水线，输出 Markdown 或 JSON。

> 设计文档与实现计划（历史过程产物，英文）见 [`docs/superpowers/`](docs/superpowers/)。

---

## 1. 交付物：8 区块制作方案

最终产出是一份结构化的 `Plan`（见 `backend/internal/model/plan.go`），含 8 个区块：

| # | 区块 | 内容 |
|---|------|------|
| 1 | **立意 Concept** | 一句话梗概、主题、目标观众、基调、**核心爽点引擎**、核心冲突、所用套路 |
| 2 | **剧集圣经 Series Bible** | 适合追看的剧名、题材标签、集数/单集秒数/总时长、分发平台、品牌植入主线 |
| 3 | **人物 Characters** | 主角 / 反派 / 爱情线等人物的小传、人物弧光、关系网 |
| 4 | **分集 Episodes** | 每集梗概、节拍、黄金 3 秒钩子、结尾悬念、爽点/反转 |
| 5 | **品牌植入 Placements** | 哪一集、哪个场景、植入哪个 Ashley SKU、踩在哪个情绪节点、CTA 何时出现 |
| 6 | **英雄分镜 Hero** | 为 1-2 个爽点最高的剧集生成完整分镜表 + 样例台词 |
| 7 | **制作 Production** | 画幅、预算档位、镜头数、卡司数、场景地点、家具道具清单 |
| 8 | **分发 Distribution** | CTA 文案、挂链位置、话题标签 |

**默认 5 集 × 30 秒。** 服务端会对缺省字段补默认值（见 `Brief.ApplyDefaults`：集数 5、单集 30 秒、市场"中国"、语言"中文"）。

---

## 2. 业务理解：国内短剧手法

流水线把真实的国内短剧创作手法编码进确定性规则与提示词，而不是简单地"帮我写个剧本"：

- **黄金 3 秒钩子。** 每集开场必须有让人停止划走的强钩子，第 1 集尤其要在第一屏就抓住人。这一条由确定性校验器**强制**，而非"但愿如此"。
- **每集结尾悬念。** 每集都留一个未闭合的钩子，逼观众点"下一集"。每集 `cliffhanger` 必须非空。
- **爽点/反转密度。** 短剧靠高频的解气时刻（揭晓、打脸、逆袭、阶层跃升）留人。要求全季**至少 60% 的剧集**带爽点。
- **CTA 代替付费墙。** 付费短剧用金币锁集变现；**品牌**短剧不锁任何东西——它做的是转化，在情绪最高点放一张购物 CTA。

**家居向爽剧套路**（精选于 `backend/internal/tools/tropes.go`）是题材引擎，每条都因"家具能自然赢得镜头"而入选：家装改造逆袭、离婚后爆改出租屋、重生之打造梦想之家、婆媳和解之家、赘婿战神回归。

**Ashley 植入策略**把这一切串起来：植入阶段把 Ashley 的真实 SKU（`backend/internal/tools/catalog.go`，如 *Maeford Sectional* 转角沙发、*Realyn Queen Bed* 大床、*Haddigan Dining Set* 餐桌套装）按 **情绪节点 → SKU → CTA 时机** 映射到场景。合家欢落在转角沙发上，重新开始的清晨落在大床上。

---

## 3. 功能总览

- **登录门**：单用户 token 鉴权，默认 `admin / admin`（可经环境变量覆盖）。
- **三步向导**：① 填需求 → ② 生成中（流式时间线）→ ③ 查看方案。
- **多模态参考图**：6 张预设家居素材 + 本地上传（合计 ≤ 3 张），作为参考图喂给模型，影响空间风格与家具质感。
- **真实 / 示例两种生成**：配 Key 走真实大模型；无 Key 自动降级为内置 Mock，返回一份完整可信的中文示例方案——**整套流程与测试无需任何 Key**。
- **历史**：配置数据库后，每次生成自动持久化，可在历史页查看列表与完整方案详情。
- **导出**：方案页一键导出 **JSON** 或 **Markdown**。

---

## 4. 架构

**编排器驱动的多阶段流水线。** 每个阶段 = 一个内嵌的提示词模板 + Gemini 结构化 JSON 输出，读写线程间共享的 `*PlanState`。确定性 Go 工具为大模型提供真实数据接地，并以一次性自纠正回路对 episodes 阶段**把关**。可插拔的 `Provider` 让整条链在无 Key 时也能跑、能测。

```
   Brief {genre, episodes, episodeSecs, brandFocus, images[]}  +  登录 token
                              │
   POST /api/generate ──▶ Bearer 鉴权中间件 ──▶ llm.FromEnv() 选 Provider
                              │
   ┌────────────────────  编排器 Orchestrator  ───────────────────────┐
   │  按序运行各阶段，线程化 *PlanState，逐阶段 emit SSE 事件             │
   │  单阶段失败自动重试（≤ 3 次、线性退避、尊重 ctx 取消）              │
   │                                                                    │
   │  [1] concept ◀── GetWinningTropes()  (家居套路库)  ◀── 参考图       │
   │  [2] bible                                                          │
   │  [3] characters                                                     │
   │  [4] episodes                                                       │
   │        │                                                           │
   │        ▼  ValidatePacing()  纯 Go：每集有钩子+悬念？爽点密度≥60%？   │
   │        │ pass            fail │                                     │
   │        │                      ▼                                     │
   │        │   [4b] episodes_refine  一次性 LLM 重写（喂入失败规则报告）  │
   │        │◀─────────────────────┘                                    │
   │  [5] placements ◀── GetProductCatalog()  (Ashley SKU)  ◀── 参考图   │
   │  [6] hero  (1-2 个高爽点剧集的分镜表 + 台词)         ◀── 参考图       │
   │  [7] production_distribution                                        │
   │        ▼                                                            │
   │  complete ──▶ 完整 Plan（8 区块）                                    │
   └───────────────────────────────────────────────────────────────────┘
                              │
   每阶段调用：Provider.GenerateJSON(ctx, stage, prompt, images, schema)
                              │
        ┌─────────────────┬──┴──────────────────┐
        ▼                 ▼                      ▼
   Vertex AI          AI Studio            Mock / DemoMock
 (SA JSON → OAuth   (GEMINI_API_KEY,      (无 Key，返回完整
  → 区域端点)        可选 BASE_URL 代理)     中文示例方案)
        └────── 同一套 generateContent 线格式；仅 URL/鉴权不同 ──────┘

   生成完成后：若配置了 DATABASE_URL → 异步写入 Postgres（plans 表）
```

- **7 个流水线阶段**（`AllStages()`，`backend/internal/agent/stages.go`）：`concept → bible → characters → episodes → placements → hero → production_distribution`。`episodes_refine` 是 episodes 阶段内部的一次性重写，不是顶层阶段。
- **节奏自纠正**：episodes 生成后跑确定性 `ValidatePacing` 闸门；不通过则把结构化失败报告（`PacingReport.FormatIssues()`）连同被拒草稿喂进 `episodes_refine` 提示词，做**恰好一次**修正重写。
- **单阶段重试**：编排器对任一阶段的瞬时失败（真实大模型偶发的传输错误或脏 JSON）重试 ≤ 3 次，线性退避（attempt × 700ms），ctx 取消即停（客户端断开不空转）。
- **3 个确定性工具**（`backend/internal/tools/`）：`GetWinningTropes`（中文家居题材库）、`GetProductCatalog`（Ashley SKU 库）、`ValidatePacing`（纯 Go 节奏校验闸门）。
- **多模态**：`Brief.Images`（`[]Image{mimeType, data(纯 base64), label}`）只喂给 **concept / placements / hero** 三个视觉阶段以控制 token 成本；其余阶段不带图。
- **2 个入口、1 套编排器**：`cmd/server`（HTTP+SSE）与 `cmd/cli` 都调用 `agent.New(provider, emit).Run(ctx, brief)`。

---

## 5. Prompt 与 AI 编排

**为什么分阶段，而非自由 ReAct。** 固定的单一职责提示词序列是**可控的**（明确知道跑了什么、什么顺序）、**可观测的**（每阶段 emit 一个独立 SSE 事件携带局部产物）、**可测的**（每阶段与整条链都能对 Mock 跑测试）。自主函数调用 Agent 不可预测、难测试、token 昂贵；而单个巨型提示词根本谈不上"编排"。

**为什么 `ValidatePacing` 是确定性 Go，而非又一次 LLM 调用。** 节奏规则是精确可验证的——*每集是否都有非空钩子和悬念？爽点密度是否 ≥ 60%？*——用另一次随机的大模型调用去判定，会更慢、更贵、且**更不可靠**。核心设计立场是：**大模型负责创作，确定性代码负责校验，反馈回路负责重写。** 这才是系统中真正"Agentic"的部分，且接地于代码而非感觉。

**结构化输出。** Gemini provider 请求 `responseMimeType: "application/json"`，每个阶段的提示词末尾都给出目标结构体的精确 JSON 形状。

**重试 + JSON 修复。** `Gemini.GenerateJSON`（`backend/internal/llm/gemini.go`）内部最多 3 次尝试：传输/HTTP 错误退避重试；若返回了内容但非合法 JSON（剥掉 ``` 围栏后），下一次会把模型自己的坏输出回喂，要求只返回严格 JSON。Gemini 2.5 系模型默认会"思考"并把推理混进文本，破坏严格 JSON 解析，因此对 2.5 模型设 `thinkingBudget=0`。

**多模态如何接入。** 参考图以 `inlineData`（mimeType + 纯 base64）跟在文本提示词后，组成单条 user content；AI Studio 与 Vertex 线格式一致。仅视觉相关的三个阶段带图。

---

## 6. 运行指南

**前置：** Go 1.23+、Node 18+；历史功能需要 Docker（Postgres）。

### 6.1 起数据库（历史持久化，可选）

```bash
# 默认端口 5432；本机若被占用，用 DB_PORT 覆盖，例如 5433
DB_PORT=5433 docker compose up -d
# 可选数据库浏览器 Adminer：http://localhost:8082
```

数据库就绪后，后端通过 `DATABASE_URL` 连接（注意端口要和上面一致）：

```bash
export DATABASE_URL="postgres://drama:drama@localhost:5433/drama?sslmode=disable"
```

> **优雅降级**：不设 `DATABASE_URL` 或连不上时，生成照常工作；`/api/history` 返回空列表、详情返回 404。

### 6.2 后端：三种 Provider 模式

`llm.FromEnv()` 按优先级择一（`backend/internal/llm/factory.go`）：

**① Vertex AI**（优先级最高）

```bash
export VERTEX_CREDENTIALS_FILE=/path/to/vertex-sa.json   # 服务账号 JSON
export VERTEX_LOCATION=us-central1                        # 可选，默认 us-central1
export VERTEX_PROJECT=your-gcp-project                    # 可选，缺省取 SA 的 project_id
export GEMINI_MODEL=gemini-2.5-flash                      # 可选，Vertex 默认 gemini-2.5-flash
```

**② AI Studio Key**

```bash
export GEMINI_API_KEY=your-key
export GEMINI_MODEL=gemini-2.0-flash    # 可选，AI Studio 默认 gemini-2.0-flash
export GEMINI_BASE_URL=...              # 可选，指向兼容 generateContent 的代理
```

**③ 无 Key 演示模式**：以上都不配，自动用 `DemoMock`，返回一份完整中文示例方案。

> ⚠️ Vertex 的 `vertex-sa.json`、本机 `run-dev.sh`、`.env.local` 等均被 gitignore，**请勿提交任何密钥**。

**鉴权环境变量**（可选）：

```bash
export AUTH_USERNAME=admin   # 默认 admin
export AUTH_PASSWORD=admin   # 默认 admin
```

**端口**：`PORT` 默认 `8080`。

**启动**（命令均在 `backend/` 下，`make` 目标见 `backend/Makefile`）：

```bash
cd backend
make server                                  # 启动 HTTP+SSE 服务（默认 :8080）
make cli                                      # 跑 CLI（通过 ARGS 传参）
make cli ARGS="-episodes 5 -format json"      # 例：5 集、JSON 输出
go run ./cmd/cli -episodes 5 -format markdown # 直接运行 CLI
make build                                    # 编译出 bin/server 与 bin/cli
```

> CLI 现有默认值（`backend/cmd/cli/main.go`）仍为占位的英文示例（`-genre "home makeover revenge"`、`-episodes 12`、`-secs 90`），传中文题材即可，例如：
> `go run ./cmd/cli -genre "家装改造逆袭" -episodes 5 -secs 30 -brand "客厅沙发、卧室套装"`

调用流式接口（需先登录拿 token）：

```bash
# 1) 登录拿 token
curl -s -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}'

# 2) 带 Bearer token 调用生成
curl -N -X POST http://localhost:8080/api/generate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"genre":"家装改造逆袭","episodes":5,"episodeSecs":30,"brandFocus":"客厅沙发、卧室套装"}'
```

会看到一串 `stage_start` / `stage_done` 事件，以一个携带完整方案的 `complete` 事件收尾。

**HTTP 端点一览**（`backend/cmd/server/main.go`）：

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/login` | 公开 | 校验账密，返回 `{"token": "..."}` |
| POST | `/api/generate` | Bearer | SSE 流式生成方案 |
| GET | `/api/history` | Bearer | 历史方案列表（无 DB 时返回空数组） |
| GET | `/api/history/{id}` | Bearer | 历史方案详情（无 DB 时 404） |
| GET | `/api/health` | 公开 | 健康检查，返回 `ok` |

### 6.3 前端

```bash
cd frontend
npm install
npm run dev      # http://localhost:3000
```

前端默认调用 `http://localhost:8080`；如后端在别处，用 `NEXT_PUBLIC_API` 覆盖（或写进 `frontend/.env.local`，参考 `frontend/.env.local.example`）：

```bash
NEXT_PUBLIC_API=http://localhost:8091 npm run dev
```

> 本机开发若 8080 被占用，常把后端跑在 8091，并令 `.env.local` 指向 8091。

打开页面后用 `admin / admin` 登录即可进入工作台。

---

## 7. 测试

整套测试**无需任何 Key、无需数据库**——Mock provider 顶替大模型，store 集成测试在 `DATABASE_URL` 未设时自动 `Skip`：

```bash
cd backend && go test ./...
```

值得一提的测试：

- **确定性节奏校验**（`internal/tools/pacing_test.go`）：`TestPacingPerfectScoresHigh`、`TestPacingMissingHookFails`、`TestPacingLowPayoffDensityFails` 钉死驱动自纠正回路的闸门规则。
- **鉴权**（`internal/auth/auth_test.go`）：`TestLoginRejectsBadCredentials`、`TestLoginAcceptsDefaultsAndTokenWorks`。
- **编排器**（`internal/agent/orchestrator_test.go`）：`TestOrchestratorRunsAllStages`、`TestOrchestratorRetriesTransientStageFailure`（验证单阶段重试）。
- **阶段与自纠正**（`internal/agent/stages_test.go`）：`TestEpisodeStageRefinesOnBadPacing`。
- **数据库往返**（`internal/store/store_test.go`）：`TestRoundTrip`，需指向 docker-compose 的 Postgres 才会真正执行：
  `DATABASE_URL=postgres://drama:drama@localhost:5433/drama?sslmode=disable go test ./internal/store`

---

## 8. 项目结构

```
backend/
  cmd/server/main.go        HTTP + SSE 服务：登录、鉴权中间件、生成、历史、健康检查
  cmd/cli/main.go           复用同一编排器的 CLI（markdown/json 输出）
  internal/
    model/plan.go           领域模型：Brief / Plan / Episode / Image ... 及默认值
    model/events.go         SSE 事件类型
    llm/provider.go         Provider 接口（GenerateJSON）
    llm/gemini.go           Gemini provider：AI Studio + Vertex 双模式、3 次重试、JSON 修复
    llm/factory.go          FromEnv：Vertex → AI Studio → Mock 优先级选择
    llm/mock.go             Mock 与 DemoMock（无 Key 的完整中文示例方案）
    tools/tropes.go         GetWinningTropes：中文家居题材库
    tools/catalog.go        GetProductCatalog：Ashley SKU 库
    tools/pacing.go         ValidatePacing：纯 Go 节奏校验闸门
    prompts/*.tmpl          每阶段一个内嵌中文提示词模板（含 episodes_refine）
    prompts/embed.go        模板嵌入与渲染
    agent/stage.go          Stage 接口、PlanState、Emitter
    agent/stages.go         7 个阶段实现 + episodes 自纠正 + AllStages()
    agent/orchestrator.go   编排器：按序运行、单阶段重试、emit 事件
    auth/auth.go            单用户 token 鉴权（LoginHandler + Bearer 中间件）
    store/store.go          Postgres 持久化（plans 表，brief/plan 存 jsonb）
    render/markdown.go      Plan → 中文 Markdown
  Makefile                  test / server / cli / build
  .env.example              环境变量示例
docker-compose.yml          Postgres（持久化）+ 可选 Adminer
frontend/
  app/page.tsx              工作台主页：登录门 + 三步向导 + 历史切换
  components/LoginForm.tsx  登录表单
  components/InputForm.tsx  需求表单 + 参考素材（预设 + 上传，≤3 张）
  components/StageTimeline.tsx  SSE 实时阶段时间线（可展开原始输出）
  components/PlanView.tsx   8 区块方案展示
  components/ExportBar.tsx  JSON / Markdown 导出
  components/HistoryView.tsx  历史列表与详情
  components/Stepper.tsx    三步进度指示
  lib/api.ts               POST + SSE 解析、UnauthorizedError
  lib/auth.ts              登录、token 存取
  lib/history.ts           历史列表/详情接口
  lib/materials.ts         6 张预设家居素材定义
  lib/markdown.ts          客户端 Plan → Markdown 导出
  lib/types.ts             Go 模型的 TypeScript 镜像
  public/materials/        预设素材缩略图
docs/superpowers/          设计文档 + 实现计划（历史过程产物，英文）
```

---

## 9. 设计取舍与后续演进

**范围刻意收敛（YAGNI）。** 当前聚焦"可控、可测的 Agentic 生成核心 + 一条可用的端到端体验"，因此有意保持简单：

- 鉴权是**单用户、进程内 token**，默认 `admin/admin`（demo 便利，env 可覆盖）。生产加固方向：去掉默认值、缺失关键 env 即启动失败、token 持久化与过期。
- 数据库是**单表 + jsonb**，够历史功能用即可，不做复杂建模。
- 工具是**精选静态数据**（套路库、SKU 库），尚非真 RAG。
- 多模态参考图仅喂给 3 个视觉阶段以控成本。
- DemoMock 返回固定示例（固定 ~12 集），用于演示链路、不随入参变化。

**后续演进**（大致按优先级）：

1. **逐阶段人工编辑** —— 审批/修改某阶段产物后，从该处重新生成下游阶段。
2. **视频生成衔接** —— 把英雄分镜表喂给文生视频流水线。
3. **真 RAG** —— 套路库与 SKU 库改由实时商品流 + 趋势/表现指数支撑。
4. **多市场 / 多语言** —— 题材工具已为 `market` 预留参数位。
