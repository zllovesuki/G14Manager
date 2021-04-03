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
import { Features } from "../rpc/protocol/features_pb_service";
import {
  Feature,
  UpdateFeaturesRequest,
  FeaturesResponse,
} from "../rpc/protocol/features_pb";
// import { Empty } from "google-protobuf/google/protobuf/empty_pb";

export default Vue.extend({
  name: "Home",
  components: {
    HelloWorld,
  },
  mounted() {
    const feat = new Feature();
    feat.setRogremapList(["Taskmgr.exe", "start Spotify.exe"]);
    const req = new UpdateFeaturesRequest();
    req.setFeature(feat);
    grpc.unary(Features.UpdateFeatures, {
      request: req,
      host: "http://127.0.0.1:41959",
      onEnd: (res) => {
        const { status, statusMessage, headers, message, trailers } = res;
        console.log("onEnd.status", status, statusMessage);
        console.log("onEnd.headers", headers);
        if (status === grpc.Code.OK && message) {
          //   console.log("onEnd.message", message.toObject());
          const resp = FeaturesResponse.deserializeBinary(
            message.serializeBinary()
          );
          console.log("success", resp.getSuccess());
          console.log("rog", resp.getFeature().getRogremapList());
        }
        console.log("onEnd.trailers", trailers);
      },
    });
  },
});
</script>
