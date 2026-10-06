#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const ROOT = process.cwd();
const COMPONENTS_DIR = path.join(ROOT, 'ui', 'components');
const SOURCE_EXTENSIONS = ['.js', '.jsx', '.ts', '.tsx'];

function findSourceFiles(dir) {
  if (!fs.existsSync(dir)) return [];

  return fs.readdirSync(dir, { withFileTypes: true }).flatMap(entry => {
    const fullPath = path.join(dir, entry.name);

    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'dist') return [];
      return findSourceFiles(fullPath);
    }

    return entry.isFile() && SOURCE_EXTENSIONS.some(extension => fullPath.endsWith(extension))
      ? [fullPath]
      : [];
  });
}

const violations = findSourceFiles(COMPONENTS_DIR).filter(file =>
  /\bpersist\s*:\s*true\b/.test(fs.readFileSync(file, 'utf8'))
);

if (violations.length > 0) {
  console.log('⚠️  ADVERTENCIA: Estás persistiendo en componentes UI (mala práctica):');
  violations.forEach(file => console.log(`  - ${path.relative(ROOT, file)}`));
  console.log('  -> Quita persist:true de UI, solo va en controllers (background)');
} else {
  console.log('✅ No hay persistencia indebida en UI components');
}
