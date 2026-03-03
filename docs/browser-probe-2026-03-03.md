# Browser Probe Report (2026-03-03)

Method:

- Local Playwright headless probe
- OpenClaw Browser Relay probe on a logged-in real browser tab

## Douyin

### Logged-in page probe (`/jingxuan`)

Observed stable nodes:

- Search nav link: `a[href*='/aisearch']`
- Header search input: `input[placeholder*='搜索你感兴趣的内容']`
- Header search submit: `button:has-text('搜索')`

Observed constraints:

- Direct URL `https://www.douyin.com/search/<keyword>` frequently renders anti-bot/captcha iframe in automation contexts.
- `aisearch` route is AI search UI, not classic sortable video list.

Working selector strategy for now:

- Keep toolbar selectors from prior probe for classic search route.
- Add fallback to header search input path when classic search route is blocked.

### Classic search route probe (`/search/<keyword>`)

From previous probe (partially blocked in relay, visible in headless):

- Toolbar root: `#search-toolbar-container`
- Sort comprehensive: `#search-toolbar-container span:has-text('综合')`
- Sort latest: `#search-toolbar-container span:has-text('最新')`
- Filter: `#search-toolbar-container div:has-text('筛选')`

## Kuaishou

### Search page probe (`/search/video?searchKey=<keyword>`)

Observed stable nodes:

- Search input: `input[placeholder='搜索你感兴趣的内容']`
- Search button: nearby button in the same search group
- Type tabs:
  - `button:has-text('视频')`
  - `button:has-text('用户')`

Result list/card candidates:

- Container area has repeated result cards with video title + author + like count.
- Card click opens right-side video/detail panel.

### Detail panel / comment area probe

Observed nodes after opening a result card:

- Comment panel trigger in right action rail: `.hover-tip.commentPanel`
- Right side panel root: `.side-area`
- Tab container: `.tab-nav-wrapper`
- Comment tab text node: `.tab-item.is-normal` with text `评论`

Observed constraint:

- Comment input box was not exposed in this run (likely lazy render / state-dependent / per-post limitation).

Fallback selector strategy:

- Try multiple comment input candidates in order:
  - `textarea`
  - `div[contenteditable='true']`
  - `input[placeholder*='说点']`
- If no input appears, mark the post as `comment_unavailable` and skip.

## Practical conclusion

- Login/search/type-tab nodes are stable enough for first implementation.
- Comment submit on Kuaishou requires runtime fallback + skip policy.
- Douyin classic search route may require anti-bot fallback path; implementation should support both routes.
