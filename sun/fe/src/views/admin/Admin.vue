<template>
  <div>
    <el-row>
      <el-col :span="6">
        <el-input placeholder="Please input the username"
          prefix-icon="el-icon-search"
          class="hundred-width" size="small"
          v-model="qusername">
        </el-input>
      </el-col>
      <el-col :span="2" :offset="16">
        <el-button size="small" icon="el-icon-plus"
          @click="showDialog = true"
          class="hundred-width" round>
        </el-button>
      </el-col>
    </el-row>

    <el-table :data="users" class="hundred-width table-margin-top" v-loading.body="loading">
      <el-table-column prop="name" label="name" sortable>
      </el-table-column>
      <el-table-column prop="email" label="e-mail" sortable>
      </el-table-column>
      <el-table-column label="admin" sortable>
        <template slot-scope="scope">
          <el-tag :type="scope.row.is_admin ? 'danger' : 'gray'">
            {{ scope.row.is_admin }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="created" sortable>
      </el-table-column>
      <el-table-column label="action">
        <template slot-scope="scope">
          <el-button @click.native.prevent="deleteUser(scope.row)"
            type="text" size="small" :disabled="scope.row.is_admin">delete
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog title="Create a New User" :visible.sync="showDialog"
      width="30%" :before-close="closeCreateUserDialog">
      <el-form :model="form" label-position="right">
        <el-form-item label="Username" label-width="80px">
          <el-input v-model="form.username" auto-complete="off" size="small">
          </el-input>
        </el-form-item>
        <el-form-item label="Password" label-width="80px">
          <el-input v-model="form.password" auto-complete="off" size="small">
          </el-input>
        </el-form-item>
        <el-form-item label="E-mail" label-width="80px">
          <el-input v-model="form.email" auto-complete="off" size="small">
          </el-input>
        </el-form-item>
      </el-form>
      <div slot="footer" class="dialog-footer">
        <el-button @click="closeCreateUserDialog">Cancel</el-button>
        <el-button type="primary" @click="createUser">Create</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
  import {breadcrumbs} from '../../g.js'
  import {notifyErrResponse} from '../../utils.js'

  export default {
    data () {
      return {
        loading: false,
        users: [],
        qusername: "",
        showDialog: false,
        form: {
          username: "",
          password: "",
          email: "",
        },
      }
    },
    created () {
      breadcrumbs.splice(0, breadcrumbs.length)
      breadcrumbs.push({route: {name: "Index"}, name: "Index"})
      breadcrumbs.push({route: {name: "Admin"}, name: "Admin"})
      this.fetchUsers()
    },
    methods: {
      fetchUsers () {
        var that = this
        that.loading = true
        that.$http.get('/api/users').then(
          (response) => {
            response.json().then((data) => {
              that.users = data
            }).catch((reason) => {
            })
            that.loading = false
          },
          (response) => {
            that.loading = false
            notifyErrResponse(response)
          }
        )
      },
      searchUser () {
      },
      createUser () {
        var that = this
        if (that.form.username === "") {
          that.$notify.error({message: "Username can not be empty"})
          return
        }
        if (that.form.email === "") {
          that.$notify.error({message: "E-mail can not be empty"})
          return
        }
        if (that.form.password === "") {
          that.$notify.error({message: "Password can not be empty"})
          return
        }
        that.$http.post('/api/users', that.form).then(
          (response) => {
            that.users.push({
              name: that.form.username,
              email: that.form.email,
              is_admin: false,
              created_at: "just now",
            })
            that.$notify.success({
              message: "Create user(" + that.form.username + ") success"
            })
            that.closeCreateUserDialog()
          },
          notifyErrResponse
        )
      },
      deleteUser (user) {
        var that = this
        that.$http.delete('/api/users/' + user.name).then(
          (response) => {
            var index = that.users.indexOf(user)
            that.users.splice(index, 1)
            that.$notify.success({message: "Delete user(" + user.name + ") success"})
          },
          notifyErrResponse
        )
      },
      closeCreateUserDialog () {
        this.showDialog = false
        this.form = {
          name: "",
          password: "",
          email: "",
        }
      },
    }
  }
</script>

<style scoped>
</style>
