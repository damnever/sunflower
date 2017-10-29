import {Notification} from 'element-ui'

import router from './router.js'
import {user} from './g.js'

export var notifyErrResponse = (response) => {
  var needLogin = () => {
    if (response.status === 401) {
      router.push({name: "Login"})
      user.reset()
    }
  }
  var msg = {
    title: response.status,
    message: response.statusText,
  }

  response.json().then((data) => {
    if ('message' in data) {
      msg.message = data.message
    }
    Notification.error(msg)
    needLogin()
  }).catch((reason) => {
    Notification.error(msg)
    needLogin()
  })
}

export var isEmptyObj = (obj) => {
  return (Object.keys(obj).length === 0 && obj.constructor === Object)
}

export var toParams = (obj) => {
  return '?'+Object.keys(obj).reduce(function(a,k){a.push(k+'='+encodeURIComponent(obj[k]));return a},[]).join('&')
}
