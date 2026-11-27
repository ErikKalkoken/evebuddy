# EVE Buddy website

The website is built with [Hugo](https://gohugo.io/) from this folder and deployed to GitHub Pages by the `pages.yml` workflow.

## Local preview

Run from the repository root:

```sh
go run github.com/gohugoio/hugo@v0.170.1 server --source website
```

Download links fall back to the releases page, unless you fetch the latest release data first:

```sh
gh api repos/ErikKalkoken/evebuddy/releases/latest > website/data/release.json
```

## Where to edit

- `content/`: page text in Markdown. The home page text and images are in the front matter of `content/_index.md`.
- `data/footer.yaml`: footer text and links.
- `data/platforms.yaml`: download options per platform.
- `hugo.toml`: site settings, menu and links.
- `static/images/`: screenshots as lossless WebP, e.g. `convert in.png -define webp:lossless=true -define webp:method=6 out.webp`. Exceptions: `social.png` (social preview image, 1200×630) stays PNG; `desktop.webp` (home page hero and repo README) is lossy (`-quality 90`) for a faster home page.
