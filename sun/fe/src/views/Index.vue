<template>
  <div>
    <el-row>
      <el-col :span="6">
        <el-input placeholder="Please input tag" icon="search"
          class="hundred-width" size="small"
          v-model="qtag" :on-icon-click="searchTag">
        </el-input>
      </el-col>
      <el-col :span="2":offset="16">
        <el-button :plain="true" size="small" type="info" icon="plus"
          @click="showAddDialog = true"
          class="hundred-width">
        </el-button>
      </el-col>
    </el-row>

    <el-table :data="agents"  class="hundred-width table-margin-top" v-loading.body="loading">
      <el-table-column prop="hash" label="id" sortable>
      </el-table-column>
      <el-table-column prop="device" label="device" sortable>
      </el-table-column>
      <el-table-column prop="version" label="version" sortable>
      </el-table-column>
      <el-table-column prop="delayed" label="delayed" sortable>
      </el-table-column>
      <el-table-column prop="created_at" label="created" sortable>
      </el-table-column>
      <el-table-column prop="status" label="status" sortable>
      </el-table-column>
      <el-table-column prop="tag" label="tag" sortable>
      </el-table-column>
      <el-table-column label="action" fixed="right" width="200px">
        <template scope="scope">
          <el-button @click="deleteAgent(scope.row)"
            type="text" size="small">delete
          </el-button>
          <el-button @click="preDownloadAgent(scope.row.hash)"
            type="text" size="small">download
          </el-button>
          <el-button  @click="clickHash(scope.row.hash, scope.row.tag)"
            type="text" size="small">tunnels
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog title="Create a New Agent" :visible.sync="showAddDialog"
      size="tiny" :before-close="closeAddDialog">
      <el-form :model="addForm" label-position="right">
        <el-form-item label="Tag" label-width="60px">
          <el-input v-model="addForm.tag" auto-complete="off" size="small"
            placeholder="a good name to distinguish from others">
          </el-input>
        </el-form-item>
      </el-form>
      <div slot="footer" class="dialog-footer">
        <el-button @click="closeAddDialog">Cancel</el-button>
        <el-button type="primary" @click="createAgent">Create</el-button>
      </div>
    </el-dialog>

    <el-dialog :title="'Download Agent #' + dlAgentHash" :visible.sync="showDlDialog"
      size="tiny" :before-close="closeDlDialog">
      <el-form :model="dlForm" label-position="right">
        <el-form-item label="OS" label-width="100px">
          <el-select v-model="dlForm.GOOS" auto-complete="off" size="small">
            <el-option v-for="os in OSs" :key="os.value" :label="os.label" :value="os.value">
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="Arch" label-width="100px">
          <el-select v-model="dlForm.GOARCH" auto-complete="off" size="small">
            <el-option v-for="arch in archs" :key="arch" :label="arch" :value="arch">
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item v-if="arms.length !== 0" label="Arm Version" label-width="100px">
          <el-select v-model="dlForm.GOARM" auto-complete="off" size="small">
            <el-option v-for="arm in arms" :key="arm" :label="arm" :value="arm">
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <div slot="footer" class="dialog-footer">
        <el-button @click="closeDlDialog">Cancel</el-button>
        <el-button type="primary" @click="downloadAgent">Download</el-button>
      </div>
    </el-dialog>

  </div>
</template>

<script>
  import {breadcrumbs} from '../g.js'
  import {notifyErrResponse, toParams} from "../utils.js"

  export default {
    data () {
      return {
        agents: [],
        qtag: "",
        loading: false,
        showAddDialog: false,
        addForm: {
          tag: "",
        },
        showDlDialog: false,
        dlAgentHash: "",
        dlForm: {
          GOOS: "linux",
          GOARCH: "amd64",
          GOARM: "",
        },
        OSs: [
          {label: "Linux", value: "linux"},
          {label: "macOS", value: "darwin"},
          {label: "Windows", value: "windows"},
        ],
        armVersions: ["5", "6", "7"],
      }
    },
    created () {
      breadcrumbs.splice(0, breadcrumbs.length)
      breadcrumbs.push({route: {name: "Index"}, name: "Index"})
      this.fetchAgents()
    },
    computed: {
      archs () {
        if (this.dlForm.GOOS === "linux") {
          return ["amd64", "386", "arm", "arm64"]
        }
        this.dlForm.GOARCH = "amd64"
        return ["amd64", "386"]
      },
      arms () {
        if (this.dlForm.GOARCH !== "arm") {
          return []
        }
        this.dlForm.GOARM = "7"
        return ["5", "6", "7"]
      },
    },
    methods: {
      fetchAgents () {
        var that = this
        that.loading = true
        that.$http.get("/api/user/agents").then(
          (response) => {
            response.json().then((data) => {
              that.agents = data
            }).catch((reason) => {})
            that.loading = false
          },
          (response) => {
            that.loading = false
            notifyErrResponse(response)
          }
        )
      },
      clickHash(hash, tag) {
        this.$router.push({
          name: "Agent",
          params: {ahash: hash, etag: tag},
        })
      },
      searchTag () {
      },
      deleteAgent (agent) {
        var that = this
        that.$http.delete("/api/user/agents/" + agent.hash).then(
          (response) => {
            var index = that.agents.indexOf(agent)
            that.agents.splice(index, 1)
            that.$notify.success({message: "Agent(" + agent.hash + ") deleted"})
          },
          notifyErrResponse
        )
      },
      updateAgent () {},
      createAgent () {
        var that = this
        if (that.addForm.tag === "") {
          that.$notify.error({message: "Tag can not be empty"})
          return
        }
        that.$http.post("/api/user/agents", that.addForm).then(
          (response) => {
            response.json().then((data) => {
              that.agents.push({
                hash: data.hash,
                device: "waiting",
                version: "waiting",
                delayed: "waiting",
                created_at: "just now",
                status: "UNKNOWN",
                enabled: true,
                tag: that.addForm.tag,
              })
              that.$notify.success({message: "Agent created"})
              that.closeAddDialog()
            }).catch((reason) => {})
          },
          notifyErrResponse
        )
      },
      closeAddDialog () {
        this.showAddDialog = false
        this.addForm.tag = ""
      },
      preDownloadAgent (hash) {
        this.showDlDialog = true
        this.dlAgentHash = hash
      },
      downloadAgent () {
        var url = "/api/user/agents/" + this.dlAgentHash +"/bin" + toParams(this.dlForm)
        var win = window.open(url, '_blank')
        win.focus()
        this.closeDlDialog()
      },
      closeDlDialog () {
        this.showDlDialog = false
        this.dlAgentHash = ""
        this.dlForm.GOOS = "linux"
        this.dlForm.GOARCH = "amd64"
        this.dlForm.GOARM = ""
      }
    }
  }
</script>

<style scoped>
</style>
