<template>
  <div class="home">
    <img alt="Vue logo" src="../assets/logo.png" />
    <HelloWorld msg="Welcome to Your Vue.js + TypeScript App" />
  </div>
</template>

<script lang="ts">
import Vue from "vue";
import HelloWorld from "@/components/HelloWorld.vue"; // @ is an alias to /src
import { grpc } from "@improbable-eng/grpc-web";
// import { ManagerControl } from "../rpc/protocol/manager_pb_service";
import { ConfigList } from "../rpc/protocol/config_list_pb_service";
import { SetConfigsResponse } from "../rpc/protocol/config_list_pb";
import { Empty } from "google-protobuf/google/protobuf/empty_pb";

export default Vue.extend({
  name: "Home",
  components: {
    HelloWorld,
  },
  mounted() {
    const test = new Empty();
    grpc.unary(ConfigList.GetCurrentConfigs, {
      request: test,
      host: "http://127.0.0.1:41959",
      onEnd: (res) => {
        const { status, statusMessage, headers, message, trailers } = res;
        console.log("onEnd.status", status, statusMessage);
        console.log("onEnd.headers", headers);
        if (status === grpc.Code.OK && message) {
          //   console.log("onEnd.message", message.toObject());
          const resp = SetConfigsResponse.deserializeBinary(
            message.serializeBinary()
          );
          console.log("success", resp.getSuccess());
          console.log("configs", resp.getConfigs());
        }
        console.log("onEnd.trailers", trailers);
      },
    });
  },
});
</script>
