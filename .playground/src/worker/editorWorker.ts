import * as monaco from 'monaco-editor/editor'
import 'monaco-editor/features/register.all'
import editorWorker from 'monaco-editor/editor/editor.worker?worker'
import cssWorker from 'monaco-editor/languages/features/css/css.worker?worker'
import htmlWorker from 'monaco-editor/languages/features/html/html.worker?worker'
import jsonWorker from 'monaco-editor/languages/features/json/json.worker?worker'
import { typescript } from 'monaco-editor/languages/features/register.all'
import tsWorker from 'monaco-editor/languages/features/typescript/ts.worker?worker'

;(self as any).MonacoEnvironment = {
  getWorker(_: any, label: string) {
    if (label === 'json') {
      return new jsonWorker()
    }
    if (label === 'css' || label === 'scss' || label === 'less') {
      return new cssWorker()
    }
    if (label === 'html' || label === 'handlebars' || label === 'razor') {
      return new htmlWorker()
    }
    if (label === 'typescript' || label === 'javascript') {
      return new tsWorker()
    }
    return new editorWorker()
  }
}

typescript.typescriptDefaults.setExtraLibs([])
typescript.javascriptDefaults.setEagerModelSync(true)
