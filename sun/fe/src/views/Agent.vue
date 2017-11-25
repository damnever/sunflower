<template>
  <div>
    <el-row>
      <el-col :span="6">
        <el-input placeholder="Please input tag" prefix-icon="el-icon-search"
          class="hundred-width" size="small" v-model="qtag">
        </el-input>
      </el-col>
      <el-col :span="2" :offset="15">
        <el-button size="small" icon="el-icon-plus"
          @click="showDialog = true"
          class="eighty-width" round>
        </el-button>
      </el-col>
      <el-col :span="1">
        <el-button size="small" icon="el-icon-refresh"
          @click="fetchTunnels"
          class="hundred-width" round>
        </el-button>
      </el-col>

    </el-row>

    <el-table :data="tunnels" class="hundred-width table-margin-top" v-loading.body="loading">
      <el-table-column type="expand">
        <template slot-scope="props">
          <el-form label-position="left" inline class="table-expand">
            <el-form-item label="Created At">
              <span>{{ props.row.created_at }}</span>
            </el-form-item>
            <el-form-item label="Num Conn">
              <span>{{ props.row.num_conn }}</span>
            </el-form-item>
            <el-form-item label="Traffic In">
              <span>{{ props.row.traffic_in }} (B)</span>
            </el-form-item>
            <el-form-item label="Traffic Out">
              <span>{{ props.row.traffic_out }} (B)</span>
            </el-form-item>
          </el-form>
        </template>
      </el-table-column>
      <el-table-column prop="hash" label="ID" sortable>
      </el-table-column>
      <el-table-column prop="proto" label="Protocol" sortable>
        <template slot-scope="scope">
          <el-tag :type="scope.row.proto === 'TCP' ? 'primary' : 'success'">
            {{ scope.row.proto }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="export_addr" label="LocalAddr" sortable>
      </el-table-column>
      <el-table-column prop="server_addr" label="ServerAddr" sortable>
        <!-- TODO -->
        <template slot-scope="scope">
          <el-tooltip class="item" effect="dark" placement="top-start">
            <div slot="content">
              Copy manually.. {{ scope.row.server_addr.replace("0.0.0.0", config.ip) }}<span v-if="scope.row.proto === 'HTTP'">.{{ config.domain }}</span>
            </div>
            <el-tag type="info">{{ scope.row.server_addr.replace("0.0.0.0", "") }}</el-tag>
          </el-tooltip>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="Status" sortable>
      </el-table-column>
      <el-table-column prop="enabled" label="Enabled">
        <template slot-scope="scope">
          <el-switch v-model="scope.row.enabled"
            @change="enableTunnel(scope.row.hash, scope.row.enabled)">
          </el-switch>
        </template>
      </el-table-column>
      <el-table-column prop="tag" label="Tag" sortable>
      </el-table-column>
      <el-table-column label="Action" fixed="right" width="90px">
        <template slot-scope="scope">
          <el-button @click="deleteTunnel(scope.row)"
            type="danger" size="mini" round>delete
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog title="Create a New Tunnel" :visible.sync="showDialog"
      width="50%" :before-close="closeDialog">
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
  import {breadcrumbs, user, config} from '../g.js'
  import {notifyErrResponse} from '../utils.js'

  export default {
    data() {
      return {
        user: user,
        hash: this.$route.params.ahash,
        tag: this.$route.params.etag,
        tunnels: [],
        config: config,
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
            that.$notify.success({message: "Tunnel(" + hash + ") " + msg})
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
                "num_conn": 0,
                "traffic_in": 0,
                "traffic_out": 0,
                "tag": that.form.tag,
                "created_at": "just now",
              })
              that.$notify.success({message: "Tunnel(" + data.hash + ") created"})
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
