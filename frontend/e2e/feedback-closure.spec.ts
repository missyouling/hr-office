import { test, expect, type Locator, type Page } from "@playwright/test";
import { ACCOUNTS, login } from "./helpers/auth";

const API_PORT = "8080";

type SeedFeedback = { id: number; messageId: number };

/** 通过当前登录态创建真实聊天消息，并以该消息提交差评。 */
async function createFeedbackFromChat(page: Page, question: string, comment: string): Promise<SeedFeedback> {
  return page.evaluate(async ({ question, comment, apiPort }) => {
    const token = localStorage.getItem("token");
    if (!token) throw new Error("登录态中缺少 token");

    const apiBase = `${window.location.protocol}//${window.location.hostname}:${apiPort}/api`;
    const chatResponse = await fetch(`${apiBase}/knowledge/chat/stream`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ question, session_id: "" }),
    });
    if (!chatResponse.ok || !chatResponse.body) {
      throw new Error(`聊天请求失败：${chatResponse.status} ${await chatResponse.text()}`);
    }

    const reader = chatResponse.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    let messageId: number | undefined;
    while (messageId === undefined) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() || "";
      for (const line of lines) {
        if (!line.startsWith("data: ")) continue;
        const event = JSON.parse(line.slice(6)) as { type?: string; message_id?: unknown; content?: string };
        if (event.type === "error") throw new Error(`聊天服务返回错误：${event.content || "未知错误"}`);
        if (event.type === "done" && typeof event.message_id === "number") messageId = event.message_id;
      }
    }
    if (typeof messageId !== "number" || !Number.isInteger(messageId) || messageId <= 0) {
      throw new Error("聊天完成事件未返回有效的数值 message_id");
    }
    const realMessageId = messageId;

    const feedbackResponse = await fetch(`${apiBase}/feedback`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify({ message_id: String(realMessageId), rating: "negative", comment }),
    });
    if (!feedbackResponse.ok) throw new Error(`提交反馈失败：${feedbackResponse.status} ${await feedbackResponse.text()}`);
    const feedback = await feedbackResponse.json() as { id?: unknown };
    if (typeof feedback.id !== "number" || !Number.isInteger(feedback.id)) {
      throw new Error("反馈创建响应未返回数值 ID");
    }
    return { id: feedback.id, messageId: realMessageId };
  }, { question, comment, apiPort: API_PORT });
}

async function openMyFeedback(page: Page) {
  await page.getByRole("button", { name: /我的反馈/ }).click();
  await expect(page.getByRole("dialog").getByText("我的反馈", { exact: true })).toBeVisible();
}

/**
 * 定位“我的反馈”对话框中本次 question 对应的反馈条目（AccordionItem 容器）。
 * 状态徽章位于条目内触发器按钮中，折叠时即可见；回复内容需展开后才渲染。
 * 取最近的带 data-state 的 div 祖先，避免命中 radix DialogContent 自身的 data-state。
 */
function feedbackItem(dialog: Locator, question: string): Locator {
  return dialog
    .getByRole("button", { name: `查看反馈：${question}` })
    .locator("xpath=ancestor::div[@data-state][1]");
}

test.describe("P7.2 用户反馈闭环 E2E", () => {
  test("普通用户的真实聊天差评可被管理员回复和关闭，并同步展示状态", async ({ browser }) => {
    const suffix = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
    const question = `E2E反馈闭环提问-${suffix}`;
    const comment = `E2E反馈闭环差评-${suffix}`;
    const reply = `E2E管理员回复-${suffix}`;
    const userPage = await browser.newPage();
    const adminPage = await browser.newPage();

    await login(userPage, ACCOUNTS.viewer.username, ACCOUNTS.viewer.password);
    const seed = await createFeedbackFromChat(userPage, question, comment);
    expect(seed.messageId).toBeGreaterThan(0);
    expect(String(seed.messageId)).not.toMatch(/^msg-/);

    await openMyFeedback(userPage);
    const userDialog = userPage.getByRole("dialog");
    await expect(userDialog.getByText(question, { exact: true })).toBeVisible();
    // 状态断言限定到本次 question 对应的反馈条目，避免命中历史遗留反馈
    const userItem = feedbackItem(userDialog, question);
    await expect(userItem.getByText("待处理", { exact: true })).toBeVisible();

    await login(adminPage, ACCOUNTS.admin.username, ACCOUNTS.admin.password);
    await adminPage.getByRole("button", { name: "反馈管理", exact: true }).click();
    await expect(adminPage.getByRole("heading", { name: "AI 反馈闭环", exact: true })).toBeVisible();
    await adminPage.getByLabel("按状态筛选").click();
    await adminPage.getByRole("option", { name: "待处理", exact: true }).click();
    const pendingRow = adminPage.locator("tr").filter({ hasText: comment });
    await expect(pendingRow).toHaveCount(1);
    await expect(pendingRow.getByText("待处理", { exact: true })).toBeVisible();

    await pendingRow.getByRole("button", { name: "回复", exact: true }).click();
    await adminPage.getByLabel("管理员回复").fill(reply);
    await adminPage.getByRole("button", { name: "保存回复", exact: true }).click();
    await adminPage.getByLabel("按状态筛选").click();
    await adminPage.getByRole("option", { name: "已回复", exact: true }).click();
    const repliedRow = adminPage.locator("tr").filter({ hasText: comment });
    await expect(repliedRow.getByText("已回复", { exact: true })).toBeVisible();

    await userPage.reload();
    await expect(userPage.getByRole("button", { name: "员工管理", exact: true })).toBeVisible();
    await openMyFeedback(userPage);
    const userDialogAfterReply = userPage.getByRole("dialog");
    const userItemAfterReply = feedbackItem(userDialogAfterReply, question);
    await expect(userItemAfterReply.getByText("已回复", { exact: true })).toBeVisible();
    // 回复内容在 accordion 展开后才渲染，先展开本次条目再断言
    await userItemAfterReply.getByRole("button", { name: `查看反馈：${question}` }).click();
    await expect(userDialogAfterReply.getByText(reply, { exact: true })).toBeVisible();

    await repliedRow.getByRole("button", { name: "关闭", exact: true }).click();
    await expect(adminPage.getByRole("alertdialog")).toBeVisible();
    await adminPage.getByRole("button", { name: "确认关闭", exact: true }).click();
    await adminPage.getByLabel("按状态筛选").click();
    await adminPage.getByRole("option", { name: "已关闭", exact: true }).click();
    await expect(adminPage.locator("tr").filter({ hasText: comment }).getByText("已关闭", { exact: true })).toBeVisible();

    await userPage.reload();
    await expect(userPage.getByRole("button", { name: "员工管理", exact: true })).toBeVisible();
    await openMyFeedback(userPage);
    const userDialogAfterClose = userPage.getByRole("dialog");
    const userItemAfterClose = feedbackItem(userDialogAfterClose, question);
    await expect(userItemAfterClose.getByText("已关闭", { exact: true })).toBeVisible();
  });
});
