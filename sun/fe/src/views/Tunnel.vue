<template>
  <div>
    Wow.. Nothing here..
  </div>
</template>

<script>
  import {breadcrumbs} from '../g.js'

  export default {
    data() {
      return {
        ahash: this.$route.params.ahash,
        thash: this.$route.params.thash,
      }
    },
    created () {
      // wow..
      var ahash = this.$route.params.ahash
      var etag = this.$route.params.etag
      var hash = this.$route.params.thash
      var tag = this.$route.params.ttag
      breadcrumbs.splice(0, breadcrumbs.length)
      breadcrumbs.push({route: {name: "Index"}, name: "Index"})
      breadcrumbs.push({
        route: {name: "Agent", params: {ahash: ahash, etag: etag}},
        name: "Agent: [" + etag + "]" + ahash.slice(0, 8),
      })
      breadcrumbs.push({
        route: {name: "Tunnel", params: {ahash: ahash, etag: etag, thash: hash, ttag: tag}},
        name: "Tunnel: [" + tag + "]" + hash.slice(0, 8),
      })
    },
  }
</script>

<style scoped>
</style>
