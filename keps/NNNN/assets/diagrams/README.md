Mermaid diagram rendering

This directory is intended to store rendered diagram assets (SVG/PNG) for the KEP.

How to render locally:

1. Ensure Node.js and npm are installed (tested on Node 16+).
2. Install mermaid CLI (if not installed):
   - npm install -g @mermaid-js/mermaid-cli
3. Install a Chromium binary for Puppeteer (required by mermaid CLI):
   - npx puppeteer@latest install
   - or, if you prefer a smaller runtime, try: npx puppeteer@latest browsers install chrome-headless-shell
4. Render a diagram:
   - mmdc -i /path/to/diagram.mmd -o /path/to/diagram.svg

Notes:
- The KEP currently contains Mermaid code blocks in `0000-csi-direct-env-injection.md`. If you need images for the enhancement PR, run the commands above locally and add the generated files under this directory (e.g., `assets/diagrams/diagram_1.svg`).
- Rendering failed in this environment due to a missing Chromium binary; the instructions above should work on developer machines with network access to download Chromium.
