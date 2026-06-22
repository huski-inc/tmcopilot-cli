#!/usr/bin/env node

const { execFileSync } = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");

const ext = process.platform === "win32" ? ".exe" : "";
const bin = path.join(__dirname, "..", "bin", "tmc" + ext);
const commandName = "tmc" + ext;
const aliasName = "tmcopilot" + ext;
const args = process.argv.slice(2);

function installBinary() {
  execFileSync(process.execPath, [path.join(__dirname, "install.js")], {
    stdio: "inherit",
    env: { ...process.env, TMC_CLI_RUN: "1" },
  });
}

function expandHome(dir) {
  if (dir === "~") return os.homedir();
  if (dir.startsWith("~/")) return path.join(os.homedir(), dir.slice(2));
  return dir;
}

function uniqueDirs(dirs) {
  const seen = new Set();
  return dirs
    .filter(Boolean)
    .map((dir) => path.resolve(expandHome(dir)))
    .filter((dir) => {
      const key = process.platform === "win32" ? dir.toLowerCase() : dir;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });
}

function isNpmExecutionDir(dir) {
  const normalized = path.resolve(dir).toLowerCase();
  return (
    normalized.includes(`${path.sep}node_modules${path.sep}.bin`) ||
    normalized.includes(`${path.sep}.npm${path.sep}_npx${path.sep}`) ||
    normalized.includes(`${path.sep}_npx${path.sep}`) ||
    normalized.includes(`${path.sep}@npmcli${path.sep}run-script${path.sep}`) ||
    normalized.includes(`${path.sep}node_modules${path.sep}npm${path.sep}node_modules${path.sep}`) ||
    normalized.endsWith(`${path.sep}node-gyp-bin`)
  );
}

function canInstallTo(dir) {
  try {
    fs.mkdirSync(dir, { recursive: true });
    fs.accessSync(dir, fs.constants.W_OK);
    return true;
  } catch (_) {
    return false;
  }
}

function fallbackInstallDir() {
  if (process.platform === "win32") {
    const localAppData = process.env.LOCALAPPDATA || path.join(os.homedir(), "AppData", "Local");
    return path.join(localAppData, "Programs", "TMCopilot", "bin");
  }
  return path.join(os.homedir(), ".local", "bin");
}

function npmPrefixBin() {
  const prefix = process.env.npm_config_prefix;
  if (!prefix) return "";
  return process.platform === "win32" ? prefix : path.join(prefix, "bin");
}

function installDirCandidates() {
  const pathDirs = (process.env.PATH || "")
    .split(path.delimiter)
    .filter((dir) => dir && !isNpmExecutionDir(dir));
  const preferred =
    process.platform === "darwin"
      ? [npmPrefixBin(), "/opt/homebrew/bin", "/usr/local/bin", path.join(os.homedir(), ".local", "bin")]
      : process.platform === "win32"
        ? [npmPrefixBin()]
        : [npmPrefixBin(), path.join(os.homedir(), ".local", "bin"), "/usr/local/bin"];
  return uniqueDirs([process.env.TMC_INSTALL_DIR, ...preferred, ...pathDirs, fallbackInstallDir()]);
}

function resolveInstallDir() {
  for (const dir of installDirCandidates()) {
    if (canInstallTo(dir)) return dir;
  }
  throw new Error("no writable install directory found; set TMC_INSTALL_DIR to a writable directory on PATH");
}

function isOnPath(dir) {
  const target = path.resolve(dir);
  return (process.env.PATH || "")
    .split(path.delimiter)
    .filter(Boolean)
    .some((entry) => path.resolve(expandHome(entry)) === target);
}

function installPersistentCommands() {
  const installDir = resolveInstallDir();
  const dest = path.join(installDir, commandName);
  const aliasDest = path.join(installDir, aliasName);

  fs.copyFileSync(bin, dest);
  fs.chmodSync(dest, 0o755);

  fs.rmSync(aliasDest, { force: true });
  if (process.platform === "win32") {
    fs.copyFileSync(dest, aliasDest);
  } else {
    try {
      fs.symlinkSync(commandName, aliasDest);
    } catch (_) {
      fs.copyFileSync(dest, aliasDest);
    }
  }
  fs.chmodSync(aliasDest, 0o755);

  console.log(`Installed tmc and tmcopilot to ${installDir}`);
  if (!isOnPath(installDir)) {
    console.log(`Add this directory to PATH before running tmc or tmcopilot: ${installDir}`);
  }
}

if (args[0] === "install" || args[0] === "update") {
  installBinary();
  installPersistentCommands();
  process.exit(0);
}

if (!fs.existsSync(bin)) {
  try {
    installBinary();
  } catch (_) {
    console.error(
      "\nFailed to auto-install TMCopilot CLI binary.\n" +
        `Run the installer manually:\n  node "${path.join(__dirname, "install.js")}"\n`
    );
    process.exit(1);
  }
}

try {
  execFileSync(bin, args, { stdio: "inherit" });
} catch (err) {
  process.exit(err.status || 1);
}
