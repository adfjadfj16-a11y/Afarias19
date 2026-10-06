import hashlib
import json
import os
import sqlite3
import traceback
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional


def _utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


def _hash256(code: str, message: str, timestamp: str, prev_hash: str = "") -> str:
    raw = f"{code}{message}{timestamp}{prev_hash}".encode()
    return hashlib.sha256(raw).hexdigest()


@dataclass
class ErrorRecord:
    code: str
    message: str
    timestamp: str
    hash256: str
    prev_hash: str
    caller_stack: str
    environment: str


class ErrorRegistry:
    def __init__(self, db_path: str = "audit/error_registry.sqlite") -> None:
        self.db_path = db_path
        os.makedirs(os.path.dirname(db_path) or ".", exist_ok=True)
        self.conn = sqlite3.connect(db_path)
        self.conn.execute(
            """
            CREATE TABLE IF NOT EXISTS error_registry (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                code TEXT NOT NULL,
                message TEXT NOT NULL,
                timestamp TEXT NOT NULL,
                hash256 TEXT NOT NULL,
                prev_hash TEXT NOT NULL,
                caller_stack TEXT NOT NULL,
                environment TEXT NOT NULL
            )
            """
        )
        self.conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_error_registry_code_ts ON error_registry(code, timestamp)"
        )
        self.conn.commit()

    def register(self, code: str, message: str, environment: Optional[Dict[str, Any]] = None) -> ErrorRecord:
        timestamp = _utc_now()
        prev_hash = self._last_hash()
        caller_stack = "".join(traceback.format_stack(limit=10))
        env = environment or {}
        safe_env = {k: v for k, v in env.items() if "secret" not in k.lower() and "token" not in k.lower()}
        environment_json = json.dumps(safe_env, sort_keys=True)
        hash256 = _hash256(code, message, timestamp, prev_hash)
        self.conn.execute(
            "INSERT INTO error_registry(code, message, timestamp, hash256, prev_hash, caller_stack, environment) VALUES (?, ?, ?, ?, ?, ?, ?)",
            (code, message, timestamp, hash256, prev_hash, caller_stack, environment_json),
        )
        self.conn.commit()
        return ErrorRecord(code, message, timestamp, hash256, prev_hash, caller_stack, environment_json)

    def audit_trail(self, error_code: str) -> List[Dict[str, Any]]:
        rows = self.conn.execute(
            "SELECT code, message, timestamp, hash256, prev_hash, caller_stack, environment FROM error_registry WHERE code = ? ORDER BY timestamp ASC",
            (error_code,),
        ).fetchall()
        trail = []
        for row in rows:
            code, message, timestamp, hash256, prev_hash, caller_stack, environment = row
            trail.append(
                {
                    "code": code,
                    "message": message,
                    "timestamp": timestamp,
                    "hash256": hash256,
                    "prev_hash": prev_hash,
                    "verified": _hash256(code, message, timestamp, prev_hash) == hash256,
                    "caller_stack": caller_stack,
                    "environment": json.loads(environment),
                }
            )
        return trail

    def export_json(self, output_path: str) -> str:
        rows = self.conn.execute(
            "SELECT code, message, timestamp, hash256, prev_hash, caller_stack, environment FROM error_registry ORDER BY id ASC"
        ).fetchall()
        payload = []
        for row in rows:
            payload.append(
                {
                    "code": row[0],
                    "message": row[1],
                    "timestamp": row[2],
                    "hash256": row[3],
                    "prev_hash": row[4],
                    "caller_stack": row[5],
                    "environment": json.loads(row[6]),
                }
            )
        os.makedirs(os.path.dirname(output_path) or ".", exist_ok=True)
        with open(output_path, "w", encoding="utf-8") as fh:
            json.dump(payload, fh, indent=2, sort_keys=True)
        return output_path

    def verify_chain(self) -> bool:
        rows = self.conn.execute(
            "SELECT code, message, timestamp, hash256, prev_hash FROM error_registry ORDER BY id ASC"
        ).fetchall()
        previous = ""
        for code, message, timestamp, hash256, prev_hash in rows:
            if prev_hash != previous:
                return False
            if _hash256(code, message, timestamp, prev_hash) != hash256:
                return False
            previous = hash256
        return True

    def _last_hash(self) -> str:
        row = self.conn.execute("SELECT hash256 FROM error_registry ORDER BY id DESC LIMIT 1").fetchone()
        return row[0] if row else ""
