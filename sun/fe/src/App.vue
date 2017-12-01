<template>
  <div>
    <div class="navbar">
      <el-row>
        <el-col :span="14" :offset="3">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{path: '/'}"><b>Sunflower</b></el-breadcrumb-item>
            <el-breadcrumb-item v-for="(item, index) in breadcrumbs" :key="index" @click.native="clickNav(index)" replace>
              {{ item.name }}
            </el-breadcrumb-item>
          </el-breadcrumb>
        </el-col>
        <el-col :span="4">
          <div id="user" v-show="user.name !== ''">
            <el-dropdown @command="clickDropdown">
              <span>
                {{ user.name }} <i class="el-icon-arrow-down el-icon--right"></i>
              </span>
              <el-dropdown-menu slot="dropdown">
                <el-dropdown-item command="Profile">Profile</el-dropdown-item>
                <el-dropdown-item v-show="user.isadmin" command="Admin">Admin</el-dropdown-item>
                <el-dropdown-item v-show="user.isadmin" command="Stats">Stats</el-dropdown-item>
                <el-dropdown-item command="Logout" divided>Logout</el-dropdown-item>
              </el-dropdown-menu>
            </el-dropdown>
          </div>
        </el-col>
      </el-row>
    </div>
    <div class="content">
      <el-col :span="18" :offset="3">
        <router-view></router-view>
      </el-col>
    </div>
  </div>
</template>

<script>
  import {breadcrumbs, user, config} from './g.js'
  import {notifyErrResponse} from './utils.js'

  export default {
    data () {
      return {
        user: user,
        breadcrumbs: breadcrumbs,
      }
    },
    created () {},
    beforeUpdate () {
      this.fetchUserInfo()
      this.fetchConfig()
    },
    watch: {},
    methods: {
      fetchUserInfo () {
        var that = this
        if (that.$router.currentRoute.name === "Login" || user.name !== "") {
          return
        }
        that.$http.get("/api/user").then(
          (response) => {
            response.json().then((data) => {
              user.name = data.name
              user.isadmin = data.is_admin
              user.email = data.email
            }).catch((reason)=>{})
          },
          notifyErrResponse
        )
      },
      fetchConfig () {
        var that = this
        that.$http.get("/api/config").then(
          (response) => {
            response.json().then((data) => {
              config.domain = data.domain
              config.ip = data.ip
            }).catch((reason) => {})
          },
          notifyErrResponse
        )
      },
      clickNav (index) {
        if (index + 1 == this.breadcrumbs.length) {
          return
        }
        var item = this.breadcrumbs[index]
        breadcrumbs.splice(index, breadcrumbs.length-index)
        this.$router.push(item.route)
      },
      clickDropdown (command) {
        switch(command) {
          case 'Logout':
            this.logout()
            break
          default:
            this.$router.push({name: command})
        }
      },
      logout () {
        var that = this
        that.$http.delete("/api/logout").then(
          (response) => {
            user.reset()
            that.$router.push({name: "Login"})
          },
          notifyErrResponse
        )
      },
    }
  }
</script>

<style>
  .content {
    padding: 6.8em 0.4em;
  }

  .navbar {
    position: fixed;
    z-index: 2000;
    padding-top: 1.8em;
    padding-bottom: 1.2em;
    border-bottom: 1px solid rgba(220, 220, 220, 0.4);
    box-shadow: 0 1px 5px rgba(0, 0, 0, 0.12);
    width: 100%;
  }

  #user {
    float: right;
    color: #475669;
    font-size: 1em;
  }

  .el-breadcrumb__item__inner:hover {
    color: #000;
  }

  .el-switch.is-disabled .el-switch__core span {
    background-color: #fff;
  }
  .el-input.is-disabled .el-input__inner {
    color: #aaa;
  }
</style>
