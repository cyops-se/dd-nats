module.exports = {
  root: true,

  env: {
    node: true,
  },

  extends: 'vuetify',

  parserOptions: {
    // parser: 'babel-eslint',
    parser: '@babel/eslint-parser',
  },

  rules: {
    'no-console': 'off',
    'no-debugger': 'off',
    // https://github.com/babel/babel-eslint/issues/681#issuecomment-420663038
    'template-curly-spacing': 'off',
    'indent': 'off',
    //
    'no-unused-vars': 'off',
  },

  overrides: [
    {
      files: [
        '**/__tests__/*.{j,t}s?(x)',
        '**/tests/unit/**/*.spec.{j,t}s?(x)',
      ],
      env: {
        jest: true,
      },
    },
  ],
}
