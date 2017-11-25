<template>
  <div>
    <span v-for="agent in agents" :key="agent.hash" v-loading.body="loading">
      <h4>Agent #{{ agent.tag }} [{{ agent.hash }}] (delayed {{ agent.delayed }} on {{ agent.device }})</h4>
      <!-- FIXME: dumplicated code - component -->
      <el-table :data="agent.tunnels" class="hundred-width table-margin-top">
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
            <el-switch v-model="scope.row.enabled" disabled>
            </el-switch>
          </template>
        </el-table-column>
        <el-table-column prop="tag" label="Tag" sortable>
        </el-table-column>
      </el-table>
    </span>
  </div>
</template>

<script>
  import {breadcrumbs, config} from '../../g.js'
  import {notifyErrResponse} from '../../utils.js'

  export default {
    data () {
      return {
        loading: false,
        username: this.$route.params.username,
        config: config,
        agents: [],
      }
    },
    created () {
      breadcrumbs.splice(0, breadcrumbs.length)
      breadcrumbs.push({route: {name: "Index"}, name: "Index"})
      breadcrumbs.push({route: {name: "Admin"}, name: "Admin"})
      breadcrumbs.push({route: {name: "Admin/Tunnels"}, name: "User: " + this.username})
      this.fetchAgents()
    },
    methods: {
      fetchAgents () {
        var that = this
        that.loading = true
        that.$http.get('/api/users/' + that.username + '/tunnels').then(
          (response) => {
            response.json().then((data) => {
              that.agents = data
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
    }
  }
</script>

<style scoped>
</style>
