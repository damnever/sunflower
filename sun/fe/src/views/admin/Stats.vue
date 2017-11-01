<template>
  <div>
    <el-form label-position="left" inline class="form-x">
      <el-form-item label="Server Uptime">
        <span>{{ stats.uptime }}</span>
      </el-form-item>
      <el-form-item label="Current Goroutines">
        <span>{{ stats.num_goroutine }}</span>
      </el-form-item>
      <div class="line"></div>

      <el-form-item label="Current Memory Usage">
        <span>{{ stats.mem_allocated }}</span>
      </el-form-item>
      <el-form-item label="Total Memory Allocated">
        <span>{{ stats.mem_total }}</span>
      </el-form-item>
      <el-form-item label="Memory Obtained">
        <span>{{ stats.mem_sys }}</span>
      </el-form-item>
      <el-form-item label="Total Memory Allocated">
        <span>{{ stats.mem_total }}</span>
      </el-form-item>
      <el-form-item label="Pointer Lookup Times">
        <span>{{ stats.lookups }}</span>
      </el-form-item>
      <el-form-item label="Memory Allocate Times">
        <span>{{ stats.mem_mallocs }}</span>
      </el-form-item>
      <el-form-item label="Memory Free Times">
        <span>{{ stats.mem_frees }}</span>
      </el-form-item>
      <div class="line"></div>

      <el-form-item label="Current Heap Usage">
        <span>{{ stats.heap_alloc }}</span>
      </el-form-item>
      <el-form-item label="Heap Memory Obtained">
        <span>{{ stats.heap_sys }}</span>
      </el-form-item>
      <el-form-item label="Heap Memory Idle">
        <span>{{ stats.heap_idle }}</span>
      </el-form-item>
      <el-form-item label="Heap Memory In Use">
        <span>{{ stats.heap_inuse }}</span>
      </el-form-item>
      <el-form-item label="Heap Memory Released">
        <span>{{ stats.heap_released }}</span>
      </el-form-item>
      <el-form-item label="Heap Objects">
        <span>{{ stats.heap_objects }}</span>
      </el-form-item>
      <div class="line"></div>

      <el-form-item label="Bootstrap Stack Usage">
        <span>{{ stats.stack_inuse }}</span>
      </el-form-item>
      <el-form-item label="Stack Memory Obtained">
        <span>{{ stats.stack_sys }}</span>
      </el-form-item>
      <el-form-item label="MSpan Structures Usage">
        <span>{{ stats.mspan_inuse }}</span>
      </el-form-item>
      <el-form-item label="MSpan Structures Obtained">
        <span>{{ stats.mspan_sys }}</span>
      </el-form-item>
      <el-form-item label="MCache Structures Usage">
        <span>{{ stats.mcache_inuse }}</span>
      </el-form-item>
      <el-form-item label="MCache Structures Obtained">
        <span>{{ stats.mcache_sys }}</span>
      </el-form-item>
      <el-form-item label="Profiling Bucket Hash Table Obtained">
        <span>{{ stats.buck_hash_sys }}</span>
      </el-form-item>
      <el-form-item label="GC Metadata Obtained">
        <span>{{ stats.gc_sys }}</span>
      </el-form-item>
      <el-form-item label="Other System Allocation Obtained">
        <span>{{ stats.other_sys }}</span>
      </el-form-item>
      <div class="line"></div>

      <el-form-item label="Next GC Recycle">
        <span>{{ stats.next_gc }}</span>
      </el-form-item>
      <el-form-item label="Since Last GC Time">
        <span>{{ stats.last_gc }}</span>
      </el-form-item>
      <el-form-item label="Total GC Pause">
        <span>{{ stats.pause_total_ns }}</span>
      </el-form-item>
      <el-form-item label="Last GC Pause">
        <span>{{ stats.pause_ns }}</span>
      </el-form-item>
      <el-form-item label="GC Times">
        <span>{{ stats.num_gc }}</span>
      </el-form-item>
    </el-form>
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
        activeNames: ['1', '2', '3', '4', '5']
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
.form-x {
  font-size: 0;
}
.form-x label {
  width: 200px;
  color: #99a9bf;
}
.form-x .el-form-item {
  margin-right: 0;
  padding: 0 0;
  margin-bottom: 0;
  width: 100%;
}
.line {
  border-top: 1px solid #eaeefb;
  margin-bottom: 2em;
}
</style>
