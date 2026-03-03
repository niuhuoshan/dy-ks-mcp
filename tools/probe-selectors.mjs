let chromium;
try {
  ({ chromium } = await import("playwright"));
} catch {
  ({ chromium } = await import("../../social-job-bot/node_modules/playwright/index.mjs"));
}

const TARGETS = [
  {
    name: "douyin",
    url: "https://www.douyin.com/search/%E6%B1%82%E8%81%8C",
    labels: ["综合", "筛选", "视频", "用户", "最新"]
  },
  {
    name: "kuaishou",
    url: "https://www.kuaishou.com/search/video?searchKey=%E6%B1%82%E8%81%8C",
    labels: ["综合", "筛选", "视频", "用户", "最新"]
  }
];

function cssPath(el) {
  if (!(el instanceof Element)) return "";
  const path = [];
  while (el && el.nodeType === 1 && path.length < 6) {
    let selector = el.nodeName.toLowerCase();
    if (el.id) {
      selector += `#${el.id}`;
      path.unshift(selector);
      break;
    }
    if (el.className && typeof el.className === "string") {
      const cls = el.className
        .split(/\s+/)
        .filter(Boolean)
        .slice(0, 2)
        .join(".");
      if (cls) selector += `.${cls}`;
    }
    const siblings = el.parentNode
      ? Array.from(el.parentNode.children).filter((x) => x.nodeName === el.nodeName)
      : [];
    if (siblings.length > 1) {
      selector += `:nth-of-type(${siblings.indexOf(el) + 1})`;
    }
    path.unshift(selector);
    el = el.parentElement;
  }
  return path.join(" > ");
}

async function probe(target) {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    userAgent:
      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
    locale: "zh-CN"
  });
  const page = await context.newPage();

  await page.addInitScript(() => {
    Object.defineProperty(navigator, "webdriver", { get: () => undefined });
  });

  await page.goto(target.url, { waitUntil: "domcontentloaded", timeout: 90000 });
  await page.waitForTimeout(5000);

  const report = await page.evaluate(({ labels }) => {
    function cssPath(el) {
      if (!(el instanceof Element)) return "";
      const path = [];
      while (el && el.nodeType === 1 && path.length < 6) {
        let selector = el.nodeName.toLowerCase();
        if (el.id) {
          selector += `#${el.id}`;
          path.unshift(selector);
          break;
        }
        if (el.className && typeof el.className === "string") {
          const cls = el.className
            .split(/\s+/)
            .filter(Boolean)
            .slice(0, 2)
            .join(".");
          if (cls) selector += `.${cls}`;
        }
        const siblings = el.parentNode
          ? Array.from(el.parentNode.children).filter((x) => x.nodeName === el.nodeName)
          : [];
        if (siblings.length > 1) {
          selector += `:nth-of-type(${siblings.indexOf(el) + 1})`;
        }
        path.unshift(selector);
        el = el.parentElement;
      }
      return path.join(" > ");
    }

    const out = {
      url: location.href,
      title: document.title,
      inputCandidates: [],
      labelNodes: {},
      postLinks: []
    };

    const inputs = Array.from(document.querySelectorAll("input,textarea,[contenteditable='true']"));
    out.inputCandidates = inputs.slice(0, 10).map((el) => ({
      tag: el.tagName,
      placeholder: el.getAttribute("placeholder") || "",
      cls: el.className || "",
      path: cssPath(el)
    }));

    for (const label of labels) {
      const nodes = Array.from(document.querySelectorAll("button,span,div,a"))
        .filter((n) => (n.textContent || "").trim() === label)
        .slice(0, 5);
      out.labelNodes[label] = nodes.map((n) => ({
        tag: n.tagName,
        cls: n.className || "",
        path: cssPath(n)
      }));
    }

    out.postLinks = Array.from(document.querySelectorAll("a[href*='/video/'],a[href*='/short-video/']"))
      .slice(0, 20)
      .map((a) => a.getAttribute("href"));

    return out;
  }, { labels: target.labels });

  await browser.close();
  return report;
}

async function main() {
  const reports = {};
  for (const target of TARGETS) {
    reports[target.name] = await probe(target);
  }
  console.log(JSON.stringify(reports, null, 2));
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
