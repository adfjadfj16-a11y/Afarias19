#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const MAX_PERSIST_BYTES = 2 * 1024 * 1024;
const FILE_EXTENSIONS = new Set([
  '.js',
  '.jsx',
  '.ts',
  '.tsx',
  '.mjs',
  '.cjs',
  '.vue',
]);

function listFiles(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  const files = [];

  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (
      entry.name === '.git' ||
      entry.name === 'node_modules' ||
      entry.name === '.next' ||
      entry.name === '.github' ||
      entry.name === 'scripts'
    ) {
      continue;
    }

    if (entry.isDirectory()) {
      files.push(...listFiles(fullPath));
    } else if (FILE_EXTENSIONS.has(path.extname(entry.name).toLowerCase())) {
      files.push(fullPath);
    }
  }

  return files;
}

function findMatchingBrace(text, openIndex) {
  if (text[openIndex] !== '{') {
    return -1;
  }

  let depth = 0;
  let inString = false;
  let quoteChar = '';
  let escaped = false;

  for (let i = openIndex; i < text.length; i += 1) {
    const ch = text[i];

    if (inString) {
      if (escaped) {
        escaped = false;
        continue;
      }
      if (ch === '\\') {
        escaped = true;
        continue;
      }
      if (ch === quoteChar) {
        inString = false;
      }
      continue;
    }

    if (ch === '"' || ch === "'" || ch === '`') {
      inString = true;
      quoteChar = ch;
      continue;
    }

    if (ch === '{') {
      depth += 1;
    } else if (ch === '}') {
      depth -= 1;
      if (depth === 0) {
        return i;
      }
    }
  }

  return -1;
}

function estimateMb(bytes) {
  return bytes / (1024 * 1024);
}

function detectListenerLeaks(filePath, text) {
  const addCalls = (text.match(/addEventListener\s*\(/g) || []).length;
  const removeCalls = (text.match(/removeEventListener\s*\(/g) || []).length;

  if (addCalls > 0 && removeCalls === 0) {
    return {
      level: 'HIGH',
      code: 'LISTENER_LEAK',
      file: filePath,
      detail: 'addEventListener() detected without matching removeEventListener() handling.',
    };
  }

  return null;
}

function detectStorageUsage(filePath, text) {
  if (/chrome\.storage\.(local|sync)\s*/.test(text)) {
    return {
      level: 'BLOCKER',
      code: 'DIRECT_STORAGE_USAGE',
      file: filePath,
      detail: 'Direct chrome.storage.local/sync access is not allowed.',
    };
  }

  return null;
}

function fixPersistAnonymous(text) {
  if (/anonymous\s*:/m.test(text)) {
    return { text, changed: false };
  }

  const modified = text.replace(/persist\s*:\s*true\b/g, 'persist: true, anonymous: true');
  return { text: modified, changed: modified !== text };
}

function main() {
  const rootDir = process.cwd();
  const files = listFiles(rootDir);
  const findings = [];
  const fixedFiles = [];
  let totalPersistBytes = 0;

  if (files.length === 0) {
    console.log('No JavaScript/TypeScript files detected. No persist-memory violations found.');
    console.log('Persistent memory budget: 0.00MB / 2MB');
    return 0;
  }

  for (const filePath of files) {
    const text = fs.readFileSync(filePath, 'utf8');

    const storageIssue = detectStorageUsage(filePath, text);
    if (storageIssue) {
      findings.push(storageIssue);
    }

    const listenerLeakIssue = detectListenerLeaks(filePath, text);
    if (listenerLeakIssue) {
      findings.push(listenerLeakIssue);
    }

    const persistRegex = /persist\s*:\s*true/g;
    let match;
    while ((match = persistRegex.exec(text)) !== null) {
      const openIndex = text.lastIndexOf('{', match.index);
      if (openIndex === -1) {
        continue;
      }

      const closeIndex = findMatchingBrace(text, openIndex);
      if (closeIndex === -1) {
        continue;
      }

      const objectText = text.slice(openIndex, closeIndex + 1);
      const bytes = Buffer.byteLength(objectText, 'utf8');
      totalPersistBytes += bytes;

      if (!/anonymous\s*:/m.test(objectText)) {
        findings.push({
          level: 'BLOCKER',
          code: 'PERSIST_SIN_ANONYMOUS',
          file: filePath,
          detail: `persist: true slice missing anonymous: true (${estimateMb(bytes).toFixed(2)}MB estimated).`,
        });

        const fixed = fixPersistAnonymous(text);
        if (fixed.changed) {
          fs.writeFileSync(filePath, fixed.text, 'utf8');
          fixedFiles.push(filePath);
        }
      }

      if (bytes > MAX_PERSIST_BYTES) {
        findings.push({
          level: 'HIGH',
          code: 'PERSIST_PESADO',
          file: filePath,
          detail: `Persistent slice exceeds 2MB (${estimateMb(bytes).toFixed(2)}MB estimated).`,
        });
      }
    }
  }

  const totalPersistMb = estimateMb(totalPersistBytes);
  console.log(`Persistent memory budget: ${totalPersistMb.toFixed(2)}MB / 2MB`);

  if (fixedFiles.length > 0) {
    console.log('Auto-fixed anonymous: true for persist slices that were safe to patch.');
    for (const file of fixedFiles) {
      console.log(`- ${file}`);
    }
  }

  if (findings.length === 0) {
    console.log('All persistent-memory checks passed.');
    return 0;
  }

  const blockerCount = findings.filter((issue) => issue.level === 'BLOCKER').length;
  const highCount = findings.filter((issue) => issue.level === 'HIGH').length;

  if (highCount > 0 || blockerCount > 0) {
    console.log(`${blockerCount} blocker(s) and ${highCount} high-priority issue(s) found.`);
  }

  for (const issue of findings) {
    console.log(`[${issue.level}] ${issue.code} ${issue.file}: ${issue.detail}`);
  }

  return blockerCount > 0 ? 1 : 0;
}

process.exit(main());
