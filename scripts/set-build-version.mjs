import { readFile, writeFile } from "node:fs/promises";
import { resolve } from "node:path";

const version = process.argv[2];

if (!/^(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)$/.test(version ?? "")) {
  throw new Error("Usage: node scripts/set-build-version.mjs <major.minor.patch>");
}

const files = [
  ["build/config.yml", /productVersion: \d+\.\d+\.\d+/, `productVersion: ${version}`],
  ["build/linux/nfpm/nfpm.yaml", /version: "\d+\.\d+\.\d+"/, `version: "${version}"`],
  ["build/windows/info.json", /"file_version": "\d+\.\d+\.\d+"/, `"file_version": "${version}"`],
  ["build/windows/info.json", /"ProductVersion": "\d+\.\d+\.\d+"/, `"ProductVersion": "${version}"`],
];

for (const [file, pattern, replacement] of files) {
  const path = resolve(file);
  const content = await readFile(path, "utf8");

  if (!pattern.test(content)) {
    throw new Error(`Could not set the version in ${file}`);
  }

  await writeFile(path, content.replace(pattern, replacement));
}

console.log(`Set build metadata version to ${version}`);
