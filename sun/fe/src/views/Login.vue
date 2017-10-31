<template>
  <div>
    <el-row>
      <el-col :span="10" :offset="7">
        <el-alert
          title="For the Newest Users"
          description="There is no way to register by yourself, for now, please contact the gracious administrator if you do not have an account. (There may be a ticket system in the future, wait and see :)"
          type="warning" :closable="false" show-icon>
        </el-alert>
      </el-col>
    </el-row>
    <el-row>
      <el-col :span="6" :offset="9">
        <el-form label-position="left" label-width="80px" :model="form" class="hundred-width top-margin">
          <el-form-item label="Username">
            <el-input v-model="form.username" size="small"></el-input>
          </el-form-item>
          <el-form-item label="Password">
            <el-input @keyup.native.enter="login" v-model="form.password" type="password" size="small"></el-input>
          </el-form-item>
          <el-form-item>
            <el-button @click="login" :plain="true" round>Login</el-button>
          </el-form-item>
        </el-form>
      </el-col>
    </el-row>
  </div>
</template>

<script>
  import {breadcrumbs, user} from "../g.js"
  import {notifyErrResponse} from "../utils.js"

  export default {
    data() {
      return {
        form: {
          username: "",
          password: "",
        }
      }
    },
    created () {
      breadcrumbs.splice(0, breadcrumbs.length)
      breadcrumbs.push({route: {name: "Login"}, name: "Login"})
    },
    methods: {
      login () {
        var that = this
        if (that.username === "") {
          that.$notify.error({message: "User name can not be empty"})
          return
        }
        if (that.form.password === "") {
          that.$notify.error({message: "Password can not be empty"})
          return
        }
        that.$http.post("/api/login", that.form).then(
          (response) => {
            user.reset()
            that.$router.push({name: "Index"})
          },
          notifyErrResponse
        )
      }
    }
  }
</script>

<style scoped>
  .top-margin {
    margin-top: 2em;
  }
</style>
