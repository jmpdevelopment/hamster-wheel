import fs from "node:fs/promises";
import inspector from "node:inspector";
import path from "node:path";
import { fileURLToPath } from "node:url";

const SOURCE_EXTENSIONS = new Set([".ts", ".tsx", ".js", ".jsx"]);
const COVERAGE_STATE_KEY = "__hwVitestCoverageState__";

function getCoverageState() {
  if (!globalThis[COVERAGE_STATE_KEY]) {
    globalThis[COVERAGE_STATE_KEY] = {
      session: null,
      started: false,
    };
  }
  return globalThis[COVERAGE_STATE_KEY];
}

function toPct(covered, total) {
  if (total === 0) {
    return 100;
  }
  return Number(((covered / total) * 100).toFixed(1));
}

function overlaps(lineStart, lineEnd, range) {
  return range.end > lineStart && range.start < lineEnd;
}

function isSourceFile(filePath, srcRoot) {
  if (!filePath) {
    return false;
  }
  if (!filePath.startsWith(srcRoot + path.sep) && filePath !== srcRoot) {
    return false;
  }

  const ext = path.extname(filePath);
  if (!SOURCE_EXTENSIONS.has(ext)) {
    return false;
  }
  if (
    filePath.endsWith(".d.ts") ||
    filePath.includes(".test.") ||
    filePath.includes(".spec.") ||
    filePath.endsWith(path.join("src", "test-setup.ts"))
  ) {
    return false;
  }
  return true;
}

function normalizeScriptPath(rawURL, root) {
  if (!rawURL) {
    return null;
  }

  const clean = rawURL.split("?")[0].split("#")[0];
  if (clean.startsWith("/@fs/")) {
    return path.normalize(clean.slice("/@fs".length));
  }
  if (clean.startsWith("file://")) {
    try {
      return path.normalize(fileURLToPath(clean));
    } catch {
      return null;
    }
  }

  if (path.isAbsolute(clean)) {
    return path.normalize(clean);
  }

  if (clean.startsWith("/src/")) {
    return path.normalize(path.join(root, clean.slice(1)));
  }

  if (clean.startsWith("http://") || clean.startsWith("https://")) {
    try {
      const parsed = new URL(clean);
      if (parsed.pathname.startsWith("/@fs/")) {
        return path.normalize(parsed.pathname.slice("/@fs".length));
      }
      if (parsed.pathname.startsWith("/src/")) {
        return path.normalize(path.join(root, parsed.pathname.slice(1)));
      }
    } catch {
      return null;
    }
  }

  return null;
}

function extractScriptCoverage(payload) {
  if (Array.isArray(payload)) {
    return payload;
  }
  if (payload && Array.isArray(payload.result)) {
    return payload.result;
  }
  if (payload && Array.isArray(payload.coverage)) {
    return payload.coverage;
  }
  if (payload && typeof payload === "object") {
    const values = Object.values(payload);
    if (values.every((v) => v && typeof v === "object" && Array.isArray(v.functions))) {
      return values;
    }
  }
  return [];
}

function buildLineRanges(source) {
  const lines = source.split(/\r?\n/);
  const ranges = [];
  let offset = 0;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const start = offset;
    const end = offset + line.length + 1;
    ranges.push({ start, end });
    offset = end;
  }

  if (ranges.length === 0) {
    ranges.push({ start: 0, end: 0 });
  }

  return ranges;
}

async function listSourceFiles(srcRoot) {
  const files = [];

  async function walk(dir) {
    const entries = await fs.readdir(dir, { withFileTypes: true });
    for (const entry of entries) {
      const fullPath = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        await walk(fullPath);
        continue;
      }
      if (isSourceFile(fullPath, srcRoot)) {
        files.push(path.normalize(fullPath));
      }
    }
  }

  try {
    await walk(srcRoot);
  } catch {
    return [];
  }

  files.sort();
  return files;
}

function summarizeFile(filePath, source, ranges, functionHits, root) {
  const lineRanges = buildLineRanges(source);
  const coveredRanges = ranges.filter((r) => r.count > 0 && r.end > r.start);

  let coveredLines = 0;
  for (const lineRange of lineRanges) {
    if (coveredRanges.some((r) => overlaps(lineRange.start, lineRange.end, r))) {
      coveredLines++;
    }
  }

  const totalLines = lineRanges.length;
  const totalFunctions = functionHits.length;
  const coveredFunctions = functionHits.filter(Boolean).length;

  const lines = {
    total: totalLines,
    covered: coveredLines,
    pct: toPct(coveredLines, totalLines),
  };
  const functions = {
    total: totalFunctions,
    covered: coveredFunctions,
    pct: toPct(coveredFunctions, totalFunctions),
  };
  const statements = {
    total: totalLines,
    covered: coveredLines,
    pct: toPct(coveredLines, totalLines),
  };
  const branches = {
    total: totalLines,
    covered: coveredLines,
    pct: toPct(coveredLines, totalLines),
  };

  return {
    file: path.relative(root, filePath).replaceAll(path.sep, "/"),
    lines,
    functions,
    statements,
    branches,
  };
}

function sumCoverage(fileSummaries, key) {
  const total = fileSummaries.reduce((acc, file) => acc + file[key].total, 0);
  const covered = fileSummaries.reduce((acc, file) => acc + file[key].covered, 0);
  return { total, covered, pct: toPct(covered, total) };
}

function logSummary(summary) {
  const totals = summary.totals;
  console.log("");
  console.log("Coverage summary (lite provider):");
  console.log(`  lines:      ${totals.lines.pct}% (${totals.lines.covered}/${totals.lines.total})`);
  console.log(
    `  functions:  ${totals.functions.pct}% (${totals.functions.covered}/${totals.functions.total})`
  );
  console.log(
    `  statements: ${totals.statements.pct}% (${totals.statements.covered}/${totals.statements.total})`
  );
  console.log(
    `  branches:   ${totals.branches.pct}% (${totals.branches.covered}/${totals.branches.total})`
  );
}

function post(method, params = {}) {
  return new Promise((resolve, reject) => {
    const state = getCoverageState();
    if (!state.session) {
      resolve({});
      return;
    }
    state.session.post(method, params, (err, result) => {
      if (err) {
        reject(err);
        return;
      }
      resolve(result ?? {});
    });
  });
}

async function startCoverage() {
  const state = getCoverageState();
  if (state.started) {
    return;
  }

  state.session = new inspector.Session();
  state.session.connect();
  await post("Profiler.enable");
  await post("Profiler.startPreciseCoverage", {
    callCount: true,
    detailed: true,
  });
  state.started = true;
}

async function takeCoverage() {
  const state = getCoverageState();
  if (!state.started) {
    return null;
  }
  const result = await post("Profiler.takePreciseCoverage");
  return result.result ?? [];
}

async function stopCoverage() {
  const state = getCoverageState();
  if (!state.started || !state.session) {
    return;
  }

  try {
    await post("Profiler.stopPreciseCoverage");
  } catch {
    // Best effort cleanup.
  }
  try {
    await post("Profiler.disable");
  } catch {
    // Best effort cleanup.
  }

  state.session.disconnect();
  state.session = null;
  state.started = false;
}

class LiteV8CoverageProvider {
  name = "custom-v8-lite";

  constructor() {
    this.ctx = null;
    this.options = null;
    this.fileData = new Map();
    this.receivedSuites = 0;
    this.receivedScripts = 0;
  }

  async initialize(ctx) {
    this.ctx = ctx;
    this.options = ctx.config.coverage;
    this.fileData.clear();
  }

  resolveOptions() {
    return this.options;
  }

  async clean(clean) {
    this.fileData.clear();
    this.receivedSuites = 0;
    this.receivedScripts = 0;
    const reportsDirectory =
      this.options?.reportsDirectory ?? path.join(this.ctx.config.root, "coverage");

    if (clean && reportsDirectory) {
      await fs.rm(reportsDirectory, { recursive: true, force: true });
    }
    await fs.mkdir(reportsDirectory, { recursive: true });
  }

  async onAfterSuiteRun(meta) {
    const coveragePayload = extractScriptCoverage(meta?.coverage);
    if (coveragePayload.length === 0) {
      return;
    }
    this.receivedSuites++;
    this.receivedScripts += coveragePayload.length;

    const srcRoot = path.join(this.ctx.config.root, "src");
    for (const script of coveragePayload) {
      const scriptPath = normalizeScriptPath(script.url, this.ctx.config.root);
      if (!isSourceFile(scriptPath, srcRoot)) {
        continue;
      }

      let entry = this.fileData.get(scriptPath);
      if (!entry) {
        entry = { ranges: [], functionHits: [] };
        this.fileData.set(scriptPath, entry);
      }

      for (const fn of script.functions ?? []) {
        const ranges = fn.ranges ?? [];
        entry.functionHits.push(ranges.some((range) => Number(range.count) > 0));
        for (const range of ranges) {
          entry.ranges.push({
            start: Number(range.startOffset) || 0,
            end: Number(range.endOffset) || 0,
            count: Number(range.count) || 0,
          });
        }
      }
    }
  }

  async generateCoverage() {
    const root = this.ctx.config.root;
    const srcRoot = path.join(root, "src");
    const sourceFiles = await listSourceFiles(srcRoot);
    const files = [];

    for (const filePath of sourceFiles) {
      const source = await fs.readFile(filePath, "utf8");
      const entry = this.fileData.get(filePath) ?? { ranges: [], functionHits: [] };
      files.push(
        summarizeFile(filePath, source, entry.ranges, entry.functionHits, root)
      );
    }

    const totals = {
      lines: sumCoverage(files, "lines"),
      functions: sumCoverage(files, "functions"),
      statements: sumCoverage(files, "statements"),
      branches: sumCoverage(files, "branches"),
    };

    return {
      generatedAt: new Date().toISOString(),
      debug: {
        receivedSuites: this.receivedSuites,
        receivedScripts: this.receivedScripts,
      },
      totals,
      files,
    };
  }

  async reportCoverage(summary) {
    const reportsDirectory =
      this.options?.reportsDirectory ?? path.join(this.ctx.config.root, "coverage");

    await fs.mkdir(reportsDirectory, { recursive: true });
    const summaryPath = path.join(reportsDirectory, "coverage-summary.json");
    await fs.writeFile(summaryPath, JSON.stringify(summary, null, 2), "utf8");

    logSummary(summary);
  }
}

export default {
  getProvider() {
    return new LiteV8CoverageProvider();
  },
  startCoverage,
  takeCoverage,
  stopCoverage,
};
