#!/usr/bin/env node
// Multica Board Codex sidebar patcher.
// - Locates ChatGPT.app / Codex.app on this machine.
// - Backs up app.asar + app.asar.unpacked before touching anything.
// - Applies route/nav, CSP, and webview partition patches.
// - Aborts (leaving the app untouched) if any anchor is missing.
// - `--undo` restores the most recent backup.
import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, writeFileSync, cpSync, rmSync, readdirSync, statSync, copyFileSync, mkdirSync } from "node:fs";
import { tmpdir, homedir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const appCandidates = [
  process.env.MULTICA_BOARD_CODEX_APP,
  "/Applications/ChatGPT.app",
  "/Applications/Codex.app",
  `${homedir()}/Applications/ChatGPT.app`,
  `${homedir()}/Applications/Codex.app`,
].filter(Boolean);

function args() {
  const out = { webUrl: "http://127.0.0.1:13000", home: `${homedir()}/Library/Application Support/Multica Board`, undo: false };
  const argv = process.argv.slice(2);
  for (let i = 0; i < argv.length; i++) {
    if (argv[i] === "--web-url") out.webUrl = argv[i + 1];
    if (argv[i] === "--home") out.home = argv[i + 1];
    if (argv[i] === "--undo") out.undo = true;
  }
  return out;
}

function findApp() {
  for (const candidate of appCandidates) {
    const asar = path.join(candidate, "Contents/Resources/app.asar");
    if (existsSync(asar)) return { appPath: candidate, asar };
  }
  throw new Error("Codex app not found. Set MULTICA_BOARD_CODEX_APP to the .app bundle path.");
}

function findAsarCli() {
  const candidates = [
    path.join(scriptDir, "node_modules/.bin/asar"),
    path.join(scriptDir, "../node_modules/.bin/asar"),
    path.join("/Users/zlearn/.multica-board/multica/node_modules/.bin/asar"),
  ];
  for (const c of candidates) if (existsSync(c)) return c;
  return null;
}

function runAsar(args) {
  // Prefer the runtime Node's npx so the patch tool works without bundling the
  // asar dependency tree. Falls back to a bundled CLI when one is present.
  try {
    execFileSync("npx", ["--yes", "@electron/asar@3.4.1", ...args], { stdio: "pipe" });
    return;
  } catch {}
  const cli = findAsarCli();
  if (!cli) throw new Error("@electron/asar is not available. Run with Node/npx on PATH.");
  execFileSync(cli, args, { stdio: "pipe" });
}

function sha256(file) {
  return createHash("sha256").update(readFileSync(file)).digest("hex");
}

function walk(dir, out = []) {
  for (const entry of readdirSync(dir)) {
    const p = path.join(dir, entry);
    const st = statSync(p);
    if (st.isDirectory()) walk(p, out);
    else out.push(p);
  }
  return out;
}

function findFile(root, needle) {
  for (const file of walk(root)) {
    try {
      if (readFileSync(file, "utf8").includes(needle)) return file;
    } catch {}
  }
  return null;
}

function patchText(file, from, to, label) {
  const src = readFileSync(file, "utf8");
  const count = src.split(from).length - 1;
  if (count !== 1) throw new Error(`${label}: expected 1 occurrence of anchor, found ${count}`);
  writeFileSync(file, src.replace(from, to));
  console.log(`patched: ${label}`);
}

async function applyPatch({ asar, unpacked, webUrl, workDir }) {
  runAsar(["extract", asar, path.join(workDir, "extracted")]);
  if (existsSync(unpacked)) {
    const topLevels = readdirSync(unpacked);
    for (const name of topLevels) {
      cpSync(path.join(unpacked, name), path.join(workDir, "extracted", name), { recursive: true });
    }
  }

  const root = path.join(workDir, "extracted");

  // Bridge used by the embedded Multica Board webview. Sandboxed preloads can
  // require electron, so the board can ask the host app to open a Codex thread
  // through the OS protocol handler instead of navigating the webview to a
  // codex:// URL (which Chromium blocks with "content blocked").
  writeFileSync(
    path.join(root, "webview/multica-board-preload.js"),
    `const { contextBridge, ipcRenderer } = require("electron");
contextBridge.exposeInMainWorld("multicaBoard", {
  openCodexThread: (threadId) =>
    ipcRenderer.invoke("multica-board:open-codex-thread", String(threadId)),
});
`,
    "utf8",
  );

  const ipcAnchor = "function oB({reconcileBrowserStorageId";
  const ipcFile = findFile(path.join(root, ".vite/build"), ipcAnchor);
  if (!ipcFile) throw new Error("Unsupported Codex build: IPC bridge anchor not found.");
  if (!readFileSync(ipcFile, "utf8").includes("multica-board:open-codex-thread")) {
    patchText(
      ipcFile,
      ipcAnchor,
      `try{l.ipcMain.removeHandler("multica-board:open-codex-thread")}catch{}l.ipcMain.handle("multica-board:open-codex-thread",async(e,t)=>{if(typeof t==="string"&&/^[0-9a-fA-F-]{8,}$/.test(t)){try{await l.shell.openExternal("codex://threads/"+t);return true}catch{return false}}return false});${ipcAnchor}`,
      "IPC bridge",
    );
  }

  const routeAnchor =
    'path:`/plugins`,element:(0,D7.jsx)(vhc,{})})]})]}),null,';
  const routeFile = findFile(root, routeAnchor);
  if (!routeFile) throw new Error("Unsupported Codex build: route anchor not found.");
  if (!readFileSync(routeFile, "utf8").includes("/task-board")) {
    patchText(
      routeFile,
      routeAnchor,
      `path:\`/task-board\`,element:(0,D7.jsx)(\`webview\`,{src:\`${webUrl}\`,className:\`h-full w-full border-0\`,allowpopups:\`true\`,partition:\`persist:multica-board\`})},${routeAnchor}`,
      "route/nav",
    );
  }

  const navAnchor = 'description:`Nav link that opens the skills page`})}):null,n&&r===`codex`?';
  const navFile = findFile(root, navAnchor);
  if (!navFile) throw new Error("Unsupported Codex build: nav anchor not found.");
  if (!readFileSync(navFile, "utf8").includes("任务看板")) {
    patchText(
      navFile,
      navAnchor,
      'description:`Nav link that opens the skills page`})}):null,n&&(r===`codex`||c)?(0,M8.jsx)(T8,{icon:KR,onClick:()=>{o(`/task-board`)},isActive:s.pathname.startsWith(`/task-board`),label:`任务看板`}):null,n&&r===`codex`?',
      "sidebar nav",
    );
  }

  const mainAnchor =
    "let m=(t,a,s)=>{if(n.Ra(s.partition)!=null||jR(s)||Rz(s))return;";
  const mainFile = findFile(path.join(root, ".vite/build"), mainAnchor);
  if (!mainFile) throw new Error("Unsupported Codex build: webview handler anchor not found.");
  if (!readFileSync(mainFile, "utf8").includes("multica-board-preload.js")) {
    patchText(
      mainFile,
      mainAnchor,
      "let m=(t,a,s)=>{if(n.Ra(s.partition)!=null||jR(s)||Rz(s))return;if(s.partition===`persist:multica-board`){a.partition=`persist:multica-board`,a.session=l.session.fromPartition(`persist:multica-board`),a.preload=require(\"node:path\").join(__dirname,\"../../webview/multica-board-preload.js\"),a.nodeIntegration=!1,a.nodeIntegrationInSubFrames=!1,a.contextIsolation=!0,a.sandbox=!0,a.webSecurity=!0,a.devTools=!0,a.webviewTag=!1;return}",
      "webview partition",
    );
  }

  const htmlFile = path.join(root, "webview/index.html");
  if (!existsSync(htmlFile)) throw new Error("Unsupported Codex build: webview/index.html not found.");
  const html = readFileSync(htmlFile, "utf8");
  if (!html.includes(webUrl)) {
    for (const from of ["child-src &#39;self&#39; blob:", "frame-src &#39;self&#39; blob:"]) {
      if (!html.includes(from)) throw new Error("Unsupported Codex build: CSP anchor not found.");
    }
    writeFileSync(htmlFile, html.replaceAll("child-src &#39;self&#39; blob:", `child-src &#39;self&#39; ${webUrl} blob:`).replaceAll("frame-src &#39;self&#39; blob:", `frame-src &#39;self&#39; ${webUrl} blob:`));
    console.log("patched: CSP");
  }

  const unpackDirs = existsSync(unpacked) ? readdirSync(unpacked).join(",") : "";
  const args = ["pack", path.join(workDir, "extracted"), path.join(workDir, "app.asar.new")];
  if (unpackDirs) args.push("--unpack-dir", `{${unpackDirs}}`);
  runAsar(args);

  copyFileSync(path.join(workDir, "app.asar.new"), asar);
  const newUnpacked = path.join(workDir, "app.asar.new.unpacked");
  if (existsSync(newUnpacked)) {
    rmSync(unpacked, { recursive: true, force: true });
    cpSync(newUnpacked, unpacked, { recursive: true });
  }
}

function main() {
  const opts = args();
  const { appPath, asar } = findApp();
  const unpacked = `${asar}.unpacked`;
  if (!opts.undo && process.platform === "darwin") {
    let running = "";
    try {
      running = execFileSync("pgrep", ["-f", appPath], { stdio: "pipe", encoding: "utf8" }).trim();
    } catch {}
    if (running) throw new Error("Please quit Codex/ChatGPT before patching.");
  }

  const backupRoot = path.join(opts.home, "patch-backups");
  mkdirSync(backupRoot, { recursive: true });

  if (opts.undo) {
    const backups = readdirSync(backupRoot).filter((n) => existsSync(path.join(backupRoot, n, "app.asar"))).sort();
    const latest = backups.at(-1);
    if (!latest) throw new Error("No patch backup found.");
    const dir = path.join(backupRoot, latest);
    copyFileSync(path.join(dir, "app.asar"), asar);
    if (existsSync(path.join(dir, "app.asar.unpacked"))) {
      rmSync(unpacked, { recursive: true, force: true });
      cpSync(path.join(dir, "app.asar.unpacked"), unpacked, { recursive: true });
    }
    console.log(`restored backup ${latest}`);
    return;
  }

  const stamp = new Date().toISOString().replace(/[:.]/g, "-");
  const backupDir = path.join(backupRoot, stamp);
  mkdirSync(backupDir, { recursive: true });
  copyFileSync(asar, path.join(backupDir, "app.asar"));
  if (existsSync(unpacked)) cpSync(unpacked, path.join(backupDir, "app.asar.unpacked"), { recursive: true });
  writeFileSync(path.join(backupDir, "manifest.json"), JSON.stringify({ appPath, asarSha256: sha256(asar), at: new Date().toISOString() }, null, 2));

  const workDir = mkdtempSync(path.join(tmpdir(), "multica-board-patch-"));
  try {
    applyPatch({ asar, unpacked, webUrl: opts.webUrl, workDir });
    console.log(`patched ${appPath} (backup: ${backupDir})`);
  } finally {
    rmSync(workDir, { recursive: true, force: true });
  }
}

try {
  main();
} catch (e) {
  console.error(`✗ ${e.message}`);
  process.exit(1);
}
