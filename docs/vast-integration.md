# VAST Integration

OpenAdSource emits **VAST 4.2 inline responses** with one `<Linear>`
creative, one `<MediaFile>`, and the full tracking event set
(`impression`, `start`, `firstQuartile`, `midpoint`, `thirdQuartile`,
`complete`, `clickThrough`). Any player that speaks VAST 2/3/4 can
consume it.

This page is a copy-paste catalog for the common players. The formal
HTTP contract is in `api.md`; for the adserver-side mechanics see
`architecture.md`.

---

## What the response looks like

Request:

```
GET https://ads.example.com/vast?pos=pre&country=FR&device=mobile
```

Response (abridged, real responses include the full quartile set):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad id="9a8b7c…">
    <InLine>
      <AdSystem>OpenAdSource</AdSystem>
      <AdTitle><![CDATA[Demo: Big Buck Bunny]]></AdTitle>
      <Impression><![CDATA[https://ads.example.com/track?event=impression&ad_id=9a8b…&imp_id=…&exp=…&sig=…]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:10</Duration>
            <TrackingEvents>
              <Tracking event="start"><![CDATA[https://ads.example.com/track?event=start&…]]></Tracking>
              <Tracking event="firstQuartile"><![CDATA[…]]></Tracking>
              <Tracking event="midpoint"><![CDATA[…]]></Tracking>
              <Tracking event="thirdQuartile"><![CDATA[…]]></Tracking>
              <Tracking event="complete"><![CDATA[…]]></Tracking>
            </TrackingEvents>
            <VideoClicks>
              <ClickThrough><![CDATA[https://example.com/landing]]></ClickThrough>
              <ClickTracking><![CDATA[https://ads.example.com/track?event=click&…]]></ClickTracking>
            </VideoClicks>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4"
                         width="1280" height="720" bitrate="1500">
                <![CDATA[https://cdn.example.com/openadsource/9a8b…/master.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>
```

A no-fill (no candidate, all budgets exhausted, or rate-limited) is the
same shape minus `<Ad>`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2"/>
```

Both responses are `200 OK` `application/xml`. Players treat the empty
form as "no ad available right now" and play the content immediately.

---

## The bundled test player

`examples/test-player/index.html` is a static HTML page wrapping
**video.js** with `videojs-contrib-ads` + `videojs-vast-vpaid`. Run
the stack with the dev overlay and open
<http://localhost:8090> — it requests a VAST tag against the local
adserver, plays the ad if one is returned, then plays the content
fallback.

It's also a useful skeleton: copy `examples/test-player/index.html`
into your project, edit the `adTagUrl` to point at your adserver, and
you have a working integration.

---

## video.js (HTML5)

The most common open-source player. Pair it with
[`videojs-contrib-ads`](https://github.com/videojs/videojs-contrib-ads)
and any VAST plugin (videojs-vast-vpaid, ima-sdk, etc).

```html
<link href="https://vjs.zencdn.net/8.10.0/video-js.css" rel="stylesheet" />
<script src="https://vjs.zencdn.net/8.10.0/video.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/videojs-contrib-ads@7/dist/videojs-contrib-ads.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/videojs-vast-vpaid@2/bin/videojs_5.vast.vpaid.min.js"></script>

<video id="player" class="video-js" controls width="640" height="360"
       data-setup="{}"
       poster="https://cdn.example.com/poster.jpg">
  <source src="https://cdn.example.com/content.m3u8" type="application/x-mpegURL" />
</video>

<script>
  const player = videojs('player');
  player.vastClient({
    adTagUrl: 'https://ads.example.com/vast?pos=pre',
    playAdAlways: true,
    adsCancelTimeout: 5000,
    adsEnabled: true,
    autoResize: true,
  });
</script>
```

Key options:

- `adTagUrl` — the OpenAdSource `/vast` endpoint. Append
  `?pos=pre|mid|post` and any overrides (`country`, `device`,
  `ad_id` for forced selection).
- `playAdAlways: true` — request a new VAST on every content start.
- The plugin handles parsing the `<Impression>`, quartile, and click
  tracking URLs and fires them at the right moments.

---

## hls.js (live + on-demand)

hls.js itself doesn't speak VAST; pair it with a wrapper. With
**Shaka Player**:

```html
<script src="https://cdn.jsdelivr.net/npm/shaka-player@4/dist/shaka-player.ui.min.js"></script>
<video id="v" controls></video>
<script>
  const video = document.getElementById('v');
  const player = new shaka.Player(video);
  player.load('https://cdn.example.com/live.m3u8');

  // Manual VAST request + tracking-pixel fire — Shaka v4 doesn't have a
  // built-in VAST parser, so do a lightweight prefetch + <video> hand-off.
  fetch('https://ads.example.com/vast?pos=pre')
    .then(r => r.text())
    .then(xml => parseAndPlay(xml, video));

  function parseAndPlay(xml, video) {
    const doc = new DOMParser().parseFromString(xml, 'application/xml');
    const mediaURL = doc.querySelector('MediaFile')?.textContent.trim();
    const impressions = [...doc.querySelectorAll('Impression')]
      .map(n => n.textContent.trim());
    if (!mediaURL) return; // no-fill — content plays as-is
    video.src = mediaURL;
    video.play();
    impressions.forEach(u => fetch(u, { mode: 'no-cors' }));
  }
</script>
```

This is the bare-bones pattern when the player library doesn't have its
own VAST integration. For production, use a library — handling
quartile firing, click attribution, and error recovery manually is more
work than it looks.

---

## JW Player

JW handles VAST natively through its `advertising` block:

```js
jwplayer('myDiv').setup({
  playlist: [{
    file: 'https://cdn.example.com/content.m3u8',
    title: 'Demo content',
  }],
  advertising: {
    client: 'vast',
    schedule: {
      preroll: {
        offset: 'pre',
        tag: 'https://ads.example.com/vast?pos=pre',
      },
      midroll: {
        offset: '00:00:30',
        tag: 'https://ads.example.com/vast?pos=mid',
      },
    },
  },
});
```

JW's VAST parser fires quartile + click tracking on its own; nothing to
wire by hand.

---

## Google IMA SDK

For shops that want IMA's full reporting suite, OpenAdSource fits the
SDK's "ad tag URL" slot transparently. With video.js + the
`videojs-ima` plugin:

```html
<script src="//imasdk.googleapis.com/js/sdkloader/ima3.js"></script>
<script src="https://cdn.jsdelivr.net/npm/videojs-ima@2/dist/videojs.ima.min.js"></script>

<script>
  const player = videojs('player');
  player.ima({
    adTagUrl: 'https://ads.example.com/vast?pos=pre',
  });
  player.ima.requestAds();
</script>
```

IMA fetches the tag, parses the VAST, and renders the creative in its
own overlay `<video>` element. The OpenAdSource tracking URLs fire from
IMA, not the page, so make sure your reverse proxy lets IMA hit the
`/track` endpoint cross-origin (the adserver already sends
`Access-Control-Allow-Origin: *`).

---

## CORS

OpenAdSource sets `Access-Control-Allow-Origin: *` on both `/vast` and
`/track`. That's deliberate — every video player either fetches the
VAST tag via XHR or fires the tracking pixels via `<img>` from the
content origin.

If you proxy the adserver behind a domain where this is unacceptable
(e.g. you want to scope CORS to specific origins), terminate CORS in
the reverse proxy and remove the wildcard. The Go code's CORS
middleware is configurable via the `ALLOWED_ORIGINS` env var if you
prefer to set it at the source.

---

## Query parameters available on `/vast`

| Param      | Type                              | Effect                                                                    |
|------------|-----------------------------------|---------------------------------------------------------------------------|
| `pos`      | `pre` (default), `mid`, `post`    | Position-targeted ad pool                                                 |
| `offset`   | int32 (default 0)                 | Selection rotation index — same player can pass `offset=N+1` to get a different ad |
| `country`  | ISO 3166-1 alpha-2                | Override the GeoIP-derived country (useful in test / preview)             |
| `device`   | `desktop` / `mobile` / `ctv` / `tablet` | Override the UA-classified device                                  |
| `ad_id`    | UUID                              | Force a specific ad (skips selection; still subject to freq + budget caps)|

Any unknown parameter is ignored. Standard players will only set the
`pos` param; the rest are dev-side overrides.

See `api.md` for the full contract including status semantics and the
header set.

---

## Click tracking + landing pages

Two distinct URLs are emitted:

- `<ClickThrough>` is the user-facing landing URL. The player opens it
  in a new tab when the user clicks.
- `<ClickTracking>` is the OpenAdSource pixel. Fired by the player as a
  beacon before the navigation happens.

The dashboard stores the `landing_url` on the ad row; that value is
what becomes `<ClickThrough>`. There's no redirect through the adserver
— the click goes straight to the landing URL.

---

## Debugging tips

- The test player at <http://localhost:8090> is the fastest way to
  confirm the round-trip works. Open the browser devtools network tab
  and watch the `/vast` + `/track` requests fly.
- `curl -s 'http://localhost:8088/vast?pos=pre' | xmllint --format -`
  prints a pretty-formatted VAST response. xmllint comes with
  `libxml2-utils` on Debian / Ubuntu.
- VAST validators worth bookmarking:
  <https://adserver.guru/vast_validator/> and
  <https://www.iabtechlab.com/standards/vast/> (the spec).
- If quartile / impression pixels aren't firing, check the player's
  network tab for the `/track` URLs — most player bugs manifest as
  silent skipping rather than 4xx responses.
- The adserver's `oas_track_events_total{event="...",status="ok"}`
  counter is the ground-truth that the events made it to Redis. If
  it's stuck at 0, the problem is on the player side.
