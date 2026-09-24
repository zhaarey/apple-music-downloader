# Captured responses

Captured 2026-09-21 (UTC) without user credentials. All requests returned HTTP 200.
Catalog captures used the public web Developer Token; no token is stored here.

| File | Request |
|---|---|
| api/jp_pachelbel_en-US.json | `/api/classical/v10/query/view/jp/recording/johann-pachelbel-1653-pp429-1452536848?l=en-US` |
| api/us_beethoven_en-US.json | `/api/classical/v10/query/view/us/recording/ludwig-van-beethoven-1770-pp193-1873004116?l=en-US` |
| api/us_beethoven_default.json | same, without `l` |
| api/cn_pachelbel_zh-CN.json, api/cn_pachelbel_en-US.json | `/api/classical/v10/query/view/cn/recording/johann-pachelbel-1653-pp429-1452536848?l=…` |
| metadata/*.json | the matching `…/tracksMetadata` request |
| catalog/*.json | `https://amp-api.music.apple.com/v1/catalog/{sf}/albums/{albumId}?l=…` |
| ssr/cn_pachelbel_*.html | public page `https://classical.music.apple.com/cn/recording/johann-pachelbel-1653-pp429-1452536848?l=…` |
| redirects/cn_homepage.html | JP/US public Recording pages, which redirected to the `/cn` homepage |
| redirects/empty_response.bin | an observed empty body |
