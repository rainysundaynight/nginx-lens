from __future__ import annotations

import argparse
import os
import sys


def main(argv: list[str] | None = None) -> None:
    parser = argparse.ArgumentParser(prog="nginx-lens-hub")
    parser.add_argument("--host", default=os.environ.get("NGINX_LENS_HUB_HOST", "0.0.0.0"))
    parser.add_argument("--port", type=int, default=int(os.environ.get("NGINX_LENS_HUB_PORT", "8089")))
    parser.add_argument(
        "--agents",
        default=os.environ.get("NGINX_LENS_AGENTS", "http://localhost:8088"),
        help="Comma-separated list of agent base URLs.",
    )
    parser.add_argument("--reload", action="store_true", help="Enable auto-reload (dev only).")
    args = parser.parse_args(argv)

    os.environ["NGINX_LENS_AGENTS"] = args.agents

    from uvicorn import run
    from hub_app.main import create_app

    app = create_app()
    run(app, host=args.host, port=args.port, reload=args.reload)


if __name__ == "__main__":
    main(sys.argv[1:])

