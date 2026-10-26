import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).resolve().parents[2] / "scripts" / "run-e2e.sh"


class RunE2EReadinessTest(unittest.TestCase):
    def run_pipeline(self, mode):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            scripts = root / ".github" / "scripts"
            scripts.mkdir(parents=True)
            (root / "e2e").mkdir()
            shutil.copy(SCRIPT, scripts / SCRIPT.name)
            binaries = root / "bin"
            binaries.mkdir()
            commands = {
                "docker": '''#!/usr/bin/env python3
import os
import sys
from pathlib import Path

args = sys.argv[1:]
root = Path(os.environ["READINESS_TEST_ROOT"])
if "logs" in args and "billing-worker" in args and "--timestamps" not in args:
    probe = root / "probes"
    calls = int(probe.read_text()) if probe.exists() else 0
    probe.write_text(str(calls + 1))
    # Duplicates and another topic must not satisfy the four-partition check.
    for partition in (0, 0, 1, 2):
        print(f"consumer/broker/0 added subscription to om_sys.api_events/{partition}")
    print("consumer/broker/0 added subscription to another_topic/3")
    if calls > 0 and os.environ["READINESS_TEST_MODE"] == "ready":
        print("consumer/broker/0 added subscription to om_sys.api_events/3")
''',
                "make": '''#!/usr/bin/env python3
import os
import sys
from pathlib import Path

root = Path(os.environ["READINESS_TEST_ROOT"])
(root / "make_target").write_text(sys.argv[1])
''',
                "curl": "#!/bin/sh\nexit 0\n",
                "sleep": "#!/bin/sh\nexit 0\n",
            }
            for name, source in commands.items():
                executable = binaries / name
                executable.write_text(source)
                executable.chmod(0o755)
            result = subprocess.run(
                ["bash", str(scripts / SCRIPT.name), "credits-disabled", "test-image", "never", "upstream"],
                env={
                    **os.environ,
                    "PATH": str(binaries) + os.pathsep + os.environ["PATH"],
                    "READINESS_TEST_ROOT": str(root),
                    "READINESS_TEST_MODE": mode,
                },
                capture_output=True,
                text=True,
                timeout=30,
            )
            target = root / "make_target"
            return result, int((root / "probes").read_text()), target.read_text() if target.exists() else None

    def test_waits_for_every_partition_before_starting_tests(self):
        result, probes, target = self.run_pipeline("ready")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(probes, 2)
        self.assertEqual(target, "test-credits-disabled")

    def test_missing_partition_fails_without_running_tests(self):
        result, probes, target = self.run_pipeline("missing")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(probes, 60)
        self.assertIsNone(target)
        self.assertIn("did not initialize all four system-event partitions", result.stderr)


if __name__ == "__main__":
    unittest.main()
