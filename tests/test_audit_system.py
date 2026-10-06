import json
from pathlib import Path

from legalivypolyci import ErrorRegistry


def test_register_10k_errors_and_verify_chain(tmp_path: Path):
    registry = ErrorRegistry(str(tmp_path / "registry.sqlite"))
    for i in range(10000):
        registry.register(f"ERR-{i % 25:03d}", f"public message {i}", {"service": "self-check", "iteration": i})
    trail = registry.audit_trail("ERR-005")
    assert len(trail) == 400
    assert all(item["verified"] for item in trail)
    assert registry.verify_chain() is True


def test_export_json_and_recovery(tmp_path: Path):
    db_path = tmp_path / "registry.sqlite"
    export_path = tmp_path / "audit.json"
    registry = ErrorRegistry(str(db_path))
    registry.register("ERR-CRIT", "critical public message", {"service": "self-check", "secret": "omit"})
    registry.export_json(str(export_path))
    payload = json.loads(export_path.read_text())
    assert payload[0]["code"] == "ERR-CRIT"
    assert "secret" not in payload[0]["environment"]

    reopened = ErrorRegistry(str(db_path))
    assert reopened.verify_chain() is True
    reopened.conn.execute("UPDATE error_registry SET message = 'tampered' WHERE code = 'ERR-CRIT'")
    reopened.conn.commit()
    assert reopened.verify_chain() is False
