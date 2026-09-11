#!/usr/bin/env python3
"""Batch snapshot runner. Does not change geometry.py."""
from __future__ import annotations

import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from geometry import load, snapshot  # noqa: E402


def main() -> None:
    if len(sys.argv) < 2:
        raise SystemExit("Usage: python runner.py scenario.json [t_s]")
    scenario = load(sys.argv[1])
    if len(sys.argv) >= 3:
        t = float(sys.argv[2])
        print(json.dumps(snapshot(scenario, t), ensure_ascii=False, allow_nan=False))
        return
    env = scenario["environment"]
    step = int(env["step_s"])
    horizon = int(env["horizon_s"])
    for t in range(0, horizon, step):
        print(json.dumps(snapshot(scenario, t), ensure_ascii=False, allow_nan=False))


if __name__ == "__main__":
    main()
