// Copyright (c) 2026 Lerian Studio. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0
//
// Unit tests for the engine release version logic. Run: node --test scripts/release/
// Zero npm dependencies — uses node:test + node:assert.

import { test } from 'node:test';
import assert from 'node:assert/strict';
import {
  computeNextVersion,
  compareSemver,
  parseSemver,
  engineReplaces,
  assertEngineWiring,
} from './engine-release.mjs';

const ENGINE = 'github.com/LerianStudio/fetcher/pkg/engine';

test('first engine release starts the v1 line per channel', () => {
  assert.equal(computeNextVersion(null, 'none', 'beta'), '1.0.0-beta.1');
  assert.equal(computeNextVersion(null, 'minor', 'beta'), '1.0.0-beta.1');
  assert.equal(computeNextVersion(null, 'none', 'rc'), '1.0.0-rc.1');
  assert.equal(computeNextVersion(null, 'none', ''), '1.0.0');
});

test('no-op engine bump still increments the prerelease counter (zero-drift price)', () => {
  assert.equal(computeNextVersion('v1.0.0-beta.1', 'none', 'beta'), '1.0.0-beta.2');
  assert.equal(computeNextVersion('v1.0.0-beta.3', 'none', 'beta'), '1.0.0-beta.4');
});

test('same-channel prerelease series ticks the counter regardless of bump type', () => {
  // Matches semantic-release semver.inc(v,'prerelease'): type is ignored mid-series;
  // the base advances only at the stable cut.
  assert.equal(computeNextVersion('v1.0.0-beta.2', 'patch', 'beta'), '1.0.0-beta.3');
  assert.equal(computeNextVersion('v1.0.0-beta.4', 'minor', 'beta'), '1.0.0-beta.5');
  assert.equal(computeNextVersion('v1.0.0-beta.4', 'major', 'beta'), '1.0.0-beta.5');
  assert.equal(computeNextVersion('v1.2.3-beta.9', 'minor', 'beta'), '1.2.3-beta.10');
});

test('prerelease opened from a stable tag applies the bump (at least a patch)', () => {
  assert.equal(computeNextVersion('v1.0.0', 'none', 'beta'), '1.0.1-beta.1');
  assert.equal(computeNextVersion('v1.0.0', 'minor', 'beta'), '1.1.0-beta.1');
  assert.equal(computeNextVersion('v1.0.0', 'major', 'beta'), '2.0.0-beta.1');
});

test('stable channel promotes an in-flight prerelease base', () => {
  assert.equal(computeNextVersion('v1.1.0-beta.2', 'patch', ''), '1.1.0');
  assert.equal(computeNextVersion('v2.0.0-beta.5', 'minor', ''), '2.0.0');
});

test('stable channel from a stable tag applies the bump', () => {
  assert.equal(computeNextVersion('v1.0.0', 'patch', ''), '1.0.1');
  assert.equal(computeNextVersion('v1.0.0', 'minor', ''), '1.1.0');
  assert.equal(computeNextVersion('v1.0.0', 'none', ''), '1.0.1');
});

test('rc channel ticks its own counter mid-series like beta', () => {
  assert.equal(computeNextVersion('v1.0.0-rc.1', 'none', 'rc'), '1.0.0-rc.2');
  assert.equal(computeNextVersion('v1.0.0-rc.1', 'minor', 'rc'), '1.0.0-rc.2');
});

test('engineReplaces finds a local engine replace and ignores unrelated ones', () => {
  assert.equal(engineReplaces({}).length, 0);
  assert.equal(engineReplaces({ Replace: [] }).length, 0);
  const mod = {
    Replace: [
      { Old: { Path: ENGINE }, New: { Path: './pkg/engine' } },
      { Old: { Path: 'example.com/other' }, New: { Path: './other' } },
    ],
  };
  assert.equal(engineReplaces(mod).length, 1);
});

test('assertEngineWiring passes when require is pinned to the cut tag and no replace', () => {
  const mod = { Require: [{ Path: ENGINE, Version: 'v1.2.3' }] };
  assert.doesNotThrow(() => assertEngineWiring(mod, '1.2.3'));
});

test('assertEngineWiring rejects a committed engine replace (the masking bug)', () => {
  const mod = {
    Require: [{ Path: ENGINE, Version: 'v1.2.3' }],
    Replace: [{ Old: { Path: ENGINE }, New: { Path: './pkg/engine' } }],
  };
  assert.throws(() => assertEngineWiring(mod, '1.2.3'), /must carry NO replace/);
});

test('assertEngineWiring rejects a require that does not match the cut tag (the stale-require bug)', () => {
  const mod = { Require: [{ Path: ENGINE, Version: 'v1.0.0-beta.1' }] };
  assert.throws(() => assertEngineWiring(mod, '1.0.0'), /does not match the cut engine tag/);
});

test('assertEngineWiring rejects a missing engine require', () => {
  assert.throws(() => assertEngineWiring({ Require: [] }, '1.0.0'), /missing a require/);
});

test('compareSemver orders prerelease below stable and by counter', () => {
  const sorted = ['v1.0.0', 'v1.0.0-beta.2', 'v1.0.0-beta.10', 'v0.9.0']
    .map(parseSemver)
    .sort(compareSemver);
  assert.deepEqual(
    sorted.map((s) => `${s.major}.${s.minor}.${s.patch}${s.preChannel ? `-${s.preChannel}.${s.preNum}` : ''}`),
    ['0.9.0', '1.0.0-beta.2', '1.0.0-beta.10', '1.0.0'],
  );
});
