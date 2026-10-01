#!/usr/bin/env node
/**
 * MetaMask Project 70 - Issue #18714 Auto-Migrator
 * Busca constantes deprecadas del design system y las migra a enums
 * + Check de robustez y memoria persistente
 */

const fs = require('fs');
const path = require('path');

const REPLACEMENTS = {
  DISPLAY: 'Display',
  FLEX_DIRECTION: 'FlexDirection',
  FLEX_WRAP: 'FlexWrap',
  BLOCK_SIZES: 'BlockSize',
  SEVERITIES: 'Severity',
  TEXT_ALIGN: 'TextAlign',
  TEXT_VARIANT: 'TextVariant',
  FONT_WEIGHT: 'FontWeight',
  FONT_STYLE: 'FontStyle',
  OVERFLOW_WRAP: 'OverflowWrap',
  JUSTIFY_CONTENT: 'JustifyContent',
  ALIGN_ITEMS: 'AlignItems',
};

const ROOT = process.cwd();
const UI_GLOB = path.join(ROOT, 'ui');

function findFiles(dir, exts = ['.js', '.tsx', '.ts', '.jsx']) {
  let results = [];
  if (!fs.existsSync(dir)) return results;
  const list = fs.readdirSync(dir);
  for (const file of list) {
    const full = path.join(dir, file);
    const stat = fs.statSync(full);
    if (stat.isDirectory() && !file.startsWith('node_modules') && !file.startsWith('dist')) {
      results = results.concat(findFiles(full, exts));
    } else if (exts.some(e => full.endsWith(e))) {
      results.push(full);
    }
  }
  return results;
}

console.log('🔍 [Project 70 - #18714] Buscando constantes deprecadas...\n');

const files = findFiles(UI_GLOB);
let found = [];
let fixed = 0;

for (const file of files) {
  const content = fs.readFileSync(file, 'utf8');
  for (const [oldConst, newEnum] of Object.entries(REPLACEMENTS)) {
    const regex = new RegExp(`\\b${oldConst}\\.`, 'g');
    if (regex.test(content)) {
      found.push({ file: file.replace(ROOT + '/', ''), oldConst, newEnum });
      let newContent = content;
      newContent = newContent.replace(
        new RegExp(`import\\s*{[^}]*\\b${oldConst}\\b[^}]*}.*from.*design-system.*`, 'g'),
        match => {
          if (!match.includes(newEnum)) {
            return match.replace(oldConst, newEnum);
          }
          return match;
        }
      );
      newContent = newContent.replace(new RegExp(`\\b${oldConst}\\.`, 'g'), `${newEnum}.`);

      if (newContent !== content) {
        fs.writeFileSync(file, newContent, 'utf8');
        fixed++;
      }
    }
  }
}

console.log('📋 Archivos encontrados:');
found.forEach(f => console.log(` - ${f.file}: ${f.oldConst} -> ${f.newEnum}`));

if (found.length === 0) {
  console.log('✅ No quedan constantes deprecadas en ui/. Ya estás limpio para #18714!');
} else {
  console.log(`\n🔧 Auto-fixeados ${fixed} archivos. Corre ahora:`);
  console.log('  yarn lint:changed:fix');
  console.log('  yarn test:unit ' + found.slice(0, 2).map(f => f.file).join(' '));
}

console.log('\n🛡️  Check de Robustez + Memoria Persistente Eficiente:');
console.log('---');

const persistViolations = [];
for (const file of files) {
  const content = fs.readFileSync(file, 'utf8');
  if (content.includes('persist: true') && file.includes('ui/components')) {
    persistViolations.push(file.replace(ROOT + '/', ''));
  }
}

if (persistViolations.length > 0) {
  console.log('⚠️  ADVERTENCIA: Estás persistiendo en componentes UI (mala práctica):');
  persistViolations.forEach(file => console.log('  - ' + file));
  console.log('  -> Quita persist:true de UI, solo va en controllers (background)');
} else {
  console.log('✅ No hay persistencia indebida en UI components');
}

console.log('\n🚀 Listo para PR: git commit -m "fix: migrate deprecated constants to enums (#18714)"');
