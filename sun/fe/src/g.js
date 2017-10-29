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

export var user = new User()
