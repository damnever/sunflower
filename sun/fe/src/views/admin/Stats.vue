<template>
  <div class="container">
    <el-button type="text" v-for="name in pprofNames" :key="name">
      <a :href="'/api/sys/debug/pprof/' + name" target="_blank">{{ name }}</a>
    </el-button>
    <el-button type="text">
      <a href="/api/sys/debug/vars" target="_blank">vars</a>
    </el-button>
    <div class="line"></div>

    <el-row class="row">
      <el-col :span="7" class="key">Server Uptime</el-col>
      <el-col :span="5" class="value">{{ stats.uptime }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Current Goroutines</el-col>
      <el-col :span="5" class="value">{{ stats.num_goroutine }}</el-col>
    </el-row>
    <div class="line"></div>

    <el-row class="row">
      <el-col :span="7" class="key">Current Memory Usage</el-col>
      <el-col :span="5" class="value">{{ stats.mem_allocated }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Total Memory Allocated</el-col>
      <el-col :span="5" class="value">{{ stats.mem_total }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Memory Obtained</el-col>
      <el-col :span="5" class="value">{{ stats.mem_sys }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Pointer Lookup Times</el-col>
      <el-col :span="5" class="value">{{ stats.lookups }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Memory Allocate Times</el-col>
      <el-col :span="5" class="value">{{ stats.mem_mallocs }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Memory Free Times</el-col>
      <el-col :span="5" class="value">{{ stats.mem_frees }}</el-col>
    </el-row>
    <div class="line"></div>

    <el-row class="row">
      <el-col :span="7" class="key">Current Heap Usage</el-col>
      <el-col :span="5" class="value">{{ stats.heap_alloc }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Heap Memory Obtained</el-col>
      <el-col :span="5" class="value">{{ stats.heap_sys }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Heap Memory Idle</el-col>
      <el-col :span="5" class="value">{{ stats.heap_idle }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Heap Memory In Use</el-col>
      <el-col :span="5" class="value">{{ stats.heap_inuse }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Heap Memory Released</el-col>
      <el-col :span="5" class="value">{{ stats.heap_released }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Heap Objects</el-col>
      <el-col :span="5" class="value">{{ stats.heap_objects }}</el-col>
    </el-row>
    <div class="line"></div>

    <el-row class="row">
      <el-col :span="7" class="key">Bootstrap Stack Usage</el-col>
      <el-col :span="5" class="value">{{ stats.stack_inuse }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Stack Memory Obtained</el-col>
      <el-col :span="5" class="value">{{ stats.stack_sys }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">MSpan Structures Usage</el-col>
      <el-col :span="5" class="value">{{ stats.mspan_inuse }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">MSpan Structures Obtained</el-col>
      <el-col :span="5" class="value">{{ stats.mspan_sys }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">MCache Structures Usage</el-col>
      <el-col :span="5" class="value">{{ stats.mcache_inuse }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">MCache Structures Obtained</el-col>
      <el-col :span="5" class="value">{{ stats.mcache_sys }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Profiling Bucket Hash Table Obtained</el-col>
      <el-col :span="5" class="value">{{ stats.buck_hash_sys }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">GC Metadata Obtained</el-col>
      <el-col :span="5" class="value">{{ stats.gc_sys }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Other System Allocation Obtained</el-col>
      <el-col :span="5" class="value">{{ stats.other_sys }}</el-col>
    </el-row>
    <div class="line"></div>

    <el-row class="row">
      <el-col :span="7" class="key">Next GC Recycle</el-col>
      <el-col :span="5" class="value">{{ stats.next_gc }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Since Last GC Time</el-col>
      <el-col :span="5" class="value">{{ stats.last_gc }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Total GC Pause</el-col>
      <el-col :span="5" class="value">{{ stats.pause_total_ns }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">Last GC Pause</el-col>
      <el-col :span="5" class="value">{{ stats.pause_ns }}</el-col>
    </el-row>
    <el-row class="row">
      <el-col :span="7" class="key">GC Times</el-col>
      <el-col :span="5" class="value">{{ stats.num_gc }}</el-col>
    </el-row>
    <div class="line"></div>
  </div>
</template>

<script>
  import {breadcrumbs} from '../../g.js'
  import {notifyErrResponse} from '../../utils.js'

  export default {
    data () {
      return {
        loading: false,
        stats: {},
        activeNames: ['1', '2', '3', '4', '5'],
        pprofNames: ["profile", "symbol", "trace", "goroutine", "heap", "block", "threadcreate"]
      }
    },
    created () {
      breadcrumbs.splice(0, breadcrumbs.length)
      breadcrumbs.push({route: {name: "Index"}, name: "Index"})
      breadcrumbs.push({route: {name: "Stats"}, name: "Stats"})
      this.fetchStats()
    },
    methods: {
      fetchStats () {
        var that = this
        that.loading = true
        that.$http.get('/api/sys/stats').then(
          (response) => {
            response.json().then((data) => {
              that.stats = data
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
.row {
  padding: 0 .2em;
  line-height: 1.5em;
}
.key {
  color: #878D99;
}
.value {
  color: #5A5E66;
}
.line {
  border-top: 1px solid #eaeefb;
  margin-top: .5em;
  margin-bottom: 1em;
}
a {
  color: #58B7FF;
  text-decoration: none;
}
a:hover {
  color: #20A0FF;
  text-decoration: underline;
}
</style>
