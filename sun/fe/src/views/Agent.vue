<template>
  <div>
    <el-row>
      <el-col :span="6">
        <el-input placeholder="Please input tag" icon="search"
          class="hundred-width" size="small"
          v-model="qtag" :on-icon-click="searchTag">
        </el-input>
      </el-col>
      <el-col :span="2" :offset="16">
        <el-button :plain="true" size="small" type="info" icon="plus"
          @click="showDialog = true"
          class="hundred-width">
        </el-button>
      </el-col>
    </el-row>

    <el-table :data="tunnels" class="hundred-width table-margin-top" v-loading.body="loading">
      <el-table-column prop="hash" label="id" sortable>
      </el-table-column>
      <el-table-column prop="proto" label="protocol" sortable>
        <template scope="scope">
          <el-tag :type="scope.row.proto === 'TCP' ? 'primary' : 'success'">
            {{ scope.row.proto }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="export_addr" label="local address" sortable>
      </el-table-column>
      <el-table-column prop="server_addr" label="server address" sortable>
        <!-- TODO -->
        <template scope="scope">
          <el-tooltip class="item" effect="dark" placement="top-start">
            <div slot="content">
              Copy manually.. {{ scope.row.server_addr.replace("0.0.0.0", config.ip) }}<span v-if="scope.row.proto === 'HTTP'">.{{ config.domain }}</span>
            </div>
            <el-tag type="gray">{{ scope.row.server_addr.replace("0.0.0.0", "") }}</el-tag>
          </el-tooltip>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="created" sortable>
      </el-table-column>
      <el-table-column prop="status" label="status" sortable>
      </el-table-column>
      <el-table-column prop="traffic_in" label="in" sortable>
      </el-table-column>
      <el-table-column prop="traffic_out" label="out" sortable>
      </el-table-column>
      <el-table-column prop="enabled" label="enabled">
        <template scope="scope">
          <el-switch v-model="scope.row.enabled"
            @change="enableTunnel(scope.row.hash, scope.row.enabled)">
          </el-switch>
        </template>
      </el-table-column>
      <el-table-column prop="tag" label="tag" sortable>
      </el-table-column>
      <el-table-column label="action" fixed="right" width="120px">
        <template scope="scope">
          <el-button @click="deleteTunnel(scope.row)"
            type="text" size="small">delete
          </el-button>
          <el-button  @click="clickHash(scope.row.hash, scope.row.tag)"
            type="text" size="small">detail
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog title="Create a New Tunnel" :visible.sync="showDialog"
      size="small" :before-close="closeDialog">
      <el-row>
      <el-col :span="20" :offset="2">
        <el-form :model="form" label-position="right">
          <el-form-item label="Tag" label-width="108px">
            <el-input v-model="form.tag" auto-complete="off" size="small"
              placeholder="a good name to distinguish from others">
            </el-input>
          </el-form-item>
          <el-form-item label="Protocol" label-width="108px">
            <el-select v-model="form.proto" placeholder="select protocol" size="small">
              <el-option v-for="proto in protocols" :key="proto" :label="proto" :value="proto">
              </el-option>
            </el-select>
          </el-form-item>
          <el-form-item label="Local Address" label-width="108px">
            <el-input v-model="form.export_addr" auto-complete="off" size="small"
              placeholder="the local address to export">
              <template slot="prepend">{{ form.proto.toLowerCase() }}://</template>
            </el-input>
          </el-form-item>
          <el-form-item label="Server Address" label-width="108px">
            <el-input v-model="form.server_addr" auto-complete="off" size="small"
              placeholder="port or subdomain which others can access">
              <template slot="prepend">{{ form.proto.toLowerCase() }}://</template>
              <template slot="prepend" v-if="form.proto === 'TCP'">{{ config.ip }}:</template>
              <template slot="append" v-if="form.proto === 'HTTP'">.{{ user.name }}.{{ config.domain }}</template>
            </el-input>
          </el-form-item>
        </el-form>
      </el-col>
      </el-row>
      <div slot="footer" class="dialog-footer">
        <el-button @click="closeDialog">Cancel</el-button>
        <el-button type="primary" @click="createTunnel">Create</el-button>
      </div>
    </el-dialog>

  </div>
</template>

<script>
  import {breadcrumbs, user} from '../g.js'
  import {notifyErrResponse} from '../utils.js'

  export default {
    data() {
      return {
        user: user,
        hash: this.$route.params.ahash,
        tag: this.$route.params.etag,
        tunnels: [],
        config: {},
        qtag: "",
        loading: false,
        showDialog: false,
        form: {
          tag: "",
          proto: "",
          export_addr: "",
          server_addr: "",
        },
        protocols: ["TCP", "HTTP"],
      }
    },
    created () {
      var hash = this.$route.params.ahash
      var tag = this.$route.params.etag
      breadcrumbs.splice(0, breadcrumbs.length)
      breadcrumbs.push({route: {name: "Index"}, name: "Index"})
      breadcrumbs.push({
        route: {name: "Agent", params: {ahash: hash, etag: tag}},
        name: "Agent: [" + tag + "]" + hash,
      })
      this.fetchTunnels()
      this.fetchConfig()
    },
    methods: {
      searchTag () {
      },
      fetchTunnels () {
        var that = this
        that.loading = true
        that.$http.get("/api/user/agents/" + that.hash + "/tunnels").then(
          (response) => {
            response.json().then((data) => {
              that.tunnels = data
            }).catch((reason) => {})
            that.loading = false
          },
          (response) => {
            that.loading = false
            notifyErrResponse(response)
          }
        )
      },
      fetchConfig () {
        var that = this
        that.$http.get("/api/config").then(
          (response) => {
            response.json().then((data) => {
              that.config = data
            }).catch((reason) => {})
          },
          notifyErrResponse
        )
      },
      clickHash (hash, tag) {
        this.$router.push({
          name: "Tunnel",
          params: {ahash: this.hash, etag: this.tag, thash: hash, ttag: tag},
        })
      },
      deleteTunnel (tunnel) {
        var that = this
        that.$http.delete("/api/user/agents/" + that.hash + "/tunnels/" + tunnel.hash).then(
          (response) => {
            var index = that.tunnels.indexOf(tunnel)
            that.tunnels.splice(index, 1)
            that.$notify.success({message: "Tunnel(" + tunnel.hash + ") deleted"})
          },
          notifyErrResponse
        )
      },
      enableTunnel (hash, enabled) {
        var that = this
        var data = {enabled: enabled}
        that.$http.patch("/api/user/agents/" + that.hash + "/tunnels/" + hash, data).then(
          (response) => {
            var msg = {true: "enabled", false: "disabled"}[enabled]
            that.$notify.success({message: "Tunnel (" + hash + ")" + msg})
          },
          notifyErrResponse
        )
      },
      createTunnel () {
        var that = this
        if (that.tag === "") {
          that.$notify.error({message: "Tag can not be empty"})
          return
        }
        if (that.proto === "") {
          that.$notify.error({message: "Protocol can not be empty"})
          return
        }
        if (that.export_addr === "") {
          that.$notify.error({message: "Local address can not be empty"})
          return
        }
        if (that.server_addr === "") {
          that.$notify.error({message: "Server address can not be empty"})
          return
        }

        that.$http.post("/api/user/agents/" + that.hash + "/tunnels", that.form).then(
          (response) => {
            response.json().then((data) =>{
              var saddr = that.form.server_addr
              if (that.form.proto === "HTTP") {
                saddr = saddr + "." + user.name
              } else {
                saddr = "0.0.0.0:" + saddr
              }
              that.tunnels.push({
                "hash": data.hash,
                "proto": that.form.proto,
                "export_addr": that.form.export_addr,
                "server_addr": saddr,
                "status": "PENDING",
                "enabled": true,
                "traffic_in": 0,
                "traffic_out": 0,
                "tag": that.form.tag,
                "created_at": "just now",
              })
              that.$notify.success({message: "Tunnel created"})
              that.closeDialog()
            }).catch((reason) => {})
          },
          notifyErrResponse
        )
      },
      closeDialog () {
        this.showDialog = false
        this.form = {
          tag: "",
          proto: "",
          export_addr: "",
          server_addr: "",
        }
      }
    }
  }
</script>

<style scoped>
</style>
