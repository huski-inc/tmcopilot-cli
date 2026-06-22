#!/usr/bin/env node

const crypto = require("crypto");
const fs = require("fs");
const os = require("os");
const path = require("path");
const { execFileSync } = require("child_process");

const pkg = require("../package.json");

const REPO = process.env.TMC_REPO || "huski-inc/tmcopilot-cli";
const NAME = "tmc";
const VERSION = (process.env.TMC_VERSION || pkg.version).replace(/^v/, "");
const TAG = process.env.TMC_RELEASE_TAG || `v${VERSION}`;

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

const platform = PLATFORM_MAP[process.platform];
const arch = ARCH_MAP[process.arch];
const isWindows = process.platform === "win32";
const archiveExt = isWindows ? ".zip" : ".tar.gz";
const binaryName = NAME + (isWindows ? ".exe" : "");
const archiveName = `${NAME}-${VERSION}-${platform}-${arch}${archiveExt}`;
const releaseBaseURL = `https://github.com/${REPO}/releases/download/${TAG}`;
const archiveURL = `${releaseBaseURL}/${archiveName}`;
const checksumsURL = `${releaseBaseURL}/checksums.txt`;
const binDir = path.join(__dirname, "..", "bin");
const dest = path.join(binDir, binaryName);

if (!platform || !arch) {
  console.error(`Unsupported platform: ${process.platform}-${process.arch}`);
  process.exit(1);
}

if (process.env.npm_command === "exec" && !process.env.TMC_CLI_RUN) {
  process.exit(0);
}

function curlArgs(url, output) {
  const args = [
    "--fail",
    "--location",
    "--silent",
    "--show-error",
    "--connect-timeout",
    "10",
    "--max-time",
    "120",
    "--max-redirs",
    "3",
    "--output",
    output,
    url,
  ];
  return args;
}

function download(url, output) {
  execFileSync("curl", curlArgs(url, output), { stdio: ["ignore", "ignore", "pipe"] });
}

function readExpectedChecksum(checksumsPath, targetArchiveName) {
  const content = fs.readFileSync(checksumsPath, "utf8");
  for (const line of content.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    const parts = trimmed.split(/\s+/);
    if (parts.length >= 2 && parts[1] === targetArchiveName) {
      return parts[0];
    }
  }
  throw new Error(`checksum entry not found for ${targetArchiveName}`);
}

function sha256(filePath) {
  const hash = crypto.createHash("sha256");
  const fd = fs.openSync(filePath, "r");
  try {
    const buf = Buffer.alloc(64 * 1024);
    let n;
    while ((n = fs.readSync(fd, buf, 0, buf.length, null)) > 0) {
      hash.update(buf.subarray(0, n));
    }
  } finally {
    fs.closeSync(fd);
  }
  return hash.digest("hex");
}

function verifyChecksum(archivePath, checksumsPath) {
  const expected = readExpectedChecksum(checksumsPath, archiveName);
  const actual = sha256(archivePath);
  if (actual.toLowerCase() !== expected.toLowerCase()) {
    throw new Error(`checksum mismatch for ${archiveName}: expected ${expected}, got ${actual}`);
  }
}

function extractZipWindows(archivePath, destDir) {
  const env = { ...process.env, TMC_ARCHIVE: archivePath, TMC_DEST: destDir };
  const args = ["-NoProfile", "-ExecutionPolicy", "Bypass", "-Command"];
  const script =
    "$ErrorActionPreference='Stop';" +
    "Expand-Archive -LiteralPath $env:TMC_ARCHIVE -DestinationPath $env:TMC_DEST -Force";
  try {
    execFileSync("powershell.exe", [...args, script], { stdio: "inherit", env });
  } catch (primaryErr) {
    try {
      execFileSync("tar", ["-xf", archivePath, "-C", destDir], { stdio: "inherit" });
    } catch (fallbackErr) {
      throw new Error(
        `failed to extract ${archivePath}: PowerShell failed (${primaryErr.message}); tar failed (${fallbackErr.message})`
      );
    }
  }
}

function install() {
  fs.mkdirSync(binDir, { recursive: true });
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "tmc-cli-"));
  const archivePath = path.join(tmpDir, archiveName);
  const checksumsPath = path.join(tmpDir, "checksums.txt");

  try {
    download(archiveURL, archivePath);
    download(checksumsURL, checksumsPath);
    verifyChecksum(archivePath, checksumsPath);

    if (isWindows) {
      extractZipWindows(archivePath, tmpDir);
    } else {
      execFileSync("tar", ["-xzf", archivePath, "-C", tmpDir], { stdio: "inherit" });
    }

    const extractedBinary = path.join(tmpDir, binaryName);
    fs.copyFileSync(extractedBinary, dest);
    fs.chmodSync(dest, 0o755);
    console.log(`${NAME} v${VERSION} installed successfully`);
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }
}

try {
  install();
} catch (err) {
  console.error(`Failed to install ${NAME}: ${err.message}`);
  console.error(
    `\nYou can download the archive manually from:\n  ${archiveURL}\n` +
      "Then place the extracted binary on your PATH."
  );
  process.exit(1);
}
