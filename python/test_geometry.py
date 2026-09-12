import copy
import math
import unittest
from pathlib import Path
from unittest.mock import patch

import numpy as np

import geometry


class GeometryContractTest(unittest.TestCase):
    def setUp(self):
        self.scenario = geometry.load(Path(__file__).resolve().parents[1] / "data/01_full_constellation.json")

    def test_known_orbit_orientation_and_earth_rotation(self):
        s = self.scenario
        s["design"]["planes"] = [{"id": "p", "raan_deg": 0, "phase_deg": 0}]
        s["design"]["satellites"] = [{"id": "s", "plane_id": "p", "slot_deg": 90, "launch_batch": 1}]
        s["environment"]["inclination_deg"] = 90
        _, inertial, fixed = geometry.positions(s, 0)
        np.testing.assert_allclose(inertial[0], [0, 0, 6921], atol=1e-9)
        np.testing.assert_allclose(fixed[0], [0, 0, 6921], atol=1e-9)
        s["design"]["satellites"][0]["slot_deg"] = 0
        s["environment"]["earth_angle0_deg"] = 90
        _, _, fixed = geometry.positions(s, 0)
        np.testing.assert_allclose(fixed[0], [0, -6921, 0], atol=1e-9)

    def test_strict_isl_distance_and_earth_obstruction(self):
        s = self.scenario
        s["design"]["satellites"] = [copy.deepcopy(s["design"]["satellites"][0]) for _ in range(2)]
        for i, sat in enumerate(s["design"]["satellites"]):
            sat["id"] = str(i)
        s["environment"]["isl_range_km"] = 2000
        for positions, expected in [
            ([[7000, 0, 0], [7000, 1999, 0]], True),
            ([[7000, 0, 0], [7000, 2000, 0]], False),
            ([[6371, -500, 0], [6371, 500, 0]], False),
            ([[6300, -500, 0], [6300, 500, 0]], False),
        ]:
            xyz = np.array(positions, dtype=float)
            with patch.object(geometry, "positions", return_value=(["0", "1"], xyz, xyz)):
                snap = geometry.snapshot(s, 0)
            linked = any(set(edge[:2]) == {"0", "1"} for edge in snap["edges"])
            self.assertEqual(expected, linked)

    def test_orbit_radius_and_half_open_outage_boundaries(self):
        s = self.scenario
        sat_id = s["design"]["satellites"][0]["id"]
        s["failures"] = [{"satellite_id": sat_id, "start_s": 120.5, "end_s": 240.5}]
        for t, active in [(120.499, True), (120.5, False), (240.499, False), (240.5, True)]:
            snap = geometry.snapshot(s, t)
            sat = next(x for x in snap["satellites"] if x["id"] == sat_id)
            self.assertEqual(active, sat["active"])
            radius = math.sqrt(sum(sat[k] ** 2 for k in ("x_km", "y_km", "z_km")))
            self.assertAlmostEqual(radius, 6921, places=8)
            if not active:
                self.assertFalse(any(sat_id in edge[:2] for edge in snap["edges"]))


if __name__ == "__main__":
    unittest.main()
