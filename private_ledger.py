"""Local, credential-free, tamper-evident ledger for paper and imported trades."""

import csv
import hashlib
import json
import os
import re
from datetime import datetime, timezone
from pathlib import Path


REPOSITORY_ROOT = Path(__file__).resolve().parent


def default_data_dir():
    state_home = Path(os.environ.get("XDG_STATE_HOME", Path.home() / ".local" / "state"))
    return state_home / "afarias19" / "binance"


def secure_mkdir(path):
    path = Path(path)
    path.mkdir(parents=True, exist_ok=True, mode=0o700)
    if os.name == "posix":
        os.chmod(path, 0o700)


def ensure_outside_repository(path):
    resolved = Path(path).expanduser().resolve()
    try:
        resolved.relative_to(REPOSITORY_ROOT)
    except ValueError:
        return resolved
    raise ValueError(
        f"Refusing to store personal financial records inside the repository: {resolved}"
    )


def _canonical_json(value):
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"), allow_nan=False)


def _record_hash(record):
    contents = {key: value for key, value in record.items() if key != "hash"}
    return hashlib.sha256(_canonical_json(contents).encode("utf-8")).hexdigest()


def verify_ledger(path):
    path = Path(path)
    if not path.exists():
        return {"valid": True, "records": 0, "last_hash": "0" * 64}

    previous_hash = "0" * 64
    count = 0
    with path.open(encoding="utf-8") as ledger:
        for line_number, line in enumerate(ledger, 1):
            try:
                record = json.loads(line)
            except json.JSONDecodeError as error:
                raise ValueError(f"Invalid ledger JSON on line {line_number}.") from error
            if not isinstance(record, dict):
                raise ValueError(f"Invalid ledger record on line {line_number}.")
            if record.get("sequence") != count + 1:
                raise ValueError(f"Invalid ledger sequence on line {line_number}.")
            if record.get("previous_hash") != previous_hash:
                raise ValueError(f"Broken ledger chain on line {line_number}.")
            if record.get("hash") != _record_hash(record):
                raise ValueError(f"Ledger integrity check failed on line {line_number}.")
            previous_hash = record["hash"]
            count += 1
    return {"valid": True, "records": count, "last_hash": previous_hash}


def append_event(path, event_type, payload):
    if not isinstance(event_type, str) or not event_type or not isinstance(payload, dict):
        raise ValueError("Ledger events require an event type and object payload.")
    path = ensure_outside_repository(path)
    secure_mkdir(path.parent)
    status = verify_ledger(path)
    event_id = payload.get("event_id")
    if event_id and path.exists():
        with path.open(encoding="utf-8") as ledger:
            for line in ledger:
                existing = json.loads(line)
                existing_payload = existing.get("payload", {})
                if existing_payload.get("event_id") == event_id:
                    if existing["event_type"] != event_type or existing_payload != payload:
                        raise ValueError(f"Conflicting event already exists in ledger: {event_id}")
                    return existing
    record = {
        "sequence": status["records"] + 1,
        "timestamp_utc": datetime.now(timezone.utc).isoformat(),
        "event_type": event_type,
        "payload": payload,
        "previous_hash": status["last_hash"],
    }
    record["hash"] = _record_hash(record)
    encoded = (_canonical_json(record) + "\n").encode("utf-8")
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600)
    try:
        if os.name == "posix":
            os.fchmod(descriptor, 0o600)
        with os.fdopen(descriptor, "ab", closefd=False) as ledger:
            ledger.write(encoded)
            ledger.flush()
            os.fsync(ledger.fileno())
    finally:
        os.close(descriptor)
    return record


def _normalized_headers(fieldnames):
    return {
        re.sub(r"[^a-z0-9]", "", (name or "").lower()): name
        for name in fieldnames or []
    }


def _column(headers, *aliases):
    for alias in aliases:
        key = re.sub(r"[^a-z0-9]", "", alias.lower())
        if key in headers:
            return headers[key]
    raise ValueError(f"Trade export is missing a required column: {aliases[0]}.")


def _decimal_text(value, field):
    value = (value or "").strip().split()[0].replace(",", "")
    if not re.fullmatch(r"(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?", value):
        raise ValueError(f"Invalid numeric {field} in Binance CSV export.")
    number = float(value)
    if not (number > 0 and number < float("inf")):
        raise ValueError(f"Invalid positive {field} in Binance CSV export.")
    return value


def import_binance_csv(source_path, ledger_path):
    source_path = Path(source_path)
    contents = source_path.read_bytes()
    if len(contents) > 25 * 1024 * 1024:
        raise ValueError("Binance CSV export exceeds the 25 MiB safety limit.")
    source_hash = hashlib.sha256(contents).hexdigest()
    try:
        decoded = contents.decode("utf-8-sig")
    except UnicodeDecodeError as error:
        raise ValueError("Binance CSV export must be UTF-8 encoded.") from error
    reader = csv.DictReader(decoded.splitlines())
    headers = _normalized_headers(reader.fieldnames)
    columns = {
        "trade_id": _column(headers, "Trade ID", "ID"),
        "timestamp": _column(headers, "Date(UTC)", "Date", "Timestamp", "Time"),
        "symbol": _column(headers, "Pair", "Symbol"),
        "side": _column(headers, "Side", "Type"),
        "price": _column(headers, "Price"),
        "quantity": _column(headers, "Executed", "Quantity", "Qty"),
    }
    optional = {}
    for key, aliases in (
        ("fee", ("Fee", "Commission")),
        ("fee_asset", ("Fee Coin", "Fee Asset", "Commission Asset")),
    ):
        try:
            optional[key] = _column(headers, *aliases)
        except ValueError:
            optional[key] = None

    existing_ids = set()
    if Path(ledger_path).exists():
        verify_ledger(ledger_path)
        with Path(ledger_path).open(encoding="utf-8") as ledger:
            for line in ledger:
                record = json.loads(line)
                if record.get("event_type") == "external_trade_import":
                    event_id = record.get("payload", {}).get("event_id")
                    if event_id is not None:
                        existing_ids.add(str(event_id))

    imported = 0
    skipped_duplicates = 0
    pending_records = []
    pending_ids = set()
    for row_number, row in enumerate(reader, 2):
        if row_number > 100_001:
            raise ValueError("Binance CSV export exceeds the 100,000-row safety limit.")
        trade_id = (row.get(columns["trade_id"]) or "").strip()
        symbol = (row.get(columns["symbol"]) or "").strip().upper()
        side = (row.get(columns["side"]) or "").strip().upper()
        timestamp = (row.get(columns["timestamp"]) or "").strip()
        if not trade_id or not symbol or side not in {"BUY", "SELL"} or not timestamp:
            raise ValueError(f"Invalid trade identity or timestamp on CSV line {row_number}.")
        if len(trade_id) > 128 or not re.fullmatch(r"[A-Z0-9_-]+", symbol):
            raise ValueError(f"Invalid trade ID or symbol on CSV line {row_number}.")
        event_id = f"binance:{symbol}:{trade_id}"
        if event_id in existing_ids or event_id in pending_ids:
            skipped_duplicates += 1
            continue
        payload = {
            "event_id": event_id,
            "source": "manual_binance_csv_export_unverified",
            "source_sha256": source_hash,
            "trade_id": trade_id,
            "timestamp": timestamp,
            "symbol": symbol,
            "side": side,
            "price": _decimal_text(row.get(columns["price"]), "price"),
            "quantity": _decimal_text(row.get(columns["quantity"]), "quantity"),
        }
        if optional["fee"]:
            fee_value = (row.get(optional["fee"]) or "").strip()
            if fee_value:
                payload["fee"] = fee_value
        if optional["fee_asset"]:
            payload["fee_asset"] = (row.get(optional["fee_asset"]) or "").strip()
        pending_records.append(payload)
        pending_ids.add(event_id)
    for payload in pending_records:
        append_event(ledger_path, "external_trade_import", payload)
        imported += 1
    return {
        "imported": imported,
        "skipped_duplicates": skipped_duplicates,
        "source_sha256": source_hash,
        "integrity": verify_ledger(ledger_path),
        "note": "Imported CSV rows are not independently authenticated by Binance.",
    }
