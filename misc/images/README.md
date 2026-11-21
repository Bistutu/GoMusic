# README screenshots

These screenshots show the live GoMusic v0.0.1 interface at <https://music.unmeta.cn>, captured on 2026-11-15 in headless Chrome.

| File                     | Language | Browser width | Content                          |
| ------------------------ | -------- | ------------: | -------------------------------- |
| `gomusic-en-desktop.png` | English  |       1120 px | Desktop playlist form and result |
| `gomusic-zh-desktop.png` | Chinese  |       1120 px | Desktop playlist form and result |
| `gomusic-en-mobile.png`  | English  |        390 px | Mobile playlist form and result  |
| `gomusic-zh-mobile.png`  | Chinese  |        390 px | Mobile playlist form and result  |

The input is the public Kugou code `40725013`. Both desktop and mobile requests returned all 12 of the platform's reported 12 tracks. The images include the header, controls and export buttons; the editor scrolls to show the remaining tracks. They are actual browser captures, with no mock API responses or visual redesign applied to the images.

When refreshing these assets, use the current running site, verify the API's fetched and total counts, switch the UI language, and capture through the end of the export panel. Check that mobile controls are visible and the page has no horizontal overflow. Verify that clipboard and TXT contents match the editor without line numbers. Update both language pairs together.

The website and README files use the same [WeChat appreciation image](../../static/src/assets/approve.jpg).
