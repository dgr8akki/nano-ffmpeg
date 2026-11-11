# nano-ffmpeg Website

The marketing + documentation site for [nano-ffmpeg](https://github.com/dgr8akki/nano-ffmpeg), deployed to [nano-ffmpeg.vercel.app](https://nano-ffmpeg.vercel.app).

It is a small [Next.js](https://nextjs.org) App Router project with two user-facing pages:

- `/` -- landing page (features, operations grid, progress demo, install cards).
- `/docs` -- long-form documentation (screen flow, operations reference, keybindings, configuration, smart defaults).

The site is intentionally content-first: everything renders from data arrays and JSX, no CMS.

## Prerequisites

- Node.js 20 LTS (anything >=18 should work, but CI targets 20).
- `npm` (shipped with Node) or `bun` if you prefer -- `bun.lock` is already in the repo.

## Local development

From the `website/` directory:

```bash
npm install
npm run dev    # http://localhost:3000, hot reload
```

Or with Bun:

```bash
bun install
bun dev
```

You can also run these from the repo root with `npm --prefix website <script>`.

## Scripts

| Script | What it does |
|--------|--------------|
| `npm run dev`   | Start the Next.js dev server on port 3000. |
| `npm run build` | Production build to `.next/`. |
| `npm run start` | Serve the production build. |
| `npm run lint`  | Run ESLint across `app/` and `components/`. Must stay clean -- the root release script and Vercel deploys will fail on lint errors. |

## File layout

```
website/
├── app/
│   ├── layout.tsx        # Root layout + global chrome (Navbar, Footer)
│   ├── page.tsx          # Landing page (features, operations, progress demo, install)
│   ├── docs/page.tsx     # /docs route -- mirrors the project README
│   └── globals.css       # Tailwind base + a few site-wide rules
├── components/
│   ├── Navbar.tsx
│   ├── Footer.tsx
│   └── TerminalDemo.tsx  # Animated Home-screen mock on the landing page
├── public/             # Static assets (favicons, OG images)
├── eslint.config.mjs
├── next.config.ts
├── package.json
└── tsconfig.json
```

## Editing content

There is no CMS -- edits happen directly in the TSX files.

- **Landing page**: edit the `features` and `operations` arrays at the top of [`app/page.tsx`](app/page.tsx). Each feature is `{ icon, title, desc }` (icons come from `lucide-react`); each operation is `{ name, desc }`.
- **Docs page**: edit the `<OpDoc>` entries inside the Operations `Section` in [`app/docs/page.tsx`](app/docs/page.tsx); everything else (Getting Started, Screen Flow, Keybindings, Configuration, Progress Screen, Smart Defaults) is plain JSX inside labelled `<Section>` blocks.
- **Navbar / Footer**: [`components/Navbar.tsx`](components/Navbar.tsx) and [`components/Footer.tsx`](components/Footer.tsx).
- **Styling**: Tailwind utility classes. The accent palette is `#7C3AED` (purple) + `#06B6D4` (cyan); reuse those hexes for new UI so the page stays on-brand.

### Keep `/docs` in sync with the project README

When a CLI flag, operation, keybinding, or config field changes, update **both**:

1. [`../README.md`](../README.md) (the canonical source of truth).
2. [`app/docs/page.tsx`](app/docs/page.tsx) (user-facing docs surface).

> Tip: React's `react/no-unescaped-entities` rule will fail `npm run lint` if you write raw apostrophes in JSX text. Use `&apos;` in copy like `input&apos;s total duration`.

## Deployment

Vercel auto-deploys on every push to `main`. A preview environment is built for each pull request. There is no manual deploy step; merge to `main` is the release mechanism for the site.

If you need to promote a Vercel preview manually, the project is linked under the `dgr8akki` account on Vercel.

## Contributing

Same as the root project (see [`../README.md#contributing`](../README.md#contributing)). In short: branch from `main`, run `npm run lint` before opening the PR, and keep the docs page lined up with any behavioral changes in the Go code.
