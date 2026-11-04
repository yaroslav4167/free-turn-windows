import {test} from 'node:test'
import assert from 'node:assert/strict'
import {availableLanguages, configureLanguage, currentLanguage, languagePreference, registerCatalogs, subscribeLanguage, systemLanguage, tr} from '../src/language.ts'
import {connectionState} from '../src/connectionState.ts'
import {updateProgress} from '../src/updateProgress.ts'
import {readdirSync, readFileSync} from 'node:fs'
import ts from 'typescript'
import english from '../src/locales/en.json' with {type: 'json'}

test('system language detection normalizes locales and falls back to English', () => {
  for (const locale of ['ru', 'ru-RU', 'ru_RU.UTF-8', 'RU-ru']) assert.equal(systemLanguage([locale]), 'ru')
  for (const locale of ['en-US', 'de-DE', 'fr', 'C', 'russian']) assert.equal(systemLanguage([locale]), 'en')
  assert.equal(systemLanguage([]), 'en')
  assert.equal(systemLanguage(['de-DE', 'ru-RU']), 'en')
})

test('Russian UI literals are paired with English, except the native language name', () => {
  const checkDirectory = (directory) => {
    for (const entry of readdirSync(directory, {withFileTypes: true})) {
      const path = new URL(entry.name + (entry.isDirectory() ? '/' : ''), directory)
      if (entry.isDirectory()) { checkDirectory(path); continue }
      if (!/\.tsx?$/.test(entry.name)) continue
      const source = ts.createSourceFile(path.pathname, readFileSync(path, 'utf8'), ts.ScriptTarget.Latest, true,
        entry.name.endsWith('.tsx') ? ts.ScriptKind.TSX : ts.ScriptKind.TS)
      const visit = (node) => {
        if ((ts.isStringLiteral(node) || ts.isJsxText(node) || ts.isTemplateHead(node) || ts.isTemplateMiddle(node) || ts.isTemplateTail(node)) && /[А-Яа-яЁё]/.test(node.text)) {
          let parent = node.parent
          let paired = false
          while (parent) {
            if (ts.isCallExpression(parent) && parent.expression.getText(source) === 'tr' && parent.arguments.length >= 1) { paired = true; break }
            parent = parent.parent
          }
          assert.ok(paired || node.text === 'Русский', `Untranslated UI text in ${path.pathname}: ${node.text}`)
        }
        if (ts.isCallExpression(node) && node.expression.getText(source) === 'tr' && ts.isStringLiteral(node.arguments[0])) {
          assert.ok(Object.hasOwn(english.messages, node.arguments[0].text), `Message missing from JSON: ${node.arguments[0].text}`)
        }
        ts.forEachChild(node, visit)
      }
      visit(source)
    }
  }
  checkDirectory(new URL('../src/', import.meta.url))
})

test('new JSON catalogs are selectable and use English fallback with safe interpolation', () => {
  registerCatalogs([{code: 'fr', name: 'Français', messages: {'Настройки': 'Paramètres'}}])
  assert.ok(availableLanguages().some(language => language.code === 'fr'))
  assert.equal(systemLanguage(['fr_FR.UTF-8']), 'fr')
  configureLanguage('fr', 'ru')
  assert.equal(tr('Настройки', 'ignored inline fallback'), 'Paramètres')
  assert.equal(tr('Серверы'), 'Servers')
  assert.equal(tr('Удалить сервер «{name}»?', {name: '<user>{version}'}), 'Delete server “<user>{version}”?')
  assert.equal(tr('Скачать {version}'), 'Download {version}')
  configureLanguage('system', 'ru')
})

test('manual language overrides system and switching is reactive without remounting', () => {
  let changes = 0
  const unsubscribe = subscribeLanguage(() => changes++)
  configureLanguage('en', 'ru')
  assert.equal(currentLanguage(), 'en')
  assert.equal(languagePreference(), 'en')
  assert.equal(tr('Настройки', 'Settings'), 'Settings')
  assert.equal(connectionState('connecting', false, true).action, 'Cancel')
  assert.equal(connectionState('connected', false, true).heading, 'VPN enabled')
  assert.equal(updateProgress(null, 'checking').title, 'Checking for updates')
  configureLanguage('system', 'ru')
  assert.equal(currentLanguage(), 'ru')
  assert.equal(tr('Настройки', 'Settings'), 'Настройки')
  assert.equal(connectionState('connected', false, true).heading, 'VPN включён')
  configureLanguage('invalid', 'en')
  assert.equal(languagePreference(), 'system')
  assert.equal(currentLanguage(), 'en')
  assert.equal(changes, 3)
  unsubscribe()
  configureLanguage('system', 'ru')
  assert.equal(changes, 3)
})
