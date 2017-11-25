export var breadcrumbs = []

function User() {
  this.name = ""
  this.isadmin = false
  this.email = ""
  this.reset = () => {
    this.name = ""
    this.isadmin = false
    this.email = ""
  }
}

function Config() {
  this.domain = ""
  this.ip = ""
}

export var user = new User()
export var config = new Config()
