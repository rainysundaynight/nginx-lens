from __future__ import annotations

from fastapi import FastAPI, HTTPException, Query
from fastapi.responses import JSONResponse

from agent_api.data import collect_snapshot
from utils.version import get_version


def create_app() -> FastAPI:
    app = FastAPI(title="nginx-lens-agent", version=get_version())

    @app.get("/healthz")
    def healthz():
        return {"ok": True}

    @app.get("/version")
    def version():
        return {"version": get_version()}

    @app.get("/snapshot")
    def snapshot(config_path: str | None = Query(default=None, description="Path to nginx.conf")):
        try:
            return JSONResponse(collect_snapshot(config_path=config_path))
        except Exception as e:
            raise HTTPException(status_code=500, detail=str(e))

    return app


app = create_app()

