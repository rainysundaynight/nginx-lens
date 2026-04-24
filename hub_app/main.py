from __future__ import annotations

import os
from typing import Any, Dict, List

import httpx
from fastapi import FastAPI, HTTPException
from fastapi.responses import HTMLResponse, JSONResponse

from utils.version import get_version


def create_app() -> FastAPI:
    app = FastAPI(title="nginx-lens-hub", version=get_version())

    @app.get("/healthz")
    def healthz():
        return {"ok": True}

    @app.get("/version")
    def version():
        return {"version": get_version()}

    @app.get("/", response_class=HTMLResponse)
    def index():
        return HTMLResponse(
            "<html><body><h3>nginx-lens-hub</h3><p>API is up. UI build is served in Docker image.</p></body></html>"
        )

    @app.get("/api/agents")
    def agents():
        raw = os.environ.get("NGINX_LENS_AGENTS", "http://localhost:8088")
        items = [x.strip() for x in raw.split(",") if x.strip()]
        return {"agents": items}

    @app.get("/api/snapshots")
    async def snapshots() -> JSONResponse:
        raw = os.environ.get("NGINX_LENS_AGENTS", "http://localhost:8088")
        agents = [x.strip() for x in raw.split(",") if x.strip()]
        if not agents:
            return JSONResponse({"agents": [], "snapshots": []})

        out: List[Dict[str, Any]] = []
        async with httpx.AsyncClient(timeout=10.0) as client:
            for a in agents:
                url = a.rstrip("/") + "/snapshot"
                try:
                    r = await client.get(url)
                    r.raise_for_status()
                    out.append({"agent": a, "snapshot": r.json()})
                except Exception as e:
                    out.append({"agent": a, "error": str(e)})

        return JSONResponse({"agents": agents, "results": out})

    @app.get("/api/proxy")
    async def proxy(url: str) -> JSONResponse:
        # Very small helper for UI experiments; DO NOT expose to untrusted networks.
        async with httpx.AsyncClient(timeout=10.0) as client:
            try:
                r = await client.get(url)
                r.raise_for_status()
                return JSONResponse({"status_code": r.status_code, "json": r.json()})
            except httpx.HTTPError as e:
                raise HTTPException(status_code=502, detail=str(e))

    return app


app = create_app()

