#!/usr/bin/env node

import { createHmac } from "node:crypto";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";

let chromium;
try {
  ({ chromium } = await import("playwright"));
} catch {
  ({ chromium } = await import("../../social-job-bot/node_modules/playwright/index.mjs"));
}

function output(obj) {
  process.stdout.write(`${JSON.stringify(obj)}\n`);
}

function fail(err) {
  output({ ok: false, error: err instanceof Error ? err.message : String(err) });
}

function splitCandidates(selector) {
  if (!selector) return [];
  return selector
    .split(",")
    .map((v) => v.trim())
    .filter(Boolean);
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, Math.max(0, ms)));
}

function isTargetClosedError(err) {
  const msg = String(err?.message || err || "").toLowerCase();
  return (
    msg.includes("target page") ||
    msg.includes("has been closed") ||
    msg.includes("context or browser has been closed") ||
    msg.includes("page closed")
  );
}

async function hasVisible(page, selector, timeout = 1200) {
  if (!selector) return false;
  try {
    return await page.locator(selector).first().isVisible({ timeout });
  } catch {
    return false;
  }
}

async function clickFirst(page, selector, timeout = 3000) {
  for (const candidate of splitCandidates(selector)) {
    try {
      const loc = page.locator(candidate).first();
      await loc.waitFor({ state: "visible", timeout });
      await loc.click({ timeout });
      return true;
    } catch {
      // try next candidate
    }
  }
  return false;
}

async function fillFirst(page, selector, text, timeout = 4000) {
  for (const candidate of splitCandidates(selector)) {
    const loc = page.locator(candidate).first();
    try {
      await loc.waitFor({ state: "visible", timeout });
      await loc.click({ timeout });
      const tag = (await loc.evaluate((el) => el.tagName.toLowerCase()).catch(() => "")) || "";
      if (tag === "input" || tag === "textarea") {
        await loc.fill("");
        await loc.type(text, { delay: 25 });
        await loc.evaluate((el, value) => {
          const proto = el.tagName === "TEXTAREA" ? window.HTMLTextAreaElement.prototype : window.HTMLInputElement.prototype;
          const setter = Object.getOwnPropertyDescriptor(proto, "value")?.set;
          if (setter) setter.call(el, value);
          else el.value = value;
          el.dispatchEvent(new InputEvent("input", { bubbles: true, data: value, inputType: "insertText" }));
          el.dispatchEvent(new Event("change", { bubbles: true }));
        }, text);
      } else {
        await page.keyboard.press("Control+A").catch(() => {});
        await page.keyboard.press("Meta+A").catch(() => {});
        await page.keyboard.press("Backspace").catch(() => {});
        await page.keyboard.type(text, { delay: 30 });
      }
      return true;
    } catch {
      // try next candidate
    }
  }
  return false;
}

async function submitKuaishouFallback(page) {
  return page
    .evaluate(() => {
      const input = document.querySelector("input[placeholder*='说点什么'],textarea[placeholder*='说点什么']");
      if (!input) return false;

      const send = document.querySelector(".send-btn:not(.disabled), .send-btn");
      if (!send) return false;

      if (send.classList.contains("disabled")) {
        input.dispatchEvent(new InputEvent("input", { bubbles: true, data: input.value || "", inputType: "insertText" }));
        input.dispatchEvent(new Event("change", { bubbles: true }));
      }

      const clickable = document.querySelector(".send-btn:not(.disabled)") || send;
      clickable?.click();
      return true;
    })
    .catch(() => false);
}

async function fillDouyinFallback(page, text) {
  return page
    .evaluate((value) => {
      const candidates = [
        ...document.querySelectorAll("div[contenteditable='true']"),
        ...document.querySelectorAll("textarea"),
        ...document.querySelectorAll("[role='combobox']"),
        ...document.querySelectorAll("input[placeholder*='评论'], input[placeholder*='说点什么'], input[placeholder*='留下']"),
      ];

      const target = candidates.find((el) => {
        const r = el.getBoundingClientRect();
        if (!r || r.width < 40 || r.height < 16) return false;
        const style = window.getComputedStyle(el);
        return style.display !== "none" && style.visibility !== "hidden";
      });
      if (!target) return false;

      target.focus?.();
      if (target.isContentEditable || target.getAttribute("role") === "combobox") {
        target.textContent = value;
      } else {
        const proto = target.tagName === "TEXTAREA" ? window.HTMLTextAreaElement.prototype : window.HTMLInputElement.prototype;
        const setter = Object.getOwnPropertyDescriptor(proto, "value")?.set;
        if (setter) setter.call(target, value);
        else target.value = value;
      }
      target.dispatchEvent(new InputEvent("input", { bubbles: true, data: value, inputType: "insertText" }));
      target.dispatchEvent(new Event("change", { bubbles: true }));
      return true;
    }, text)
    .catch(() => false);
}

async function openDouyinCommentInput(page) {
  return page
    .evaluate(() => {
      const trigger = [...document.querySelectorAll("div,span,button")].find((el) => {
        const txt = (el.textContent || "").trim();
        if (!/留下你的精彩评论吧|说点什么|写评论/.test(txt)) return false;
        const r = el.getBoundingClientRect();
        if (!r || r.width < 40 || r.height < 16) return false;
        const style = window.getComputedStyle(el);
        return style.display !== "none" && style.visibility !== "hidden";
      });
      if (!trigger) return false;
      trigger.click();
      return true;
    })
    .catch(() => false);
}

async function submitDouyinFallback(page) {
  return page
    .evaluate(() => {
      const all = [...document.querySelectorAll("button, span, div[role='button'], div")];
      const target = all.find((el) => {
        const txt = (el.textContent || "").trim();
        if (!/^(发布|发送)$/.test(txt)) return false;
        const r = el.getBoundingClientRect();
        if (!r || r.width < 20 || r.height < 16) return false;
        const style = window.getComputedStyle(el);
        if (style.display === "none" || style.visibility === "hidden") return false;
        if (el.classList?.contains("disabled")) return false;
        return true;
      });
      if (!target) return false;
      target.click();
      return true;
    })
    .catch(() => false);
}

function extractPostId(platform, url) {
  if (!url) return "";
  if (platform === "douyin") {
    const m = url.match(/\/video\/([0-9A-Za-z_-]+)/);
    return m ? m[1] : "";
  }
  if (platform === "kuaishou") {
    const m1 = url.match(/\/short-video\/([0-9A-Za-z_-]+)/);
    if (m1) return m1[1];
    const m2 = url.match(/\/video\/([0-9A-Za-z_-]+)/);
    if (m2) return m2[1];
    const m3 = url.match(/[?&]photoId=([0-9A-Za-z_-]+)/);
    return m3 ? m3[1] : "";
  }
  return "";
}

function searchURL(platform, keyword) {
  const q = encodeURIComponent(keyword);
  if (platform === "douyin") return `https://www.douyin.com/search/${q}?type=video`;
  if (platform === "kuaishou") return `https://www.kuaishou.com/search/video?searchKey=${q}`;
  throw new Error(`unsupported platform: ${platform}`);
}

function postURL(platform, postId) {
  if (platform === "douyin") return `https://www.douyin.com/video/${postId}`;
  if (platform === "kuaishou") return `https://www.kuaishou.com/short-video/${postId}`;
  throw new Error(`unsupported platform: ${platform}`);
}

async function resolveGatewayToken() {
  const fromEnv = (process.env.OPENCLAW_GATEWAY_TOKEN || "").trim();
  if (fromEnv) return fromEnv;

  const configPath = path.join(os.homedir(), ".openclaw", "openclaw.json");
  try {
    const raw = await fs.readFile(configPath, "utf8");
    const parsed = JSON.parse(raw);
    const token = parsed?.gateway?.auth?.token;
    return typeof token === "string" ? token.trim() : "";
  } catch {
    return "";
  }
}

function deriveRelayToken(gatewayToken, wsURL) {
  if (!gatewayToken) return "";
  try {
    const parsed = new URL(wsURL);
    const port = parsed.port ? Number(parsed.port) : parsed.protocol === "wss:" ? 443 : 80;
    if (!Number.isFinite(port) || port <= 0) return "";
    return createHmac("sha256", gatewayToken)
      .update(`openclaw-extension-relay-v1:${port}`)
      .digest("hex");
  } catch {
    return "";
  }
}

function withRelayToken(wsURL, relayToken) {
  if (!relayToken) return wsURL;
  try {
    const parsed = new URL(wsURL);
    if (!parsed.searchParams.get("token")) {
      parsed.searchParams.set("token", relayToken);
    }
    return parsed.toString();
  } catch {
    return wsURL;
  }
}

function homeURL(platform) {
  if (platform === "douyin") return "https://www.douyin.com";
  if (platform === "kuaishou") return "https://www.kuaishou.com";
  throw new Error(`unsupported platform: ${platform}`);
}

function platformHost(platform) {
  if (platform === "douyin") return "douyin.com";
  if (platform === "kuaishou") return "kuaishou.com";
  return "";
}

async function detectLoggedIn(page, platform, selectors) {
  if (await hasVisible(page, selectors.login_button, 1200)) {
    return false;
  }
  if (platform === "douyin") {
    if (await hasVisible(page, "a[href*='/user/self']", 1200)) return true;
    if (await hasVisible(page, "text=私信", 1200)) return true;
  }
  if (platform === "kuaishou") {
    if (await hasVisible(page, "text=消息", 1200)) return true;
    if (await hasVisible(page, "text=我的", 1200)) return true;
  }
  return !(await hasVisible(page, selectors.login_button, 800));
}

async function collectPosts(page, platform, selectors, limit) {
  const locator = page.locator(selectors.post_link || "a[href*='/video/'],a[href*='/short-video/']");
  const total = await locator.count();
  const scan = Math.min(total, Math.max(limit * 5, 80));

  const posts = [];
  const seen = new Set();

  for (let i = 0; i < scan && posts.length < limit; i++) {
    const link = locator.nth(i);
    let href = "";
    try {
      href = (await link.getAttribute("href")) || "";
      if (!href) {
        href =
          (await link.evaluate((el) => {
            return (
              el.getAttribute("data-href") ||
              el.getAttribute("data-url") ||
              el.closest("a")?.getAttribute("href") ||
              ""
            );
          }).catch(() => "")) || "";
      }
    } catch {
      continue;
    }
    if (!href) continue;

    let abs = "";
    try {
      abs = new URL(href, page.url()).href;
    } catch {
      continue;
    }

    const id = extractPostId(platform, abs);
    if (!id || seen.has(id)) continue;

    let title = "";
    try {
      title = ((await link.innerText()).trim() || "").slice(0, 200);
      if (!title) {
        title = ((await link.locator("xpath=ancestor::*[1]").innerText()).trim() || "").slice(0, 200);
      }
    } catch {
      title = "";
    }

    posts.push({
      id,
      title,
      url: abs,
      author_id: "",
    });
    seen.add(id);
  }

  return posts;
}

async function applySortAndFilter(page, sortBy, timeRange, selectors) {
  if (sortBy === "latest" && selectors.sort_latest) {
    await clickFirst(page, selectors.sort_latest, 2500);
    await sleep(700);
  }

  if (!timeRange || timeRange === "all") return;

  if (selectors.filter_button) {
    await clickFirst(page, selectors.filter_button, 2500);
    await sleep(500);
  }

  const labels = {
    day: /(一天内|24小时|24 小时)/,
    week: /(一周内|7天内)/,
    month: /(一个月内|30天内)/,
    year: /(一年内|12个月内)/,
  };

  const regex = labels[timeRange];
  if (!regex) return;

  const option = page
    .locator("button,span,div")
    .filter({ hasText: regex })
    .first();
  await option.click({ timeout: 2000 }).catch(() => {});
  await sleep(600);
}

function parseKuaishouCardID(postID) {
  const m = /^kscard:([^:]+):(\d+)$/i.exec(postID || "");
  if (!m) return null;
  return {
    keyword: decodeURIComponent(m[1]),
    index: Number(m[2]),
  };
}

async function collectKuaishouCards(page, keyword, limit) {
  const candidates = await page.evaluate(({ q, maxCount }) => {
    const normalizedQuery = (q || "").toLowerCase().trim();
    const bad = /(www\.kuaishou\.com|京ICP备|京公网安备|违法和不良信息举报|举报专区|未成年人关怀热线|可灵AI|Acfun|推荐\s*发现\s*关注\s*直播\s*赛事)/i;
    const norm = (s) => String(s || "").replace(/\s+/g, " ").trim();

    const nodes = Array.from(document.querySelectorAll("main div"));
    const picked = [];

    for (const el of nodes) {
      if (!el.querySelector("img")) continue;
      const rect = el.getBoundingClientRect();
      if (rect.width < 120 || rect.height < 80) continue;

      const style = window.getComputedStyle(el);
      if (style.display === "none" || style.visibility === "hidden") continue;
      if (style.cursor !== "pointer" && el.getAttribute("role") !== "button" && !el.onclick) continue;

      const text = norm(el.innerText);
      if (!text || text.length < 8 || text.length > 220) continue;
      if (bad.test(text)) continue;
      if (normalizedQuery && !text.toLowerCase().includes(normalizedQuery)) continue;

      picked.push(text.slice(0, 200));
      if (picked.length >= Math.max(maxCount * 4, 40)) break;
    }

    return Array.from(new Set(picked));
  }, { q: keyword, maxCount: Math.max(1, limit) });

  const posts = [];
  for (let i = 0; i < candidates.length && posts.length < limit; i++) {
    posts.push({
      id: `kscard:${encodeURIComponent(keyword)}:${i}`,
      title: candidates[i],
      url: page.url(),
      author_id: "",
    });
  }
  return posts;
}

async function openKuaishouCard(page, keyword, index) {
  const clicked = await page.evaluate(
    ({ q, targetIndex }) => {
      const normalizedQuery = (q || "").toLowerCase().trim();
      const bad = /(www\.kuaishou\.com|京ICP备|京公网安备|违法和不良信息举报|举报专区|未成年人关怀热线|可灵AI|Acfun|推荐\s*发现\s*关注\s*直播\s*赛事)/i;
      const norm = (s) => String(s || "").replace(/\s+/g, " ").trim();

      const nodes = Array.from(document.querySelectorAll("main div"));
      const picked = [];

      for (const el of nodes) {
        if (!el.querySelector("img")) continue;
        const rect = el.getBoundingClientRect();
        if (rect.width < 120 || rect.height < 80) continue;

        const style = window.getComputedStyle(el);
        if (style.display === "none" || style.visibility === "hidden") continue;
        if (style.cursor !== "pointer" && el.getAttribute("role") !== "button" && !el.onclick) continue;

        const text = norm(el.innerText);
        if (!text || text.length < 8 || text.length > 220) continue;
        if (bad.test(text)) continue;
        if (normalizedQuery && !text.toLowerCase().includes(normalizedQuery)) continue;

        picked.push(el);
      }

      const target = picked[Math.max(0, targetIndex)];
      if (!target) return { ok: false, count: picked.length };

      target.scrollIntoView({ behavior: "instant", block: "center" });
      target.click();
      return { ok: true, count: picked.length };
    },
    { q: keyword, targetIndex: index }
  );

  if (!clicked?.ok) {
    throw new Error(`failed to open kuaishou card at index ${index}, candidates=${clicked?.count || 0}`);
  }
}

async function openSession(input) {
  const browserCfg = input.browser || {};
  const actionTimeout = Math.max(1000, Number(browserCfg.action_timeout_ms || 20000));

  if (browserCfg.ws_url) {
    const gatewayToken = await resolveGatewayToken();
    const relayToken = deriveRelayToken(gatewayToken, browserCfg.ws_url) || gatewayToken;
    const wsURL = withRelayToken(browserCfg.ws_url, relayToken);

    const options = { timeout: actionTimeout };
    if (relayToken) {
      options.headers = { "x-openclaw-relay-token": relayToken };
    }

    const browser = await chromium.connectOverCDP(wsURL, options);
    const context = browser.contexts()[0] || (await browser.newContext());

    const host = platformHost(input.platform);
    const alivePages = context.pages().filter((p) => !p.isClosed());
    const preferred = alivePages.find((p) => {
      const url = (p.url() || "").toLowerCase();
      return host && url.includes(host);
    });
    const page = preferred || alivePages[0] || (await context.newPage());

    page.setDefaultTimeout(actionTimeout);
    page.setDefaultNavigationTimeout(Number(browserCfg.navigation_timeout_ms || 45000));
    return {
      browser,
      context,
      page,
      close: async () => {
        await browser.close().catch(() => {});
      },
    };
  }

  const userDataRoot = browserCfg.user_data_root || "./data/browser";
  const account = (input.account_id || "default").replace(/[^a-zA-Z0-9_-]/g, "_");
  const userDataDir = path.join(userDataRoot, input.platform, account);

  const context = await chromium.launchPersistentContext(userDataDir, {
    headless: Boolean(browserCfg.headless),
    executablePath: browserCfg.executable_path || undefined,
    args: ["--disable-blink-features=AutomationControlled"],
    viewport: { width: 1440, height: 920 },
  });
  const page = context.pages()[0] || (await context.newPage());
  page.setDefaultTimeout(actionTimeout);
  page.setDefaultNavigationTimeout(Number(browserCfg.navigation_timeout_ms || 45000));

  return {
    browser: null,
    context,
    page,
    close: async () => {
      await context.close().catch(() => {});
    },
  };
}

async function run(input) {
  const action = input.action;
  const platform = input.platform;
  const selectors = input.selectors || {};
  const postLoadWait = Math.max(0, Number(input.browser?.post_load_wait_ms || 1200));

  const attempt = async () => {
    const session = await openSession(input);
    const { page, close } = session;

    try {
      if (action === "login") {
        await page.goto(homeURL(platform), { waitUntil: "domcontentloaded" });
        if (await detectLoggedIn(page, platform, selectors)) {
          return { ok: true, logged_in: true, message: "already logged in" };
        }
        await clickFirst(page, selectors.login_button, 2500);
        await sleep(400);
        return {
          ok: true,
          logged_in: false,
          message: "login flow started; complete verification manually and re-check",
        };
      }

      if (action === "check_login") {
        await page.goto(homeURL(platform), { waitUntil: "domcontentloaded" });
        const loggedIn = await detectLoggedIn(page, platform, selectors);
        return {
          ok: true,
          logged_in: loggedIn,
          message: loggedIn ? "logged in" : "not logged in",
        };
      }

      if (action === "search") {
        if (!input.keyword) throw new Error("keyword is required");
        await page.goto(searchURL(platform, input.keyword), { waitUntil: "domcontentloaded" });
        await sleep(postLoadWait);

        await applySortAndFilter(page, input.sort_by || "comprehensive", input.time_range || "all", selectors);
        await sleep(postLoadWait);

        const limit = Math.max(1, Number(input.limit || 10));
        let posts = await collectPosts(page, platform, selectors, limit);
        if (platform === "kuaishou" && posts.length === 0) {
          posts = await collectKuaishouCards(page, input.keyword, limit);
        }
        return {
          ok: true,
          posts,
          message: `found ${posts.length} posts`,
        };
      }

      if (action === "comment") {
        if (!input.post_id) throw new Error("post_id is required");
        if (!input.content) throw new Error("content is required");

        if (platform === "kuaishou") {
          const card = parseKuaishouCardID(input.post_id);
          if (card) {
            await page.goto(searchURL(platform, card.keyword), { waitUntil: "domcontentloaded" });
            await sleep(postLoadWait);
            await openKuaishouCard(page, card.keyword, card.index);
          } else {
            await page.goto(postURL(platform, input.post_id), { waitUntil: "domcontentloaded" });
          }
        } else {
          await page.goto(postURL(platform, input.post_id), { waitUntil: "domcontentloaded" });
        }
        await sleep(postLoadWait);

        if (selectors.comment_button) {
          await clickFirst(page, selectors.comment_button, 2500);
          await sleep(500);
        }

        let typed = await fillFirst(page, selectors.comment_input, input.content, 4000);
        if (!typed && platform === "douyin") {
          await openDouyinCommentInput(page);
          await sleep(350);
          typed = await fillDouyinFallback(page, input.content);
        }
        if (!typed) {
          throw new Error(`comment input not found for ${platform}`);
        }

        let submitted = await clickFirst(page, selectors.comment_submit, 3000);
        if (!submitted && platform === "kuaishou") {
          submitted = await submitKuaishouFallback(page);
        }
        if (!submitted && platform === "douyin") {
          submitted = await submitDouyinFallback(page);
        }
        if (!submitted) {
          throw new Error(`comment submit button not found for ${platform}`);
        }

        await sleep(1200);
        return {
          ok: true,
          message: "comment submitted",
        };
      }

      throw new Error(`unsupported action: ${action}`);
    } finally {
      await close();
    }
  };

  try {
    return await attempt();
  } catch (err) {
    if (isTargetClosedError(err)) {
      return await attempt();
    }
    throw err;
  }
}

(async () => {
  try {
    const raw = process.argv[2];
    if (!raw) throw new Error("missing json argument");
    const input = JSON.parse(raw);
    const result = await run(input);
    output(result);
  } catch (err) {
    fail(err);
    process.exitCode = 1;
  }
})();
