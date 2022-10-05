/* eslint-disable prefer-promise-reject-errors */
import ApiService from '@/services/api.service'
import JwtService from '@/services/jwt.service'

// action types
export const VERIFY_AUTH = 'verify'
export const LOGIN = 'login'
export const LOGOUT = 'logout'
export const REGISTER = 'register'
export const UPDATE_PASSWORD = 'updateUser'

// mutation types
export const PURGE_AUTH = 'logOut'
export const SET_AUTH = 'setUser'
export const SET_PASSWORD = 'setPassword'
export const SET_ERROR = 'setError'

const state = {
  errors: null,
  user: {},
  isAuthenticated: !!JwtService.getToken(),
}

const getters = {
  currentUser (state) {
    return state.user
  },
  isAuthenticated (state) {
    return state.isAuthenticated
  },
}

const actions = {
  [LOGIN] (context, credentials) {
    try {
      return new Promise((resolve, reject) => {
        try {
          ApiService.post('auth/login', credentials)
            .then(({ data }) => {
              try {
                context.commit(SET_AUTH, data)
                resolve(data)
              } catch (e) {
                reject(e)
              }
            })
            .catch(({ response }) => {
              context.commit(SET_ERROR, response?.data)
              reject(response)
            })
        } catch (e) {
          reject(e)
        }
      })
    } catch (e) {
    }
  },
  [LOGOUT] (context) {
    context.commit(PURGE_AUTH)
  },
  [REGISTER] (context, credentials) {
    return new Promise(resolve => {
      ApiService.post('auth/register', credentials)
        .then(({ data }) => {
          context.commit(SET_AUTH, data)
          resolve(data)
        })
        .catch(({ response }) => {
          context.commit(SET_ERROR, response.data.errors)
        })
    })
  },
  [VERIFY_AUTH] (context) {
    return new Promise((resolve, reject) => {
      if (JwtService.getToken()) {
        ApiService.get('auth/verify')
          .then(({ data }) => {
            resolve(data)
          })
          .catch(({ response }) => {
            context.commit(PURGE_AUTH)
            context.commit(SET_ERROR, { error: 'no valid token', response: response })
            reject()
          })
      } else {
        context.commit(PURGE_AUTH)
        reject()
      }
    })
  },
  [UPDATE_PASSWORD] (context, payload) {
    const password = payload

    return ApiService.put('password', password).then(({ data }) => {
      context.commit(SET_PASSWORD, data)
      return data
    })
  },
}

const mutations = {
  [SET_ERROR] (state, error) {
    state.errors = error
    state.isAuthenticated = false
  },
  [SET_AUTH] (state, user) {
    try {
      state.isAuthenticated = true
      state.user = user
      state.errors = {}
      JwtService.saveToken(state.user.token)
    } catch (e) {
      console.log('ERROR SET_AUTH exception: ', e)
    }
  },
  [SET_PASSWORD] (state, password) {
    state.user.password = password
  },
  [PURGE_AUTH] (state) {
    state.isAuthenticated = false
    state.user = {}
    state.errors = {}
    JwtService.destroyToken()
  },
}

export default {
  namespaced: true,
  state,
  actions,
  mutations,
  getters,
}
