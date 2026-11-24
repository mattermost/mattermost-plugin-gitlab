// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Helpers for the upgrade run (E2E_UPGRADE_FROM). The seed, apply, and verify
// projects run in separate processes, so they hand data over through a state file.

import {execFileSync} from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';

import './env';

import type {Client4} from '@mattermost/client';
import semver from 'semver';

import {isDisposableServer, loginRegularUser} from './admin';
import {PLUGIN_ID} from './constants';
import {loadCredentials} from './credentials';
import {GitlabApi} from './gitlab_api';
import {executeCommand} from './instance';

const REPO = 'mattermost/mattermost-plugin-gitlab';
const E2E_DIR = path.resolve(__dirname, '..');
const UPGRADE_DIR = path.join(E2E_DIR, 'results/upgrade');
const STATE_FILE = path.join(UPGRADE_DIR, 'upgrade-state.json');

// Data the seed creates on the base release. The label filter keeps label
// matching under test across the upgrade.
export const UPGRADE_LABEL = 'upgrade';
export const UPGRADE_FEATURES = `issues,issue_comments,label:"${UPGRADE_LABEL}"`;

export type PluginManifest = {
    id: string;
    version: string;
    min_server_version?: string;
};

export type BaseRelease = {
    tag: string;
    bundlePath: string;
    manifest: PluginManifest;
};

export type UpgradeState = {
    baseTag: string;
    baseVersion: string;
    groupId: number;
    projectId: number;
    projectPath: string;
    seededIssueTitle?: string;
};

type GithubAsset = {name: string; browser_download_url: string};
type GithubRelease = {tag_name: string; draft: boolean; prerelease: boolean; assets: GithubAsset[]};

export function upgradeFrom() {
    return process.env.E2E_UPGRADE_FROM?.trim() || undefined;
}

// Orders versions by semver precedence: build metadata (+hash) is ignored and
// prereleases sort before their release (2.1.0-rc1 < 2.1.0).
export function compareVersions(a: string, b: string) {
    return semver.compare(parseVersion(a), parseVersion(b));
}

function parseVersion(version: string) {
    const parsed = semver.parse(version.trim().replace(/^v/, ''));
    if (!parsed) {
        throw new Error(`"${version}" is not a semantic version`);
    }
    return parsed;
}

// Reads plugin.json from a bundle without unpacking it. Release bundles prefix
// their entries with "./", bundles from make dist do not.
export function bundleManifest(bundlePath: string): PluginManifest {
    const entries = execFileSync('tar', ['-tzf', bundlePath], {encoding: 'utf8'}).split('\n');
    const entry = entries.find((name) => /^(\.\/)?[^/]+\/plugin\.json$/.test(name.trim()));
    if (!entry) {
        throw new Error(`No plugin.json in ${bundlePath}`);
    }
    return JSON.parse(execFileSync('tar', ['-xzOf', bundlePath, entry.trim()], {encoding: 'utf8'}));
}

function githubToken() {
    const token = process.env.GH_TOKEN || process.env.GITHUB_TOKEN;
    if (token) {
        return token;
    }
    try {
        return execFileSync('gh', ['auth', 'token'], {encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore']}).trim();
    } catch {
        // Unauthenticated requests work for this public repository, with a lower rate limit.
        return '';
    }
}

async function githubGet<T>(apiPath: string, token: string): Promise<T> {
    const response = await fetch(`https://api.github.com${apiPath}`, {
        headers: {
            Accept: 'application/vnd.github+json',
            ...(token ? {Authorization: `Bearer ${token}`} : {}),
        },
    });
    if (!response.ok) {
        throw new Error(`GET ${apiPath} failed (${response.status}): ${await response.text()}`);
    }
    return response.json() as Promise<T>;
}

async function listReleases(token: string) {
    const releases: GithubRelease[] = [];
    for (let page = 1; page <= 10; page++) {
        // eslint-disable-next-line no-await-in-loop
        const batch = await githubGet<GithubRelease[]>(`/repos/${REPO}/releases?per_page=100&page=${page}`, token);
        releases.push(...batch);
        if (batch.length < 100) {
            break;
        }
    }
    return releases;
}

// A customer upgrades from the previous release on the same major line: majors can
// raise min_server_version, so a release from another major may not run on the same
// server. `latest` picks the newest published release of the build's major that is
// not newer than the build. An explicit tag must meet the same rules.
async function resolveRelease(spec: string, buildVersion: string, token: string): Promise<GithubRelease> {
    const build = parseVersion(buildVersion);
    const fits = (tag: string) => {
        const version = semver.parse(tag.replace(/^v/, ''));
        return version !== null && version.major === build.major && semver.compare(version, build) <= 0;
    };

    if (spec === 'latest') {
        const candidates = (await listReleases(token)).
            filter((release) => !release.draft && !release.prerelease && fits(release.tag_name) &&
                semver.prerelease(release.tag_name.replace(/^v/, '')) === null).
            sort((a, b) => compareVersions(b.tag_name, a.tag_name));
        if (candidates.length === 0) {
            throw new Error(`No published ${build.major}.x release at or below build ${buildVersion}. ` +
                'Set E2E_UPGRADE_FROM to a release tag, for example E2E_UPGRADE_FROM=v2.0.0.');
        }
        return candidates[0];
    }

    const tag = spec.startsWith('v') ? spec : `v${spec}`;
    parseVersion(tag);
    if (!fits(tag)) {
        throw new Error(`E2E_UPGRADE_FROM=${spec} must be a ${build.major}.x release at or below build ${buildVersion}.`);
    }
    return githubGet<GithubRelease>(`/repos/${REPO}/releases/tags/${tag}`, token);
}

// The test server runs linux/amd64. Older releases only ship the universal bundle.
function pickAsset(release: GithubRelease) {
    const asset = release.assets.find((a) => a.name.endsWith('-linux-amd64.tar.gz')) ??
        release.assets.find((a) => a.name === `mattermost-plugin-gitlab-${release.tag_name}.tar.gz`);
    if (!asset) {
        throw new Error(`Release ${release.tag_name} has no linux-amd64 or universal plugin bundle`);
    }
    return asset;
}

async function download(asset: GithubAsset) {
    const target = path.join(UPGRADE_DIR, asset.name);
    if (fs.existsSync(target) && fs.statSync(target).size > 0) {
        return target;
    }
    const response = await fetch(asset.browser_download_url);
    if (!response.ok) {
        throw new Error(`Downloading ${asset.name} failed (${response.status})`);
    }
    fs.mkdirSync(UPGRADE_DIR, {recursive: true});
    fs.writeFileSync(target, Buffer.from(await response.arrayBuffer()));
    return target;
}

// Resolves E2E_UPGRADE_FROM against the build under test, downloads that release's
// bundle, and checks that it can run on the test server.
export async function fetchBaseRelease(adminClient: Client4, spec: string, buildVersion: string): Promise<BaseRelease> {
    const token = githubToken();
    const release = await resolveRelease(spec, buildVersion, token);
    const bundlePath = await download(pickAsset(release));
    const manifest = bundleManifest(bundlePath);
    if (manifest.id !== PLUGIN_ID) {
        throw new Error(`${bundlePath} is plugin ${manifest.id}, not ${PLUGIN_ID}`);
    }

    const serverVersion = semver.coerce((await adminClient.getClientConfig()).Version);
    const minServer = manifest.min_server_version ? semver.coerce(manifest.min_server_version) : null;
    if (serverVersion && minServer && semver.lt(serverVersion, minServer)) {
        throw new Error(`${release.tag_name} needs Mattermost ${minServer.version}, but the test server runs ${serverVersion.version}.`);
    }
    return {tag: release.tag_name, bundlePath, manifest};
}

export async function runningPluginVersion(adminClient: Client4) {
    const statuses = await adminClient.getPluginStatuses();
    const status = statuses.find((s) => s.plugin_id === PLUGIN_ID);
    if (!status) {
        throw new Error(`${PLUGIN_ID} is not installed`);
    }
    return status.version;
}

// Error-level server log lines tagged with the plugin's plugin_id since the given time:
// the plugin's own errors (for example token decryption) and the server's errors about
// it (for example failed activation). The server logs JSON lines. Lines that only carry
// it as source_plugin_id are left out: the server logs one each time the plugin calls
// the mattermost-ai plugin, which the test stack does not run.
export function pluginErrorsSince(since: string): string[] {
    if (!isDisposableServer()) {
        // eslint-disable-next-line no-console
        console.warn('Skipping the server log check: it reads the docker compose logs of the disposable server.');
        return [];
    }
    const logs = execFileSync('docker', ['compose', 'logs', '--no-color', '--no-log-prefix', '--since', since, 'mattermost'], {
        cwd: E2E_DIR,
        encoding: 'utf8',
        maxBuffer: 64 * 1024 * 1024,
    });
    return logs.split('\n').filter((line) => {
        if (!line.startsWith('{') || !line.includes(PLUGIN_ID)) {
            return false;
        }
        try {
            const entry = JSON.parse(line) as {level?: string; plugin_id?: string};
            return entry.plugin_id === PLUGIN_ID && ['error', 'critical', 'fatal'].includes(entry.level ?? '');
        } catch {
            return false;
        }
    });
}

export function writeUpgradeState(state: UpgradeState) {
    fs.mkdirSync(UPGRADE_DIR, {recursive: true});
    fs.writeFileSync(STATE_FILE, `${JSON.stringify(state, null, 2)}\n`);
}

export function readUpgradeState(): UpgradeState {
    if (!fs.existsSync(STATE_FILE)) {
        throw new Error(`${STATE_FILE} is missing. The upgrade-seed project writes it, so run with E2E_UPGRADE_FROM set.`);
    }
    return JSON.parse(fs.readFileSync(STATE_FILE, 'utf8'));
}

export function readUpgradeStateIfAny(): UpgradeState | undefined {
    return fs.existsSync(STATE_FILE) ? readUpgradeState() : undefined;
}

// Removes what a seed created, the Town Square subscription and the GitLab group, then
// the state file, so the next seed does not try again. GitLab deletes groups after a
// delay, so a group removed by an earlier cleanup can still exist, marked for deletion.
export async function removeSeededData(state: UpgradeState) {
    try {
        await executeCommand((await loginRegularUser()).client, `/gitlab subscriptions delete ${state.projectPath}`);
    } catch {
        // The subscription may already be gone.
    }
    const root = new GitlabApi(loadCredentials());
    await root.closeOpenItems(state.projectId).catch(() => undefined);
    try {
        await root.deleteGroup(state.groupId);
    } catch (error) {
        if (!String(error).includes('marked for deletion')) {
            throw error;
        }
    }
    fs.rmSync(STATE_FILE, {force: true});
}
