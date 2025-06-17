import js from "@eslint/js"
import tsEslint from "typescript-eslint"
import globals from "globals"
import pluginVue from "eslint-plugin-vue"

export default tsEslint.config(
  {
    ignores: ["**/swagger-client/*"]
  },
  {
    extends: [
      js.configs.recommended,
      ...tsEslint.configs.recommended,
      ...pluginVue.configs['flat/strongly-recommended'],
    ],
    files: ["**/*.{ts,vue}"],
    languageOptions: {
      ecmaVersion: "latest",
      sourceType: "module",
      globals: globals.browser,
      parserOptions: {
        parser: tsEslint.parser,
      }
    },

    rules: {
      "vue/multi-word-component-names": 0,
      "vue/component-api-style": ["error", ["script-setup", "composition"]],
      "vue/max-attributes-per-line": [
        "warn",
        {
          "singleline": {
            "max": 5
          },
          "multiline": {
            "max": 1
          }
        }
      ],
      "vue/singleline-html-element-content-newline": [
        "warn",
        {
          "ignoreWhenNoAttributes": true,
          "ignoreWhenEmpty": true
        }
      ]
    }
  }
)
