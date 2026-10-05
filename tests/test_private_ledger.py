import csv
import json
import os
import tempfile
import unittest
from pathlib import Path

from private_ledger import append_event, import_binance_csv, verify_ledger


class PrivateLedgerTests(unittest.TestCase):
    def test_append_and_verify_hash_chain(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "private" / "ledger.jsonl"
            append_event(path, "paper_signal", {"event_id": "signal-1", "action": "HOLD"})
            append_event(path, "paper_fill", {"event_id": "fill-1", "side": "BUY"})
            self.assertEqual(verify_ledger(path)["records"], 2)

    def test_tampering_is_detected(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "ledger.jsonl"
            append_event(path, "paper_fill", {"side": "BUY"})
            record = json.loads(path.read_text(encoding="utf-8"))
            record["payload"]["side"] = "SELL"
            path.write_text(json.dumps(record) + "\n", encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "integrity"):
                verify_ledger(path)

    def test_binance_csv_import_is_separate_and_idempotent(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "export.csv"
            ledger = Path(directory) / "ledger.jsonl"
            with source.open("w", newline="", encoding="utf-8") as file:
                writer = csv.writer(file)
                writer.writerow(["Date(UTC)", "Pair", "Side", "Price", "Executed", "Trade ID", "Fee"])
                writer.writerow(["2025-01-01 00:00:00", "BTCUSDT", "BUY", "100", "0.01", "123", "0.001"])
            first = import_binance_csv(source, ledger)
            second = import_binance_csv(source, ledger)
            self.assertEqual(first["imported"], 1)
            self.assertEqual(second["imported"], 0)
            record = json.loads(ledger.read_text(encoding="utf-8").splitlines()[0])
            self.assertEqual(record["event_type"], "external_trade_import")
            self.assertIn("unverified", record["payload"]["source"])
            self.assertEqual(verify_ledger(ledger)["records"], 1)

    def test_invalid_csv_does_not_partially_import(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "export.csv"
            ledger = Path(directory) / "ledger.jsonl"
            source.write_text(
                "Date(UTC),Pair,Side,Price,Executed,Trade ID\n"
                "2025-01-01,BTCUSDT,BUY,100,0.01,1\n"
                "2025-01-02,BTCUSDT,SELL,bad,0.01,2\n",
                encoding="utf-8",
            )
            with self.assertRaisesRegex(ValueError, "numeric price"):
                import_binance_csv(source, ledger)
            self.assertFalse(ledger.exists())

    def test_duplicate_event_id_is_idempotent(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "ledger.jsonl"
            first = append_event(path, "paper_signal", {"event_id": "same", "action": "HOLD"})
            second = append_event(path, "paper_signal", {"event_id": "same", "action": "HOLD"})
            self.assertEqual(first["hash"], second["hash"])
            self.assertEqual(verify_ledger(path)["records"], 1)

    def test_conflicting_duplicate_event_id_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "ledger.jsonl"
            append_event(path, "paper_signal", {"event_id": "same", "action": "HOLD"})
            with self.assertRaisesRegex(ValueError, "Conflicting event"):
                append_event(path, "paper_signal", {"event_id": "same", "action": "BUY"})

    def test_financial_record_cannot_be_written_inside_repository(self):
        from private_ledger import REPOSITORY_ROOT

        with self.assertRaisesRegex(ValueError, "inside the repository"):
            append_event(REPOSITORY_ROOT / "should-not-exist.jsonl", "paper_signal", {})

    @unittest.skipUnless(os.name == "posix", "POSIX permission bits only")
    def test_ledger_files_are_owner_only(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "private" / "ledger.jsonl"
            append_event(path, "paper_signal", {})
            self.assertEqual(path.stat().st_mode & 0o777, 0o600)
            self.assertEqual(path.parent.stat().st_mode & 0o777, 0o700)


if __name__ == "__main__":
    unittest.main()
