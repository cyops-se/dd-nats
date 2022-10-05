module.exports = {
  devServer: {
    disableHostCheck: true,
    proxy: {
      '^/api': {
        target: 'http://localhost:3000/',
        ws: false,
        changeOrigin: true,
      },
      '^/auth': {
        target: 'http://localhost:3000/',
        ws: false,
        changeOrigin: true,
      },
      '^/static': {
        target: 'http://localhost:3000/',
        ws: false,
        changeOrigin: true,
      },
      '^/ws': {
        target: 'http://localhost:3000/ws',
        ws: true,
        changeOrigin: true,
      },
    },
  },

  transpileDependencies: ['vuetify'],

  publicPath: '/',
}
