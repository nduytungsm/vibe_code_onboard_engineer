#!/usr/bin/env node

/**
 * Frontend Setup Verification Script
 * Checks that all dependencies and configuration are working correctly
 */

import { execSync } from 'child_process';
import { readFileSync, existsSync } from 'fs';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

console.log('🔍 Repository Analyzer Frontend - Setup Verification\n');

// Test 1: Check package.json dependencies
console.log('1. Checking dependencies...');
try {
  const packageJson = JSON.parse(readFileSync(join(__dirname, 'package.json'), 'utf8'));
  const requiredDeps = [
    'react',
    'vite', 
    'tailwindcss',
    '@tailwindcss/vite',
    'lucide-react',
    'axios',
    'react-router-dom',
    '@headlessui/react'
  ];
  
  const allDeps = { ...packageJson.dependencies, ...packageJson.devDependencies };
  const missing = requiredDeps.filter(dep => !allDeps[dep]);
  
  if (missing.length === 0) {
    console.log('✅ All required dependencies installed');
  } else {
    console.log('❌ Missing dependencies:', missing.join(', '));
    process.exit(1);
  }
} catch (error) {
  console.log('❌ Failed to read package.json:', error.message);
  process.exit(1);
}

// Test 2: Check Vite configuration
console.log('2. Checking Vite configuration...');
try {
  const viteConfig = readFileSync(join(__dirname, 'vite.config.js'), 'utf8');
  if (viteConfig.includes('@tailwindcss/vite') && viteConfig.includes('tailwindcss()')) {
    console.log('✅ Vite configuration correct for Tailwind v4');
  } else {
    console.log('❌ Vite configuration missing Tailwind plugin');
    process.exit(1);
  }
} catch (error) {
  console.log('❌ Failed to read vite.config.js:', error.message);
  process.exit(1);
}

// Test 3: Check CSS configuration
console.log('3. Checking CSS configuration...');
try {
  const cssContent = readFileSync(join(__dirname, 'src/index.css'), 'utf8');
  if (cssContent.includes('@import "tailwindcss"') && cssContent.includes('@theme')) {
    console.log('✅ CSS configured for Tailwind CSS v4');
  } else {
    console.log('❌ CSS configuration incorrect');
    process.exit(1);
  }
} catch (error) {
  console.log('❌ Failed to read src/index.css:', error.message);
  process.exit(1);
}

// Test 4: Check that old config files are removed
console.log('4. Checking legacy configuration cleanup...');
const legacyFiles = ['postcss.config.js', 'tailwind.config.js'];
const remainingLegacy = legacyFiles.filter(file => existsSync(join(__dirname, file)));

if (remainingLegacy.length === 0) {
  console.log('✅ Legacy configuration files cleaned up');
} else {
  console.log('⚠️  Legacy files still present:', remainingLegacy.join(', '));
}

// Test 5: Check API utilities
console.log('5. Checking API utilities...');
try {
  const apiFile = readFileSync(join(__dirname, 'src/utils/api.js'), 'utf8');
  if (apiFile.includes('repositoryAPI') && apiFile.includes('axios')) {
    console.log('✅ API utilities configured');
  } else {
    console.log('❌ API utilities incomplete');
  }
} catch (error) {
  console.log('❌ API utilities missing:', error.message);
}

// Test 6: Try building the project
console.log('6. Testing build process...');
try {
  execSync('npm run build', { stdio: 'pipe', cwd: __dirname });
  console.log('✅ Build process successful');
  
  // Clean up build output
  execSync('rm -rf dist', { cwd: __dirname });
} catch (error) {
  console.log('❌ Build failed:', error.message);
  process.exit(1);
}

console.log('\n🎉 Frontend setup verification complete!');
console.log('🚀 Ready to start development with: npm run dev');
console.log('🌐 Application will be available at: http://localhost:5173');
