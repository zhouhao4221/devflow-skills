#!/usr/bin/env node

const { spawnSync, execSync } = require('child_process');
const fs = require('fs');
const path = require('path');
const os = require('os');
const https = require('https');

const VERSION = require('./package.json').version;
const BINARY_NAME = process.platform === 'win32' ? 'devflow-skills.exe' : 'devflow-skills';

function getPlatform() {
  const platform = os.platform();
  const arch = os.arch();

  const archMap = { x64: 'amd64', arm64: 'arm64', ia32: '386' };
  const mappedArch = archMap[arch] || arch;

  let platformName;
  switch (platform) {
    case 'linux':
      platformName = 'linux';
      break;
    case 'darwin':
      platformName = 'darwin';
      break;
    case 'win32':
      platformName = 'windows';
      break;
    default:
      console.error('不支持的平台: ' + platform);
      process.exit(1);
  }

  return `${platformName}_${mappedArch}`;
}

function getCacheDir() {
  const dir = path.join(os.homedir(), '.devflow-skills', VERSION);
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
  return dir;
}

function downloadBinary(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest, { mode: 0o755 });
    https
      .get(url, (response) => {
        if (response.statusCode === 302 || response.statusCode === 301) {
          https.get(response.headers.location, (redirectRes) => {
            redirectRes.pipe(file);
            file.on('finish', () => {
              file.close();
              resolve();
            });
          }).on('error', reject);
          return;
        }
        if (response.statusCode !== 200) {
          reject(new Error(`下载失败 (HTTP ${response.statusCode}): ${url}`));
          return;
        }
        response.pipe(file);
        file.on('finish', () => {
          file.close();
          resolve();
        });
      })
      .on('error', reject);
  });
}

async function getBinary() {
  const cacheDir = getCacheDir();
  const binaryPath = path.join(cacheDir, BINARY_NAME);

  if (fs.existsSync(binaryPath)) {
    return binaryPath;
  }

  const platform = getPlatform();
  const suffix = process.platform === 'win32' ? '.exe' : '';
  const url = `https://github.com/zhouhao4221/devflow-skills/releases/download/v${VERSION}/devflow-skills_${platform}${suffix}`;

  process.stderr.write(`首次运行，正在下载 devflow-skills v${VERSION} (${platform})...\n`);

  try {
    await downloadBinary(url, binaryPath);
    process.stderr.write('下载完成。\n');
  } catch (e) {
    process.stderr.write('下载失败，尝试使用 go install...\n');
    try {
      const goInstall = spawnSync('go', ['install', `github.com/zhouhao4221/devflow-skills@v${VERSION}`], {
        stdio: 'inherit',
      });
      if (goInstall.status !== 0) {
        console.error('go install 失败。请确保已安装 Go 1.21+ 或手动下载二进制。');
        console.error(`GitHub Releases: https://github.com/zhouhao4221/devflow-skills/releases/tag/v${VERSION}`);
        process.exit(1);
      }
      const goBinPath = path.join(os.homedir(), 'go', 'bin', BINARY_NAME);
      if (fs.existsSync(goBinPath)) {
        fs.copyFileSync(goBinPath, binaryPath);
        fs.chmodSync(binaryPath, 0o755);
        return binaryPath;
      }
    } catch (e2) {
      console.error('go install 失败:', e2.message);
      process.exit(1);
    }
  }

  fs.chmodSync(binaryPath, 0o755);
  return binaryPath;
}

async function main() {
  const binaryPath = await getBinary();
  const result = spawnSync(binaryPath, process.argv.slice(2), {
    stdio: 'inherit',
  });
  process.exit(result.status || 0);
}

main().catch((err) => {
  console.error('运行失败:', err.message);
  process.exit(1);
});
