#!/usr/bin/env node

const { execFileSync } = require("child_process");
const fs = require("fs");
const path = require("path");

const ext = process.platform === "win32" ? ".exe" : "";
const bin = path.join(__dirname, "..", "bin", "tmc" + ext);
const args = process.argv.slice(2);

function installBinary() {
  execFileSync(process.execPath, [path.join(__dirname, "install.js")], {
    stdio: "inherit",
    env: { ...process.env, TMC_CLI_RUN: "1" },
  });
}

if (args[0] === "install" || args[0] === "update") {
  installBinary();
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
