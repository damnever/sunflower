<template>
  <div>
    <el-row>
      <el-col :span="12">
        <el-card class="box-card">
          <div slot="header">
            <span style="line-height: 1.6em;">Update Password or E-mail</span>
            <el-button style="float: right;" type="primary" size="small"
              @click="updateInfo">Update</el-button>
          </div>
          <el-form label-position="right">
            <el-form-item label="Password" label-width="80px">
              <el-input v-model="password" auto-complete="off" size="small"
                type="password" placeholder="Input a new passord">
              </el-input>
            </el-form-item>
            <el-form-item label="Password" label-width="80px">
              <el-input v-model="password2" auto-complete="off" size="small"
                type="password" placeholder="Confirm the new passord">
              </el-input>
            </el-form-item>
            <el-form-item label="E-mail" label-width="80px">
              <el-input v-model="email" auto-complete="off" size="small"
                placeholder="Input a new E-amil address">
              </el-input>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
      <el-col :span="10" :offset="2">
        <el-card class="box-card">
          <div slot="header">
            <span style="line-height: 1.6em;">Danger Zone</span>
          </div>
            <el-button type="danger" @click="deleteSelf" :disabled="user.isadmin">
              Delete Your Account
            </el-button>
        </el-card>
      </el-col>

    </el-row>
  </div>
</template>

<script>
  import {breadcrumbs, user} from '../g.js'
  import {notifyErrResponse, isEmptyObj} from '../utils.js'

  export default {
    data () {
      return {
        user: user,
        password: "",
        password2: "",
        email: user.email,
      }
    },
    created () {
      // tricky..
      breadcrumbs.splice(0, breadcrumbs.length)
      breadcrumbs.push({route: {name: "Index"}, name: "Index"})
      breadcrumbs.push({route: {name: "Profile"}, name: "Profile"})
    },
    methods: {
      updateInfo() {
        var that = this
        var data = {}

        if (that.password !== "") {
          if (that.password !== that.password2) {
            that.$notify.error({message: "Two passwords doesn't match"})
            return
          }
          data.password = that.password
        }
        if (that.email !== "" && that.email !== user.email) {
          data.email = that.email
        }
        if (isEmptyObj(data)) {
          that.$notify.error({message: "Nothing can update, please at least one field"})
          return
        }

        that.$http.patch("/api/user", data).then(
          (response) => {
            that.$notify.success({message: "Update your profile success"})
            if (that.password !== "") {
              that.$http.delete("/api/logout").then(
                (response) => {
                  user.reset()
                  that.$router.push({name: "Login"})
                },
                notifyErrResponse
              )
            } else {
              user.email = data.email
            }
          },
          notifyErrResponse
        )
      },

      deleteSelf () {
        var that = this
        that.$http.delete('/api/user').then(
          (response) => {
            user.reset()
            that.$router.push({name: "Login"})
          },
          notifyErrResponse
        )
      }
    }
  }
</script>

<style scoped>
</style>
