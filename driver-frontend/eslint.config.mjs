import { fixupPluginRules } from '@eslint/compat';
import { FlatCompat } from '@eslint/eslintrc';
import js from '@eslint/js';
import typescriptEslint from '@typescript-eslint/eslint-plugin';
import tsParser from '@typescript-eslint/parser';
import eslintPluginPrettier from 'eslint-plugin-prettier';
import reactHooks from 'eslint-plugin-react-hooks';
import globals from 'globals';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const compat = new FlatCompat({
    baseDirectory: __dirname,
    recommendedConfig: js.configs.recommended,
    allConfig: js.configs.all,
});

export default [
    ...compat.extends('eslint:recommended', 'plugin:react/recommended', 'plugin:@typescript-eslint/recommended'),
    {
        plugins: {
            '@typescript-eslint': typescriptEslint,
            prettier: eslintPluginPrettier,
            'react-hooks': fixupPluginRules(reactHooks),
        },

        languageOptions: {
            globals: {
                ...globals.browser,
                ...globals.node,
                window: 'readonly',
                describe: 'readonly',
                test: 'readonly',
                expect: 'readonly',
                it: 'readonly',
                process: 'readonly',
                document: 'readonly',
                insights: 'readonly',
                shallow: 'readonly',
                render: 'readonly',
                mount: 'readonly',
            },

            parser: tsParser,
            ecmaVersion: 5,
            sourceType: 'commonjs',

            parserOptions: {
                tsx: true,
                jsx: true,
                js: true,
                useJSXTextNode: true,
                tsconfigRootDir: '.',
            },
        },

        settings: {
            react: {
                version: '^16.11.0',
            },
        },

        rules: {
            'sort-imports': [
                'error',
                {
                    ignoreDeclarationSort: true,
                },
            ],

            '@typescript-eslint/explicit-function-return-type': 'off',
            'react-hooks/rules-of-hooks': 'error',
            'react-hooks/exhaustive-deps': 'warn',
            '@typescript-eslint/interface-name-prefix': 'off',
            // "prettier/prettier": "off",
            'import/no-unresolved': 'off',
            'import/extensions': 'off',
            'react/prop-types': 'off',
            'forbid-pf-relative-imports': 'off',
            'no-unused-vars': ['error', { ignoreRestSiblings: true }],
            'prettier/prettier': ['error', { singleQuote: true }],
        },
    },
    ...compat.extends('plugin:@typescript-eslint/recommended').map((config) => ({
        ...config,
        files: ['src/**/*.ts', 'src/**/*.tsx', 'src/**/*.js'],
    })),
    {
        files: ['src/**/*.ts', 'src/**/*.tsx', 'src/**/*.js'],

        plugins: {
            '@typescript-eslint': typescriptEslint,
            prettier: eslintPluginPrettier,
        },

        languageOptions: {
            parser: tsParser,
        },

        rules: {
            'react/prop-types': 'off',
            '@typescript-eslint/no-unused-vars': 'error',
            'forbid-pf-relative-imports': 'off',
        },
    },
];
