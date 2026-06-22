#!/usr/bin/env node
// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0
//
// engine-release.mjs — dual-module release helper for the Fetcher monorepo.
//
// The repo ships two Go modules from one repository:
//   - parent:  github.com/LerianStudio/fetcher/v2            (tags: v2.x.x)
//   - engine:  github.com/LerianStudio/fetcher/pkg/engine    (tags: pkg/engine/vX.Y.Z)
//
// semantic-release manages the PARENT line. This script is invoked from the
// `@semantic-release/exec` prepare hook (see .releaserc.yml) so that, in the SAME
// release run, the engine module is cut FIRST on its own v1 line and the parent's
// go.mod require is pinned to the exact engine tag before semantic-release commits
// and tags the parent. "Cut both always" = lockstep on the release EVENT, with each
// module carrying its own independent semver. An unchanged engine still gets a fresh
// tag (a no-op prerelease bump) — that is the accepted price of zero drift.
//
// Subcommands:
//   compute  --channel <beta|rc|"">                 → prints the engine's next version (no side effects)
//   prepare  --parent-version <v> --channel <ch>     → engine-first tag + push, pin parent require,
//                                                       drop the transient replace, GOWORK=off gate,
//                                                       update engine changelog. semantic-release's
//                                                       git plugin then commits go.mod/go.sum/CHANGELOGs
//                                                       and tags the parent.
//
// Zero npm dependencies: Node stdlib + git/go only.

import { execFileSync } from 'node:child_process';
import { existsSync, readFileSync, writeFileSync } from 'node:fs';

const ENGINE_MODULE = 'github.com/LerianStudio/fetcher/pkg/engine';
const ENGINE_TAG_PREFIX = 'pkg/engine/';
const ENGINE_CHANGELOG = 'pkg/engine/CHANGELOG.md';

// Bump rules mirror .releaserc.yml releaseRules for the standard types. The engine's
// "cut both always" policy is intentionally more liberal: 'style' and unrecognized
// types map to patch so every release event yields a fresh tag. This only affects the
// stable->prerelease transition and stable cuts; mid-beta the counter ticks regardless
// (computeNextVersion also coerces 'none' -> patch on a stable cut), so the divergence
// from the parent's skip-on-unknown behavior is functionally moot.
const MINOR_TYPES = new Set(['feat', 'perf', 'build', 'refactor']);
const PATCH_TYPES = new Set(['fix', 'chore', 'ci', 'test', 'docs', 'style']);

function sh(cmd, args, opts = {}) {
  return execFileSync(cmd, args, { encoding: 'utf8', ...opts }).trim();
}

function git(args, opts = {}) {
  return sh('git', args, opts);
}

function parseArgs(argv) {
  const out = {};
  for (let i = 0; i < argv.length; i += 1) {
    if (argv[i].startsWith('--')) {
      const key = argv[i].slice(2);
      const val = argv[i + 1] && !argv[i + 1].startsWith('--') ? argv[(i += 1)] : 'true';
      out[key] = val;
    }
  }
  return out;
}

// --- semver (minimal, prerelease-aware) ----------------------------------------

function parseSemver(v) {
  const m = /^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$/.exec(v);
  if (!m) throw new Error(`unparseable semver: ${v}`);
  const [, major, minor, patch, pre] = m;
  let preChannel = null;
  let preNum = null;
  if (pre) {
    const pm = /^([0-9A-Za-z-]+)\.(\d+)$/.exec(pre);
    if (!pm) throw new Error(`unsupported prerelease format: ${pre}`);
    preChannel = pm[1];
    preNum = Number(pm[2]);
  }
  return { major: +major, minor: +minor, patch: +patch, preChannel, preNum };
}

function baseString(s) {
  return `${s.major}.${s.minor}.${s.patch}`;
}

function applyBump(base, bump) {
  if (bump === 'major') return { major: base.major + 1, minor: 0, patch: 0 };
  if (bump === 'minor') return { major: base.major, minor: base.minor + 1, patch: 0 };
  if (bump === 'patch') return { major: base.major, minor: base.minor, patch: base.patch + 1 };
  return { major: base.major, minor: base.minor, patch: base.patch };
}

// Compute the engine's next version given the last engine tag, the bump implied by
// pkg/engine commits, and the release channel. Mirrors semantic-release prerelease
// semantics for the common cases; documented in docs/RELEASING.md.
function computeNextVersion(lastTag, bump, channel) {
  const isPrerelease = channel === 'beta' || channel === 'rc';

  // First-ever engine release: start the v1 line.
  if (!lastTag) {
    return isPrerelease ? `1.0.0-${channel}.1` : '1.0.0';
  }

  const last = parseSemver(lastTag);
  const base = { major: last.major, minor: last.minor, patch: last.patch };

  if (isPrerelease) {
    if (last.preChannel === channel) {
      // Same-channel prerelease series. semantic-release does semver.inc(v,'prerelease')
      // here: a PURE counter tick, independent of the commit bump type. The base only
      // advances at the stable cut (computed from commits since the last STABLE). This
      // also guarantees a fresh tag every run — the zero-drift "no-op bump" price.
      return `${baseString(base)}-${channel}.${last.preNum + 1}`;
    }
    // Last tag was stable (or a different prerelease channel): open a fresh prerelease
    // of the next bump. A no-op bump still advances at least a patch so the tag is fresh.
    const effective = bump === 'none' ? 'patch' : bump;
    return `${baseString(applyBump(base, effective))}-${channel}.1`;
  }

  // Stable channel.
  if (last.preChannel) {
    // Promote the in-flight prerelease base to stable (1.1.0-beta.2 → 1.1.0).
    return baseString(base);
  }
  const effective = bump === 'none' ? 'patch' : bump;
  return baseString(applyBump(base, effective));
}

// --- engine state ---------------------------------------------------------------

function latestEngineTag() {
  let raw = '';
  try {
    raw = git(['tag', '--list', `${ENGINE_TAG_PREFIX}v*`]);
  } catch {
    return null;
  }
  const versions = raw
    .split('\n')
    .map((t) => t.trim())
    .filter(Boolean)
    .map((t) => t.slice(ENGINE_TAG_PREFIX.length)) // "vX.Y.Z[-pre]"
    .map((v) => {
      try {
        return { tag: v, parsed: parseSemver(v) };
      } catch {
        return null;
      }
    })
    .filter(Boolean);
  if (versions.length === 0) return null;

  versions.sort((a, b) => compareSemver(a.parsed, b.parsed));
  return versions[versions.length - 1].tag; // highest, e.g. "v1.0.0-beta.3"
}

function compareSemver(a, b) {
  if (a.major !== b.major) return a.major - b.major;
  if (a.minor !== b.minor) return a.minor - b.minor;
  if (a.patch !== b.patch) return a.patch - b.patch;
  // A version without prerelease is greater than one with.
  if (!a.preChannel && b.preChannel) return 1;
  if (a.preChannel && !b.preChannel) return -1;
  if (!a.preChannel && !b.preChannel) return 0;
  if (a.preChannel !== b.preChannel) return a.preChannel < b.preChannel ? -1 : 1;
  return a.preNum - b.preNum;
}

// Bump implied by commits touching pkg/engine/** since the last engine tag.
function engineBumpSinceTag(lastTag) {
  const range = lastTag ? `${ENGINE_TAG_PREFIX}${lastTag}..HEAD` : 'HEAD';
  let log = '';
  try {
    // %x1f unit separator between subject and body; %x1e record separator.
    log = git(['log', range, '--format=%s%x1f%b%x1e', '--', 'pkg/engine']);
  } catch {
    return 'none';
  }
  if (!log) return 'none';

  let level = 'none';
  const rank = { none: 0, patch: 1, minor: 2, major: 3 };
  for (const record of log.split('\x1e')) {
    const [subject = '', body = ''] = record.split('\x1f');
    const subj = subject.trim();
    if (!subj) continue;

    const header = /^([a-z]+)(\([^)]*\))?(!)?:/i.exec(subj);
    let lvl = 'patch';
    const breaking = (header && header[3] === '!') || /BREAKING[ -]CHANGE/.test(body) || /BREAKING[ -]CHANGE/.test(subj);
    if (breaking) {
      lvl = 'major';
    } else if (header) {
      const type = header[1].toLowerCase();
      if (MINOR_TYPES.has(type)) lvl = 'minor';
      else if (PATCH_TYPES.has(type)) lvl = 'patch';
      else lvl = 'patch';
    }
    if (rank[lvl] > rank[level]) level = lvl;
  }
  return level;
}

// --- go.mod wiring assertions ----------------------------------------------------
// Pure checks over `go mod edit -json` output. These make the GOWORK=off gate real:
// a committed `replace => ./pkg/engine` resolves the engine locally and the require
// pin is never exercised — the masking bug this guards against.

// Replace directives in `mod` that retarget the engine module (e.g. the transient
// `=> ./pkg/engine` bootstrap). The committed parent go.mod MUST have none once the
// engine ships its own tag; local/in-repo dev resolves the engine via go.work instead.
function engineReplaces(mod) {
  return (mod.Replace || []).filter((r) => r.Old && r.Old.Path === ENGINE_MODULE);
}

// Throws unless the parent go.mod requires the engine at exactly v${version} and
// carries no engine replace. Run on the final, about-to-be-committed go.mod.
function assertEngineWiring(mod, version) {
  const replaces = engineReplaces(mod);
  if (replaces.length > 0) {
    throw new Error(
      `parent go.mod must carry NO replace for ${ENGINE_MODULE} (local dev uses go.work); found: ` +
        replaces.map((r) => `${r.Old.Path} => ${r.New ? r.New.Path : '?'}`).join(', '),
    );
  }
  const req = (mod.Require || []).find((r) => r.Path === ENGINE_MODULE);
  if (!req) throw new Error(`parent go.mod is missing a require for ${ENGINE_MODULE}`);
  if (req.Version !== `v${version}`) {
    throw new Error(
      `parent require ${ENGINE_MODULE}@${req.Version} does not match the cut engine tag v${version}`,
    );
  }
}

// --- prepare side effects --------------------------------------------------------

function parsedGoMod() {
  return JSON.parse(sh('go', ['mod', 'edit', '-json']));
}

function run(cmd, args, env = {}) {
  process.stderr.write(`+ ${cmd} ${args.join(' ')}\n`);
  execFileSync(cmd, args, { stdio: 'inherit', env: { ...process.env, ...env } });
}

function prependChangelog(version, channel, parentVersion) {
  const stampSource = parentVersion || version;
  const header = `## pkg/engine ${version}\n\nReleased alongside parent ${stampSource}` +
    `${channel ? ` (${channel})` : ''}.\n`;
  let prev = '';
  if (existsSync(ENGINE_CHANGELOG)) prev = readFileSync(ENGINE_CHANGELOG, 'utf8');
  else prev = '# Changelog — github.com/LerianStudio/fetcher/pkg/engine\n';
  // Insert the new entry after the top-level title if present.
  const lines = prev.split('\n');
  if (lines[0]?.startsWith('# ')) {
    const out = [lines[0], '', header, ...lines.slice(1)].join('\n');
    writeFileSync(ENGINE_CHANGELOG, out);
  } else {
    writeFileSync(ENGINE_CHANGELOG, `${header}\n${prev}`);
  }
}

function doPrepare(opts) {
  const rawChannel = opts.channel && opts.channel !== 'true' && opts.channel !== 'null' ? opts.channel : '';
  const channel = rawChannel.trim();
  const parentVersion = opts['parent-version'] || '';

  const lastTag = latestEngineTag();
  const bump = engineBumpSinceTag(lastTag);
  const version = computeNextVersion(lastTag, bump, channel);
  const engineTag = `${ENGINE_TAG_PREFIX}v${version}`;

  process.stderr.write(
    `[engine-release] last=${lastTag ?? '<none>'} bump=${bump} channel=${channel || '<stable>'} ` +
      `→ engine ${version} (${engineTag}); parent=${parentVersion}\n`,
  );

  // 0. Pre-flight: the committed parent go.mod must NOT bootstrap-replace the engine.
  //    A committed `replace => ./pkg/engine` resolves the engine locally under
  //    GOWORK=off, making the require-resolution gate below vacuous. Fail before we
  //    push any tag, so a reintroduced bootstrap replace can never ship.
  const preReplaces = engineReplaces(parsedGoMod());
  if (preReplaces.length > 0) {
    throw new Error(
      `committed parent go.mod carries a replace for ${ENGINE_MODULE} ` +
        `(${preReplaces.map((r) => `=> ${r.New ? r.New.Path : '?'}`).join(', ')}); ` +
        `remove it — local dev uses go.work, external resolution uses the require.`,
    );
  }

  // 1. Engine-first: tag + push the engine module BEFORE the parent commit/tag so a
  //    `go get` of the parent can never observe an unresolvable engine require.
  run('git', ['tag', engineTag]);
  run('git', ['push', 'origin', engineTag]);

  // 2. Pin the parent require to the exact engine tag and drop the transient replace.
  run('go', ['mod', 'edit', `-dropreplace=${ENGINE_MODULE}`]);
  run('go', ['mod', 'edit', `-require=${ENGINE_MODULE}@v${version}`]);

  // 3. GOWORK=off gate: prove the parent resolves the engine via the proxy/tag, not
  //    via go.work. A stale/missing require fails the release here.
  //
  // Configure git globally so `go mod tidy` can resolve private LerianStudio modules
  // via git ls-remote. actions/checkout only sets credentials locally for the current
  // repo — other org repos (e.g. lib-license-go) need a global url.insteadOf override.
  const token = process.env.GITHUB_TOKEN;
  if (token) {
    execFileSync('git', [
      'config', '--global',
      `url.https://x-access-token:${token}@github.com/.insteadOf`,
      'https://github.com/',
    ], { stdio: 'inherit', env: process.env });
  }

  const off = { GOWORK: 'off', GOFLAGS: '-mod=mod' };
  run('go', ['mod', 'download', `${ENGINE_MODULE}@v${version}`], off);
  run('go', ['mod', 'tidy'], { GOWORK: 'off' });
  run('go', ['build', './...'], { GOWORK: 'off' });
  if (process.env.ENGINE_RELEASE_SKIP_TESTS !== 'true') {
    run('go', ['test', './...'], { GOWORK: 'off' });
  }

  // 3b. Final wiring assertion on the exact go.mod that @semantic-release/git will
  //     commit: engine require pinned to the just-cut tag, and no replace masking it.
  assertEngineWiring(parsedGoMod(), version);

  // 4. Engine changelog entry (parent CHANGELOG is handled by @semantic-release/git).
  prependChangelog(version, channel, parentVersion);

  process.stderr.write(`[engine-release] prepared engine ${engineTag}; parent require pinned.\n`);
  // stdout = the engine version, for any downstream capture.
  process.stdout.write(`${version}\n`);
}

function doCompute(opts) {
  const rawChannel = opts.channel && opts.channel !== 'true' && opts.channel !== 'null' ? opts.channel : '';
  const channel = rawChannel.trim();
  const lastTag = latestEngineTag();
  const bump = engineBumpSinceTag(lastTag);
  const version = computeNextVersion(lastTag, bump, channel);
  process.stdout.write(`${version}\n`);
}

function main() {
  const [, , subcommand, ...rest] = process.argv;
  const opts = parseArgs(rest);
  switch (subcommand) {
    case 'compute':
      doCompute(opts);
      break;
    case 'prepare':
      doPrepare(opts);
      break;
    default:
      process.stderr.write('usage: engine-release.mjs <compute|prepare> [--channel ch] [--parent-version v]\n');
      process.exit(2);
  }
}

// Pure helpers are exported for unit testing; main() runs only when executed directly.
export { parseSemver, compareSemver, applyBump, computeNextVersion, engineReplaces, assertEngineWiring };

if (process.argv[1] && import.meta.url === new URL(`file://${process.argv[1]}`).href) {
  main();
}
