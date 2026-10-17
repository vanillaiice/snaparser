#!/usr/bin/env python3
"""Build versioned, self-contained release archives without publishing them."""

import argparse
import hashlib
import os
from pathlib import Path
import re
import shutil
import subprocess
import tarfile
import tempfile
import zipfile

ROOT = Path(__file__).resolve().parent.parent
TARGETS = [(system, arch) for system in ("linux", "darwin", "windows") for arch in ("amd64", "arm64")]
FILES = ["README.md", "LICENSE", "CHANGELOG.md", "RELEASING.md", "snaparser.example.toml", "frontend/README.md", "frontend/assets/HTMX-LICENSE"]


def main():
    version = re.search(r'^const version = "([0-9]+\.[0-9]+\.[0-9]+)"$', (ROOT / "cmd/snaparser/main.go").read_text(), re.MULTILINE)
    if version is None:
        raise SystemExit("Cannot determine the release version from cmd/snaparser/main.go")

    version = version.group(1)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", default=os.environ.get("RELEASE_VERSION", version), help="expected version; must match the CLI")
    parser.add_argument("--output", type=Path, default=ROOT / "dist" / f"v{version}")
    args = parser.parse_args()

    if args.version.removeprefix("v") != version:
        parser.error(f"requested version {args.version!r} does not match the CLI version {version!r}")

    args.output.mkdir(parents=True, exist_ok=True)
    archives = []

    for system, arch in TARGETS:
        name = f"snaparser_{version}_{system}_{arch}"
        print(f"Building {name}", flush=True)

        with tempfile.TemporaryDirectory(prefix="snaparser-release-") as temporary:
            staging = Path(temporary) / name
            staging.mkdir()
            binary = staging / ("snaparser.exe" if system == "windows" else "snaparser")
            environment = dict(os.environ, CGO_ENABLED="0", GOOS=system, GOARCH=arch)

            subprocess.run(
                ["go", "build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-ldflags=-s -w", "-o", str(binary), "./cmd/snaparser"],
                cwd=ROOT, env=environment, check=True,
            )

            for source in FILES:
                destination = staging / source
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(ROOT / source, destination)

            if system == "windows":
                archive = args.output / f"{name}.zip"
                with zipfile.ZipFile(archive, "w", compression=zipfile.ZIP_DEFLATED) as output:
                    for path in sorted(staging.rglob("*")):
                        if path.is_file():
                            output.write(path, path.relative_to(staging.parent))
            else:
                archive = args.output / f"{name}.tar.gz"
                with tarfile.open(archive, "w:gz") as output:
                    output.add(staging, arcname=name)

            archives.append(archive)
            print(f"Created {archive.name}", flush=True)

    with (args.output / "SHA256SUMS").open("w", encoding="utf-8", newline="\n") as checksums:
        for archive in sorted(archives):
            digest = hashlib.sha256(archive.read_bytes()).hexdigest()
            checksums.write(f"{digest}  {archive.name}\n")

    print(f"Release archives and SHA256SUMS are ready in {args.output}", flush=True)


if __name__ == "__main__":
    main()
