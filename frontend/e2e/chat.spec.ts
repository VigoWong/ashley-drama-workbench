// frontend/e2e/chat.spec.ts
// Run manually with Playwright against a dev server + mock backend. Not wired into
// package.json (no test runner is installed). Documents the expected happy path of
// the conversational ReAct agent in 对话 mode (the no-key DemoMock trace).
import { test, expect } from "@playwright/test"

test("conversational agent streams a ReAct trace and builds the plan", async ({ page }) => {
  await page.goto("http://localhost:3000")
  // log in with default mock creds admin/admin
  await page.getByLabel(/用户名|username/i).fill("admin")
  await page.getByLabel(/密码|password/i).fill("admin")
  await page.getByRole("button", { name: /登录|login/i }).click()

  // chat is the default mode; send a brief
  await page.getByPlaceholder(/输入需求/).fill("做个家装逆袭短剧,植入 Ashley 沙发")
  await page.getByRole("button", { name: "发送" }).click()

  // a tool card appears (agent calling tools)
  await expect(page.getByText("🔧").first()).toBeVisible({ timeout: 15000 })
  // the canvas builds: the 立意 chip reaches done (✓)
  await expect(page.getByText(/✓ 立意/)).toBeVisible({ timeout: 30000 })
  // turn completes: composer re-enabled
  await expect(page.getByPlaceholder(/输入需求/)).toBeEnabled({ timeout: 30000 })
})
