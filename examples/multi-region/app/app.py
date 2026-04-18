import os
from flask import Flask, render_template_string

app = Flask(__name__)

TEMPLATE = """
<!DOCTYPE html>
<html>
<head>
    <title>Orkestra Multi‑Region Demo</title>
    <style>
        body { font-family: Arial, sans-serif; padding: 40px; }
        .box {
            padding: 20px;
            border: 2px solid #333;
            display: inline-block;
            border-radius: 8px;
        }
    </style>
</head>
<body>
    <h1>Orkestra Multi‑Region Demo</h1>
    <div class="box">
        <h2>Region: {{ region }}</h2>
        <p>This instance is running in <strong>{{ region }}</strong>.</p>
    </div>
</body>
</html>
"""

@app.route("/")
def index():
    region = os.environ.get("REGION", "unknown")
    return render_template_string(TEMPLATE, region=region)

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)
