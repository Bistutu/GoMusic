# GoMusic web

Vue 3 + TypeScript + Vite. Native controls keep this small tool simple; no global UI library or external font/advertising script is loaded.

Node.js 22.12+:

```sh
npm ci
npm run dev
```

`/api` proxies to the Go API on port 8081. Build with `npm run build`; the Go server serves `dist` on the same origin. See [deployment](../DEPLOYMENT.md), [API](../docs/API.md), and [reference decisions](../docs/REFERENCES.md).
