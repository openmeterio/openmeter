import { createPackage } from "@alloy-js/typescript";
import packageJson from "zod/package.json" with { type: "json" };

export const zod = createPackage({
  name: "zod",
  version: packageJson.version,
  descriptor: {
    ".": {
      named: ["z"],
    },
  },
});
