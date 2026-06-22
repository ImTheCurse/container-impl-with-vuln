import os
import socket
import subprocess
import shlex
from pathlib import Path
from typing import Any

from fastapi import FastAPI, Query
from fastapi.responses import HTMLResponse, PlainTextResponse


app = FastAPI(title="Video Converter RCE Discovery Demo")


BASE_DIR = Path(__file__).resolve().parent

VIDEO_WORKER_DIR = Path("/tmp/video-worker-demo")
UPLOADS_DIR = VIDEO_WORKER_DIR / "uploads"
OUTPUTS_DIR = VIDEO_WORKER_DIR / "outputs"

DEMO_VIDEO_FILE = UPLOADS_DIR / "video.mp4"


def ensure_demo_files() -> None:
    VIDEO_WORKER_DIR.mkdir(parents=True, exist_ok=True)
    UPLOADS_DIR.mkdir(parents=True, exist_ok=True)
    OUTPUTS_DIR.mkdir(parents=True, exist_ok=True)

    if not DEMO_VIDEO_FILE.exists():
        DEMO_VIDEO_FILE.write_text(
            "This is a fake demo video file.\n",
            encoding="utf-8",
        )


HOME_HTML = """
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <title>Video Converter RCE Discovery Demo</title>
    <style>
      body {
        font-family: Arial, sans-serif;
        background: #111827;
        color: #e5e7eb;
        padding: 40px;
      }

      .card {
        max-width: 900px;
        margin: auto;
        background: #1f2937;
        padding: 30px;
        border-radius: 16px;
      }

      a {
        color: #60a5fa;
      }

      code {
        background: #374151;
        padding: 3px 6px;
        border-radius: 6px;
      }

      .warning {
        color: #fbbf24;
      }
    </style>
  </head>

  <body>
    <div class="card">
      <h1>Video Converter RCE Discovery Demo</h1>

      <p>
        This demo shows how an attacker can discover a vulnerable runtime endpoint
        through leaked debug information inside a generated video conversion file.
      </p>

      <p>
        Open:
        <a href="/video/convert">/video/convert</a>
      </p>

      <p class="warning">
        Demo only. Do not expose this kind of endpoint in a real server.
      </p>
    </div>
  </body>
</html>
"""


VIDEO_CONVERT_HTML = """
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <title>Video Converter</title>
    <style>
      body {
        font-family: Arial, sans-serif;
        background: #0f172a;
        color: #e5e7eb;
        padding: 40px;
      }

      .card {
        max-width: 1000px;
        margin: auto;
        background: #1e293b;
        padding: 30px;
        border-radius: 16px;
      }

      button {
        background: #2563eb;
        color: white;
        border: none;
        padding: 12px 18px;
        border-radius: 8px;
        cursor: pointer;
        font-size: 16px;
      }

      pre {
        background: #020617;
        color: #d1d5db;
        padding: 20px;
        border-radius: 12px;
        overflow-x: auto;
        min-height: 220px;
      }

      a {
        color: #60a5fa;
      }

      code {
        background: #334155;
        padding: 3px 6px;
        border-radius: 6px;
      }

      .hint {
        color: #fbbf24;
      }
    </style>
  </head>

  <body>
    <div class="card">
      <h1>Video Converter</h1>

      <p>
        Normal user flow: convert a video to GIF.
      </p>

      <p>
        Normal request:
        <code>/api/video/convert?fmt=gif</code>
      </p>

      <button onclick="convertVideo()">Convert Video to GIF</button>

      <p class="hint">
        After conversion, open the generated output file and inspect its content.
      </p>

      <h2>Server Response</h2>
      <pre id="output">Waiting...</pre>
    </div>

    <script>
      async function convertVideo() {
        const response = await fetch('/api/video/convert?fmt=gif');
        const data = await response.json();

        document.getElementById('output').textContent =
          JSON.stringify(data, null, 2);
      }
    </script>
  </body>
</html>
"""


@app.get("/", response_class=HTMLResponse)
def home() -> str:
    return HOME_HTML


@app.get("/video/convert", response_class=HTMLResponse)
def video_convert_page() -> str:
    return VIDEO_CONVERT_HTML


@app.get("/api/video/convert")
def video_convert(fmt: str = Query("gif", min_length=1)) -> dict[str, Any]:
    ensure_demo_files()

    output_file = OUTPUTS_DIR / f"converted.{fmt}.txt"

    output_file.write_text(
        "FAKE VIDEO CONVERSION OUTPUT\n"
        "============================\n"
        f"input_file: {DEMO_VIDEO_FILE}\n"
        f"output_format: {fmt}\n"
        "status: conversion completed\n"
        "\n"
        "DEBUG INFORMATION - SHOULD NOT BE PUBLIC\n"
        "----------------------------------------\n"
        "worker_hostname_check: /api/runtime/hostname\n"
        "worker_directory_check: /api/runtime/pwd\n"
        "\n"
        "Developer note:\n"
        "The runtime endpoint executes the word that appears after /api/runtime/.\n",
        encoding="utf-8",
    )

    return {
        "status": "ok",
        "message": "Video conversion completed successfully.",
        "output_file": str(output_file),
        "download_output": f"/outputs/{output_file.name}",
        "normal_user_note": "A normal user would only download the converted file.",
        "security_note": (
            "The converted file accidentally leaks internal debug endpoints. "
            "An attacker can inspect the file and discover /api/runtime/{word}."
        ),
    }


@app.get("/outputs/{filename}", response_class=PlainTextResponse)
def read_output_file(filename: str) -> str:
    ensure_demo_files()

    requested_file = OUTPUTS_DIR / filename

    if not requested_file.exists():
        return "File not found."

    return requested_file.read_text(encoding="utf-8")


@app.get("/api/runtime/{word:path}")
def vulnerable_command(word: str) -> dict[str, Any]:
    command_parts = shlex.split(word)

    result = subprocess.run(
        command_parts,
        capture_output=True,
        text=True
    )

    return {
        "command": word,
        "command_parts": command_parts,
        "stdout": result.stdout,
        "stderr": result.stderr,
        "returncode": result.returncode,
        "server_runtime": {
            "hostname": socket.gethostname(),
            "pid": os.getpid(),
            "uid": os.getuid(),
            "gid": os.getgid(),
            "current_working_directory": os.getcwd(),
            "base_dir": str(BASE_DIR),
        },
    }