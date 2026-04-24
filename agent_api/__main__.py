from __future__ import annotations

import argparse
import os
import sys


def main(argv: list[str] | None = None) -> None:
    parser = argparse.ArgumentParser(prog="nginx-lens-agent")
    parser.add_argument("--host", default=os.environ.get("NGINX_LENS_AGENT_HOST", "0.0.0.0"))
    parser.add_argument("--port", type=int, default=int(os.environ.get("NGINX_LENS_AGENT_PORT", "8088")))
    parser.add_argument(
        "--config",
        dest="config_path",
        default=os.environ.get("NGINX_LENS_NGINX_CONF"),
        help="Path to nginx.conf (optional; defaults from nginx-lens config).",
    )
    parser.add_argument(
        "--reload",
        action="store_true",
        help="Enable auto-reload (dev only).",
    )
    args = parser.parse_args(argv)

    from uvicorn import run
    from agent_api.app import create_app

    app = create_app()
    if args.config_path:
        # Pass default via env; endpoint can still override via query param.
        os.environ.setdefault("NGINX_LENS_NGINX_CONF", args.config_path)

    run(app, host=args.host, port=args.port, reload=args.reload)


if __name__ == "__main__":
    main(sys.argv[1:])

