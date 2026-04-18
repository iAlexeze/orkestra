import base64
import os
from pathlib import Path

from flask import Flask, render_template_string

app = Flask(__name__)

REGION      = os.environ.get("REGION", "unknown")
PORT        = os.environ.get("PORT", "8080")        # service port — shown on the page
LISTEN_PORT = int(os.environ.get("LISTEN_PORT", "8080"))  # container listen port — fixed

# Load logo once at startup — embedded as a data URI so no static route needed.
_logo_path = Path(__file__).parent / "logo.png"
LOGO_DATA_URI = (
    "data:image/png;base64," + base64.b64encode(_logo_path.read_bytes()).decode()
    if _logo_path.exists()
    else ""
)

REGION_META = {
    "us-east-1":      {"flag": "🇺🇸", "label": "US East (N. Virginia)"},
    "eu-west-1":      {"flag": "🇮🇪", "label": "EU West (Ireland)"},
    "ap-southeast-1": {"flag": "🇸🇬", "label": "Asia Pacific (Singapore)"},
}

TEMPLATE = """
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Orkestra — {{ region }}</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

    body {
      font-family: "Inter", "Segoe UI", Arial, sans-serif;
      background: #0f1117;
      color: #e2e8f0;
      min-height: 100vh;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      gap: 28px;
      padding: 24px;
    }

    .header {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 10px;
    }

    .header img {
      width: 56px;
      height: 56px;
      object-fit: contain;
      border-radius: 10px;
    }

    .logo-text {
      font-size: 13px;
      letter-spacing: 0.18em;
      text-transform: uppercase;
      color: #64748b;
      font-weight: 600;
    }

    .card {
      background: #1e2330;
      border: 1px solid #2d3448;
      border-radius: 16px;
      padding: 40px 48px;
      max-width: 520px;
      width: 100%;
      text-align: center;
      box-shadow: 0 8px 32px rgba(0,0,0,0.4);
    }

    .flag { font-size: 52px; line-height: 1; margin-bottom: 14px; }

    .region-name {
      font-size: 26px;
      font-weight: 700;
      color: #f1f5f9;
      margin-bottom: 6px;
    }

    .region-label {
      font-size: 14px;
      color: #64748b;
      margin-bottom: 28px;
    }

    .divider {
      border: none;
      border-top: 1px solid #2d3448;
      margin-bottom: 24px;
    }

    .meta-row {
      display: flex;
      justify-content: space-between;
      align-items: center;
      font-size: 13px;
      color: #94a3b8;
      margin-bottom: 10px;
    }

    .meta-row span:last-child {
      font-family: "Menlo", "Consolas", monospace;
      color: #7dd3fc;
    }

    .badge {
      display: inline-block;
      background: #14532d;
      color: #86efac;
      border: 1px solid #16a34a;
      border-radius: 99px;
      font-size: 11px;
      font-weight: 600;
      padding: 3px 12px;
      letter-spacing: 0.06em;
      margin-top: 20px;
    }

    footer {
      font-size: 12px;
      color: #334155;
    }

    footer a { color: #475569; text-decoration: none; }
    footer a:hover { color: #94a3b8; }
  </style>
</head>
<body>

  <div class="header">
    {% if logo %}
      <img src="{{ logo }}" alt="Orkestra" />
    {% endif %}
    <div class="logo-text">Orkestra Multi‑Region Demo</div>
  </div>

  <div class="card">
    <div class="flag">{{ flag }}</div>
    <div class="region-name">{{ region }}</div>
    <div class="region-label">{{ label }}</div>

    <hr class="divider" />

    <div class="meta-row">
      <span>Region</span>
      <span>{{ region }}</span>
    </div>
    <div class="meta-row">
      <span>Port</span>
      <span>{{ port }}</span>
    </div>
    <div class="meta-row">
      <span>Managed by</span>
      <span>Orkestra operatorBox</span>
    </div>

    <div class="badge">● RUNNING</div>
  </div>

  <footer>
    Deployed declaratively &mdash;
    <a href="https://github.com/orkspace/orkestra" target="_blank">orkspace/orkestra</a>
  </footer>

</body>
</html>
"""

@app.route("/")
def index():
    meta = REGION_META.get(REGION, {"flag": "🌍", "label": REGION})
    return render_template_string(
        TEMPLATE,
        region=REGION,
        port=PORT,
        flag=meta["flag"],
        label=meta["label"],
        logo=LOGO_DATA_URI,
    )

@app.route("/healthz")
def healthz():
    return {"status": "ok", "region": REGION, "port": PORT}, 200

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=LISTEN_PORT)
