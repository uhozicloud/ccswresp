// postinstall.js - Runs after npm install to set up config
import { existsSync, mkdirSync, copyFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { dirname } from "node:path";

const __dirname = dirname(fileURLToPath(import.meta.url));
const configDir = resolve(
  process.env.HOME || process.env.USERPROFILE || "",
  ".ccswresp"
);
const configPath = resolve(configDir, ".env");
const templatePath = resolve(__dirname, "..", "env_example");

// Only create config if it doesn't exist (don't overwrite user settings)
if (!existsSync(configPath)) {
  mkdirSync(configDir, { recursive: true });
  if (existsSync(templatePath)) {
    copyFileSync(templatePath, configPath);
    console.log("  Config created at ~/.ccswresp/.env");
  }
}

console.log("  ccswresp installed! Run 'ccswresp --help' to get started.");
